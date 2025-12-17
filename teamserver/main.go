package main

import(
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
	"simplec2/pkg/logger"
	"simplec2/teamserver/api"
	"simplec2/teamserver/data"
	"simplec2/teamserver/service"
	"simplec2/teamserver/websocket"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

var cfg config.TeamServerConfig

func main() {
	// Initialize structured logger (zap)
	if err := logger.Init("info"); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	configPath := flag.String("config", "teamserver.yaml", "Path to the TeamServer configuration file.")
	hashPassword := flag.Bool("hash-password", false, "Hash the operator password from the config file and exit.")
	flag.Parse()

	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		logger.Infof("Configuration file not found. Generating a default one at '%s'", *configPath)
		if err := generateDefaultConfig(*configPath); err != nil {
			logger.Fatalf("Failed to generate default config: %v", err)
		}
		logger.Info("Please review and edit the new configuration file, then restart the server.")
		return
	}

	if err := config.LoadConfig(*configPath, &cfg); err != nil {
		logger.Fatalf("Failed to load configuration: %v", err)
	}
	logger.Info("Configuration loaded successfully.")

	if *hashPassword {
		if cfg.Auth.OperatorPassword == "" {
			logger.Fatal("Operator password is not set in the configuration file.")
		}
		hashedPassword, err := api.HashPassword(cfg.Auth.OperatorPassword)
		if err != nil {
			logger.Fatalf("Failed to hash password: %v", err)
		}
		fmt.Printf("Hashed password for your config file (replace operator_password with this value):\n%s\n", hashedPassword)
		return
	}

	// Initialize the DataStore
	store, err := data.NewDataStore(cfg.Database)
	if err != nil {
		logger.Fatalf("Failed to initialize data store: %v", err)
	}
	logger.Info("Database initialized successfully.")

	// Initialize services
	beaconService := service.NewBeaconService(store)
	taskService := service.NewTaskService(store)
	listenerService := service.NewListenerService(store)
	sessionService := service.NewSessionService(store)
	portFwdService := service.NewInMemoryPortFwdService() // Instantiate PortFwdService

	// Start session cleanup routine (run every 5 minutes)
	sessionService.StartCleanupRoutine(5 * time.Minute)
	logger.Info("Session cleanup routine started (runs every 5 minutes)")

	// Create and run the WebSocket hub
	hub := websocket.NewHub()
	go hub.Run()

	creds, err := loadTeamServerCreds(cfg.GRPC.Certs.ServerCert, cfg.GRPC.Certs.ServerKey, cfg.GRPC.Certs.CACert, func(serialNumber string) bool {
		return listenerService.IsCertificateRevoked(serialNumber)
	})
	if err != nil {
		logger.Fatalf("Failed to load TLS credentials: %v", err)
	}

	// 获取 API Key（优先使用加密版本）
	apiKey, err := cfg.Auth.GetAPIKey()
	if err != nil {
		logger.Fatalf("Failed to get API key: %v", err)
	}

	// Correctly create the auth interceptor
	interceptor := NewAuthInterceptor(apiKey)

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(interceptor),
		grpc.MaxRecvMsgSize(100*1024*1024), // 100 MB
	)

	// Correctly create an instance of the server struct with config, store, and hub
	s := NewServer(&cfg, store, hub, listenerService, portFwdService) // Pass portFwdService to NewServer
	// Correctly call the registration function with the package prefix
	bridge.RegisterTeamServerBridgeServiceServer(grpcServer, s)

	go func() {
		lis, err := net.Listen("tcp", cfg.GRPC.Port)
		if err != nil {
			logger.Fatalf("Failed to listen on gRPC port: %v", err)
		}
		logger.Infof("gRPC server listening on %s", cfg.GRPC.Port)
		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	go func() {
		router := api.NewRouter(&cfg, beaconService, taskService, listenerService, sessionService, portFwdService, hub) // Pass portFwdService to NewRouter
		logger.Infof("HTTP API server listening on %s", cfg.API.Port)
		if err := router.Run(cfg.API.Port); err != nil {
			logger.Fatalf("Failed to run HTTP server: %v", err)
		}
	}()

	select {}
}

func generateDefaultConfig(path string) error {
	defaultConfig := config.TeamServerConfig{
		GRPC: struct {
			Port  string `yaml:"port"`
			Certs struct {
				ServerCert string `yaml:"server_cert"`
				ServerKey  string `yaml:"server_key"`
				CACert     string `yaml:"ca_cert"`
			} `yaml:"certs"`
		}{
			Port: ":50052",
			Certs: struct {
				ServerCert string "yaml:\"server_cert\""
				ServerKey  string "yaml:\"server_key\""
				CACert     string "yaml:\"ca_cert\""
			}{
				ServerCert: "./certs/server.crt",
				ServerKey:  "./certs/server.key",
				CACert:     "./certs/ca.crt",
			},
		},
		API: struct {
			Port string `yaml:"port"`
		}{
			Port: ":8080",
		},
		Database: struct {
			Type string `yaml:"type"`
			DSN  string `yaml:"dsn,omitempty"`
			Path string `yaml:"path,omitempty"`
		}{
			Type: "sqlite",
			Path: "data/simplec2.db",
		},
		Auth: config.AuthConfig{
			APIKey:           "SimpleC2ListenerAPIKey_CHANGE_ME",
			OperatorPassword: "SUPER_SECRET_PASSWORD_CHANGE_ME",
			// JWT 签名密钥 - 强烈建议从环境变量读取
			// 用于签发操作员认证令牌，必须保持机密
			JWTSecret: "CHANGE_ME_TO_RANDOM_256_BIT_KEY",
		},
		LootDir:    "loot",
		UploadsDir: "uploads",
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func loadTeamServerCreds(serverCert, serverKey, caCert string, checkRevocation func(serialNumber string) bool) (credentials.TransportCredentials, error) {
	serverC, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		return nil, err
	}

	ca, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("failed to append CA cert")
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{serverC},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if len(verifiedChains) == 0 || len(verifiedChains[0]) == 0 {
				return fmt.Errorf("no verified certificate chain found")
			}
			cert := verifiedChains[0][0] // Leaf certificate
			serialNumber := cert.SerialNumber.String()
			
			logger.Infof("Verifying certificate for connection. Serial: %s, CN: %s", serialNumber, cert.Subject.CommonName)

			if checkRevocation(serialNumber) {
				logger.Warnf("Rejected connection from revoked certificate: %s (CN: %s)", serialNumber, cert.Subject.CommonName)
				return fmt.Errorf("certificate revoked")
			}
			return nil
		},
	}

	return credentials.NewTLS(config), nil
}
