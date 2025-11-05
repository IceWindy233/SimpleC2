package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

// CertConfig holds the configuration for a certificate
type CertConfig struct {
	FileName string
	Host     string
	IsCA     bool
	IsServer bool
	IsClient bool
}

func main() {
	fmt.Println("--- Generating All Cryptographic Materials ---")

	// Define paths
	teamserverCertDir := "certs/teamserver"
	listenerCertDir := "certs/listener"
	agentKeyDir := "certs/agent"

	// Create directories
	for _, dir := range []string{teamserverCertDir, listenerCertDir, agentKeyDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Part 1: Generate RSA keys for E2E Encryption
	fmt.Println("\n[1/2] Generating RSA key pair for E2E encryption...")
	generateRSAKeys(listenerCertDir, agentKeyDir)

	// Part 2: Generate certificates for mTLS
	fmt.Println("\n[2/2] Generating certificates for mTLS...")
	generateMTLSCertificates(teamserverCertDir, listenerCertDir)

	fmt.Println("\n--- All materials generated successfully! ---")
}

func generateRSAKeys(listenerDir, agentDir string) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA private key: %v", err)
	}

	// Save private key
	privateKeyPath := filepath.Join(listenerDir, "listener_rsa.key")
	savePEMKey(privateKeyPath, x509.MarshalPKCS1PrivateKey(privateKey), "RSA PRIVATE KEY", 0600)
	fmt.Printf("  -> Saved RSA private key to %s\n", privateKeyPath)

	// Save public key for listener
	publicKey := &privateKey.PublicKey
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key: %v", err)
	}
	publicKeyPath := filepath.Join(listenerDir, "listener_rsa.pub")
	savePEMKey(publicKeyPath, publicKeyBytes, "PUBLIC KEY", 0644)
	fmt.Printf("  -> Saved RSA public key to %s\n", publicKeyPath)

	// Save public key for agent embedding
	agentPublicKeyPath := filepath.Join(agentDir, "listener.pub")
	savePEMKey(agentPublicKeyPath, publicKeyBytes, "PUBLIC KEY", 0644)
	fmt.Printf("  -> Saved agent public key to %s\n", agentPublicKeyPath)
}

func generateMTLSCertificates(tsDir, listenerDir string) {
	// 1. Generate CA
	caKey, caCert, err := generateCertificate(CertConfig{IsCA: true, Host: "SimpleC2 CA"}, nil, nil)
	if err != nil {
		log.Fatalf("Failed to generate CA: %v", err)
	}
	caCertPath := filepath.Join(tsDir, "ca.crt")
	caKeyPath := filepath.Join(tsDir, "ca.key")
	saveCertificate(caCertPath, caCert.Raw)
	savePEMKey(caKeyPath, marshalECPrivateKey(caKey), "EC PRIVATE KEY", 0600)
	fmt.Println("  -> Generated self-signed CA.")

	// Copy CA cert to listener directory for trust
	listenerCaCertPath := filepath.Join(listenerDir, "ca.crt")
	saveCertificate(listenerCaCertPath, caCert.Raw)
	fmt.Printf("  -> Saved CA certificate to %s and %s\n", caCertPath, listenerCaCertPath)

	// 2. Generate TeamServer Certificate (Server)
	serverKey, serverCert, err := generateCertificate(CertConfig{IsServer: true, Host: "localhost"}, caCert, caKey)
	if err != nil {
		log.Fatalf("Failed to generate server certificate: %v", err)
	}
	serverCertPath := filepath.Join(tsDir, "server.crt")
	serverKeyPath := filepath.Join(tsDir, "server.key")
	saveCertificate(serverCertPath, serverCert.Raw)
	savePEMKey(serverKeyPath, marshalECPrivateKey(serverKey), "EC PRIVATE KEY", 0600)
	fmt.Printf("  -> Generated server certificate, signed by CA, saved in %s\n", tsDir)

	// 3. Generate Listener Certificate (Client)
	clientKey, clientCert, err := generateCertificate(CertConfig{IsClient: true, Host: "SimpleC2 Listener"}, caCert, caKey)
	if err != nil {
		log.Fatalf("Failed to generate client certificate: %v", err)
	}
	clientCertPath := filepath.Join(listenerDir, "client.crt")
	clientKeyPath := filepath.Join(listenerDir, "client.key")
	saveCertificate(clientCertPath, clientCert.Raw)
	savePEMKey(clientKeyPath, marshalECPrivateKey(clientKey), "EC PRIVATE KEY", 0600)
	fmt.Printf("  -> Generated client certificate, signed by CA, saved in %s\n", listenerDir)
}

// --- Helper Functions ---

func generateCertificate(config CertConfig, parentCert *x509.Certificate, parentKey *ecdsa.PrivateKey) (*ecdsa.PrivateKey, *x509.Certificate, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"SimpleC2"}, CommonName: config.Host},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	if config.IsCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	} else {
		if config.IsServer {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
			template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
			template.DNSNames = []string{"localhost"}
		}
		if config.IsClient {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}
	}

	var derBytes []byte
	if config.IsCA {
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	} else {
		derBytes, err = x509.CreateCertificate(rand.Reader, &template, parentCert, &privateKey.PublicKey, parentKey)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	return privateKey, cert, err
}

func savePEMKey(path string, keyBytes []byte, keyType string, perm os.FileMode) {
	keyOut, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		log.Fatalf("Failed to open %s for writing: %v", path, err)
	}
	defer keyOut.Close()
	pem.Encode(keyOut, &pem.Block{Type: keyType, Bytes: keyBytes})
}

func saveCertificate(path string, certBytes []byte) {
	certOut, err := os.Create(path)
	if err != nil {
		log.Fatalf("Failed to open %s for writing: %v", path, err)
	}
	defer certOut.Close()
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
}

func marshalECPrivateKey(key *ecdsa.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		log.Fatalf("Unable to marshal ECDSA private key: %v", err)
	}
	return b
}
