package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// EncryptedAPIKey 存储加密后的 API Key
type EncryptedAPIKey struct {
	// 加密后的 API Key（Base64 编码）
	Encrypted string `yaml:"encrypted"`
	// 随机数 (Nonce) 用于 AES-GCM
	Nonce string `yaml:"nonce"`
}

// EncryptAPIKey 使用 AES-256-GCM 加密 API Key
func EncryptAPIKey(apiKey string) (*EncryptedAPIKey, error) {
	// 从环境变量获取加密密钥
	encryptionKey := os.Getenv("SIMC2_ENCRYPTION_KEY")
	if encryptionKey == "" {
		// 如果没有环境变量，生成一个临时的（生产环境应该避免）
		encryptionKey = "dev-encryption-key-change-me"
	}

	// 将密钥转换为 32 字节 (AES-256)
	key := sha256.Sum256([]byte(encryptionKey))

	// 创建 AES-GCM cipher
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密 API Key
	ciphertext := gcm.Seal(nonce, nonce, []byte(apiKey), nil)

	return &EncryptedAPIKey{
		Encrypted: base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:     base64.StdEncoding.EncodeToString(nonce),
	}, nil
}

// DecryptAPIKey 解密 API Key
func (e *EncryptedAPIKey) DecryptAPIKey() (string, error) {
	if e == nil {
		return "", fmt.Errorf("encrypted API key is nil")
	}

	// 从环境变量获取解密密钥
	encryptionKey := os.Getenv("SIMC2_ENCRYPTION_KEY")
	if encryptionKey == "" {
		encryptionKey = "dev-encryption-key-change-me"
	}

	// 将密钥转换为 32 字节
	key := sha256.Sum256([]byte(encryptionKey))

	// 创建 AES-GCM cipher
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 解码 nonce
	nonce, err := base64.StdEncoding.DecodeString(e.Nonce)
	if err != nil {
		return "", err
	}

	// 解码加密数据
	ciphertext, err := base64.StdEncoding.DecodeString(e.Encrypted)
	if err != nil {
		return "", err
	}

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// String 方法实现 yaml.Marshaler 接口
func (e *EncryptedAPIKey) String() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("EncryptedAPIKey{Nonce: %s, Encrypted: %s}", e.Nonce[:16], e.Encrypted[:16])
}

// getJWTSecret 获取 JWT 签名密钥，优先从环境变量读取
func GetJWTSecret(configSecret string) string {
	// 1. 优先从环境变量读取
	if secret := os.Getenv("SIMC2_JWT_SECRET"); secret != "" {
		return secret
	}
	// 2. 回退到配置中的密钥
	if configSecret != "" {
		return configSecret
	}
	// 3. 如果都没有，使用默认密钥（应该避免在生产环境使用）
	return "dev-default-jwt-secret-change-in-production"
}

// TeamServerConfig holds all configuration for the TeamServer.
type TeamServerConfig struct {
	GRPC struct {
		Port    string `yaml:"port"`
		Certs struct {
			ServerCert string `yaml:"server_cert"`
			ServerKey  string `yaml:"server_key"`
			CACert     string `yaml:"ca_cert"`
		} `yaml:"certs"`
	} `yaml:"grpc"`
	API struct {
		Port string `yaml:"port"`
	} `yaml:"api"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	LootDir  string         `yaml:"loot_dir"`
	UploadsDir string       `yaml:"uploads_dir"`
}

// DatabaseConfig holds database-specific configuration.
type DatabaseConfig struct {
	Type string `yaml:"type"`           // "postgres" or "sqlite"
	DSN  string `yaml:"dsn,omitempty"`  // Optional: For Postgres
	Path string `yaml:"path,omitempty"` // Optional: For SQLite
}

// AuthConfig holds authentication-related configuration.
type AuthConfig struct {
	// 注意：为了向后兼容保留 APIKey 字段，但生产环境应该使用 EncryptedAPIKey
	APIKey string `yaml:"api_key,omitempty"`
	// 加密的 API Key - 推荐在生产环境中使用
	EncryptedAPIKey *EncryptedAPIKey `yaml:"encrypted_api_key,omitempty"`
	OperatorPassword string `yaml:"operator_password"`
	// JWT 签名密钥 - 应该从环境变量或独立的密钥文件读取
	JWTSecret string `yaml:"jwt_secret,omitempty"`
}

// GetAPIKey 获取解密后的 API Key，优先使用加密版本
func (a *AuthConfig) GetAPIKey() (string, error) {
	// 优先使用加密的 API Key
	if a.EncryptedAPIKey != nil {
		return a.EncryptedAPIKey.DecryptAPIKey()
	}
	// 回退到明文版本（仅用于向后兼容）
	return a.APIKey, nil
}

// MustGetAPIKey 获取 API Key，如果失败则 panic
func (a *AuthConfig) MustGetAPIKey() string {
	apiKey, err := a.GetAPIKey()
	if err != nil {
		panic(err)
	}
	return apiKey
}

// GetAPIKey 获取 Listener 的解密后 API Key，优先使用加密版本
func (l *ListenerConfig) GetAPIKey() (string, error) {
	// 优先使用加密的 API Key
	if l.Auth.EncryptedAPIKey != nil {
		return l.Auth.EncryptedAPIKey.DecryptAPIKey()
	}
	// 回退到明文版本（仅用于向后兼容）
	return l.Auth.APIKey, nil
}

// MustGetAPIKey 获取 Listener 的 API Key，如果失败则 panic
func (l *ListenerConfig) MustGetAPIKey() string {
	apiKey, err := l.GetAPIKey()
	if err != nil {
		panic(err)
	}
	return apiKey
}

// ListenerConfig holds all configuration for the Listener.
type ListenerConfig struct {
	TeamServer struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"teamserver"`
	Listener struct {
		Name string `yaml:"name"`
		Port string `yaml:"port"`
	} `yaml:"listener"`
	Auth struct {
		// 注意：为了向后兼容保留 APIKey 字段，但生产环境应该使用 EncryptedAPIKey
		APIKey string `yaml:"api_key,omitempty"`
		// 加密的 API Key - 推荐在生产环境中使用
		EncryptedAPIKey *EncryptedAPIKey `yaml:"encrypted_api_key,omitempty"`
	} `yaml:"auth"`
	Certs struct {
		ClientCert string `yaml:"client_cert"`
		ClientKey  string `yaml:"client_key"`
		CACert     string `yaml:"ca_cert"`
		PrivateKey string `yaml:"private_key"`
	} `yaml:"certs"`
}

// LoadConfig reads a YAML file from the given path and unmarshals it into the provided config struct.
func LoadConfig(path string, config interface{}) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(file, config)
}
