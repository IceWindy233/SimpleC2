package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"simplec2/pkg/config"
)

// HashPassword 使用 bcrypt 哈希密码
func HashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hashed), err
}

// verifyPassword 验证密码和哈希
func verifyPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// AuthRequest defines the structure for the login request body.
type AuthRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles operator authentication and JWT issuance
func (a *API) Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AuthRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		storedPassword := a.Config.Auth.OperatorPassword
		isHashed := strings.HasPrefix(storedPassword, "$2a$") || strings.HasPrefix(storedPassword, "$2b$") || strings.HasPrefix(storedPassword, "$2y$")

		// 密码验证
		if isHashed {
			// 存储的是哈希，使用 bcrypt 比较
			if err := verifyPassword(req.Password, storedPassword); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				return
			}
		} else {
			// 存储的是明文，直接比较（不安全）
			log.Println("Warning: The operator password is in plaintext. Please use the -hash-password flag to generate a hash and update your config file for better security.")
			if req.Password != storedPassword {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
				return
			}
		}

		// 获取独立的 JWT 签名密钥
		jwtSecret := config.GetJWTSecret(a.Config.Auth.JWTSecret)

		// 创建 JWT token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": req.Username,
			"iat": time.Now().Unix(),
			"exp": time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
		})

		// 使用独立的 JWT 密钥签名
		tokenString, err := token.SignedString([]byte(jwtSecret))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString})
	}
}

// AuthMiddleware creates a middleware handler for JWT validation.
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is invalid"})
			return
		}

		tokenString := parts[1]
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrAbortHandler
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		c.Next()
	}
}
