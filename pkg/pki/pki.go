package pki

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// CertConfig holds the configuration for a certificate
type CertConfig struct {
	CommonName string
	IsCA       bool
	IsServer   bool
	IsClient   bool
	DNSNames   []string
	IPs        []net.IP
}

// GenerateRSAKeyPair generates an RSA 2048-bit key pair.
// Returns private key PEM and public key PEM.
func GenerateRSAKeyPair() ([]byte, []byte, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA private key: %w", err)
	}

	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes})

	publicKey := &privateKey.PublicKey
	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})

	return privPEM, pubPEM, nil
}

// GenerateCert generates a certificate signed by a parent CA (or self-signed if parent is nil).
// Returns private key PEM and certificate PEM.
func GenerateCert(cfg CertConfig, parentCertPEM, parentKeyPEM []byte) ([]byte, []byte, error) {
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
		Subject:               pkix.Name{Organization: []string{"SimpleC2"}, CommonName: cfg.CommonName},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		DNSNames:              cfg.DNSNames,
		IPAddresses:           cfg.IPs,
	}

	if cfg.IsCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	} else {
		if cfg.IsServer {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		}
		if cfg.IsClient {
			template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
		}
	}

	var parentCert *x509.Certificate
	var parentKey interface{}

	if parentCertPEM != nil && parentKeyPEM != nil {
		block, _ := pem.Decode(parentCertPEM)
		if block == nil {
			return nil, nil, fmt.Errorf("failed to decode parent cert PEM")
		}
		parentCert, err = x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse parent cert: %w", err)
		}

		blockKey, _ := pem.Decode(parentKeyPEM)
		if blockKey == nil {
			return nil, nil, fmt.Errorf("failed to decode parent key PEM")
		}
		// Assuming ECDSA key for now as per our standard
		parentKey, err = x509.ParseECPrivateKey(blockKey.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse parent private key: %w", err)
		}
	} else {
		// Self-signed
		parentCert = &template
		parentKey = privateKey
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, parentCert, &privateKey.PublicKey, parentKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	privBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to marshal ECDSA private key: %v", err)
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})

	return privPEM, certPEM, nil
}

// SavePEMFile saves PEM encoded data to a file.
func SavePEMFile(path string, pemData []byte, perm os.FileMode) error {
	return os.WriteFile(path, pemData, perm)
}
