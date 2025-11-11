package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
	"simplec2/teamserver/api"
	"simplec2/teamserver/data"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v3"
)

var cfg config.TeamServerConfig

func main() {
	configPath := flag.String("config", "teamserver.yaml", "Path to the TeamServer configuration file.")
	hashPassword := flag.Bool("hash-password", false, "Hash the operator password from the config file and exit.")
	flag.Parse()

	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		log.Printf("Configuration file not found. Generating a default one at '%s'", *configPath)
		if err := generateDefaultConfig(*configPath); err != nil {
			log.Fatalf("Failed to generate default config: %v", err)
		}
		log.Println("Please review and edit the new configuration file, then restart the server.")
		return
	}

	if err := config.LoadConfig(*configPath, &cfg); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Println("Configuration loaded successfully.")

	if *hashPassword {
		if cfg.Auth.OperatorPassword == "" {
			log.Fatal("Operator password is not set in the configuration file.")
		}
		hashedPassword, err := api.HashPassword(cfg.Auth.OperatorPassword)
		if err != nil {
			log.Fatalf("Failed to hash password: %v", err)
		}
		fmt.Printf("Hashed password for your config file (replace operator_password with this value):\n%s\n", hashedPassword)
		return
	}

	// Initialize the DataStore
	store, err := data.NewDataStore(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize data store: %v", err)
	}
	log.Println("Database initialized successfully.")

	creds, err := loadTeamServerCreds(cfg.GRPC.Certs.ServerCert, cfg.GRPC.Certs.ServerKey, cfg.GRPC.Certs.CACert)
	if err != nil {
		log.Fatalf("Failed to load TLS credentials: %v", err)
	}

	// 获取 API Key（优先使用加密版本）
	apiKey, err := cfg.Auth.GetAPIKey()
	if err != nil {
		log.Fatalf("Failed to get API key: %v", err)
	}

	// Correctly create the auth interceptor
	interceptor := NewAuthInterceptor(apiKey)

	grpcServer := grpc.NewServer(
		grpc.Creds(creds),
		grpc.UnaryInterceptor(interceptor),
		grpc.MaxRecvMsgSize(100*1024*1024), // 100 MB
	)

	// Correctly create an instance of the server struct with config and store
	s := NewServer(&cfg, store)
	// Correctly call the registration function with the package prefix
	bridge.RegisterTeamServerBridgeServiceServer(grpcServer, s)

	go func() {
		lis, err := net.Listen("tcp", cfg.GRPC.Port)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC port: %v", err)
		}
		log.Printf("gRPC server listening on %s", cfg.GRPC.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	go func() {
		router := api.NewRouter(&cfg, store)
		log.Printf("HTTP API server listening on %s", cfg.API.Port)
		if err := router.Run(cfg.API.Port); err != nil {
			log.Fatalf("Failed to run HTTP server: %v", err)
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
			// 独立的 JWT 签名密钥 - 强烈建议从环境变量读取
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

func loadTeamServerCreds(serverCert, serverKey, caCert string) (credentials.TransportCredentials, error) {
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
	}

	return credentials.NewTLS(config), nil
}
