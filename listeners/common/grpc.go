package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"simplec2/pkg/bridge"
	"simplec2/pkg/config"
)

var TSClient bridge.TeamServerBridgeServiceClient

// IsNotFound checks if an error is a gRPC status error with the code NotFound.
func IsNotFound(err error) bool {
	s, ok := status.FromError(err)
	if !ok {
		return false
	}
	return s.Code() == codes.NotFound
}

// ConnectToTeamServer establishes a secure mTLS connection to the TeamServer.
func ConnectToTeamServer(cfg *config.ListenerConfig) (*grpc.ClientConn, error) {
	// Load client's certificate and private key
	clientCert, err := tls.LoadX509KeyPair(cfg.Certs.ClientCert, cfg.Certs.ClientKey)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate and key: %w", err)
	}

	// Load CA's certificate
	caCert, err := os.ReadFile(cfg.Certs.CACert)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS configuration
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caCertPool,
		ServerName:   cfg.TeamServer.Host,
	}

	// Create gRPC client with TLS credentials
	creds := credentials.NewTLS(tlsConfig)
	teamserverAddr := fmt.Sprintf("%s%s", cfg.TeamServer.Host, cfg.TeamServer.Port)
	conn, err := grpc.NewClient(teamserverAddr, grpc.WithTransportCredentials(creds),
		grpc.WithBlock(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(100*1024*1024), // 100 MB
			grpc.MaxCallRecvMsgSize(100*1024*1024), // 100 MB
		),
	)
	if err != nil {
		return nil, fmt.Errorf("did not connect to teamserver: %w", err)
	}

	TSClient = bridge.NewTeamServerBridgeServiceClient(conn)
	log.Printf("Successfully connected to TeamServer gRPC with mTLS at %s", teamserverAddr)
	return conn, nil
}

// CreateAuthenticatedContext creates a new context with the API key attached for gRPC calls.
func CreateAuthenticatedContext(cfg *config.ListenerConfig) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// 获取 API Key（优先使用加密版本）
	apiKey, err := cfg.GetAPIKey()
	if err != nil {
		log.Printf("Warning: Failed to get API key: %v", err)
		// 使用明文版本作为回退
		apiKey = cfg.Auth.APIKey
	}
	md := metadata.New(map[string]string{"authorization": "Bearer " + apiKey})
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, cancel
}
