package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
)

func main() {
	// Generate a new 2048-bit RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// --- Save Private Key ---

	// Encode the private key to the PEM format
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyFile, err := os.Create("listeners/http/certs/listener_rsa.key")
	if err != nil {
		log.Fatalf("Failed to create private key file: %v", err)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		log.Fatalf("Failed to write private key to file: %v", err)
	}
	log.Println("Successfully generated and saved private key to listeners/http/certs/listener_rsa.key")

	// --- Save Public Key ---

	// Get the public key from the private key
	publicKey := &privateKey.PublicKey

	// Marshal the public key to PKIX format
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		log.Fatalf("Failed to marshal public key: %v", err)
	}

	// Encode the public key to the PEM format
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.Create("listeners/http/certs/listener_rsa.pub")
	if err != nil {
		log.Fatalf("Failed to create public key file: %v", err)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		log.Fatalf("Failed to write public key to file: %v", err)
	}
	log.Println("Successfully generated and saved public key to listeners/http/certs/listener_rsa.pub")

	// --- Also save a copy for the agent to embed ---
	agentPublicKeyFile, err := os.Create("agents/http/listener.pub")
	if err != nil {
		log.Fatalf("Failed to create agent public key file: %v", err)
	}
	defer agentPublicKeyFile.Close()

	if err := pem.Encode(agentPublicKeyFile, publicKeyPEM); err != nil {
		log.Fatalf("Failed to write public key to agent file: %v", err)
	}
	log.Println("Successfully saved public key copy to agents/http/listener.pub")
}
