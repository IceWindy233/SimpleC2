package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"simplec2/pkg/config"
	"simplec2/pkg/logger"
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
			logger.Warn("The operator password is in plaintext. Please use the -hash-password flag to generate a hash and update your config file for better security.")
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

		// Create session record
		if a.SessionService != nil {
			_, err := a.SessionService.CreateSession(req.Username, tokenString, c.ClientIP(), c.Request.UserAgent(), 24*time.Hour)
			if err != nil {
				logger.Warnf("Failed to create session for user %s: %v", req.Username, err)
				// Continue anyway, session creation failure shouldn't block login
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"token": tokenString,
			"expires_at": time.Now().Add(time.Hour * 24).Unix(),
		})
	}
}

// AuthMiddlewareWithSession creates a middleware handler for JWT and session validation.
func (a *API) AuthMiddlewareWithSession(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record request start time for duration calculation in audit log
		c.Set("requestStartTime", time.Now())

		var tokenString string

		// For WebSockets, the token is passed as a query parameter
		// because headers are not easily sent.
		if c.Request.URL.Path == "/api/ws" {
			tokenString = c.Query("token")
			if tokenString == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "WebSocket token is missing"})
				return
			}
		} else {
			// For standard HTTP requests, get the token from the header.
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
			tokenString = parts[1]
		}

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

		// Validate session
		if a.SessionService != nil {
			_, valid := a.SessionService.ValidateSession(tokenString)
			if !valid {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired or invalid"})
				return
			}
		}

		// Store the token claims in the context for other middleware
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			c.Set("userClaims", claims)
			c.Set("username", claims["sub"])
			c.Set("token", tokenString)
		}

		c.Next()
	}
}

// Logout handles user logout and session invalidation.
func (a *API) Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, _ := c.Get("username")
		token, _ := c.Get("token")

			// Invalidate session
		if a.SessionService != nil && token != nil {
			if err := a.SessionService.InvalidateSession(token.(string)); err != nil {
				logger.Warnf("Failed to invalidate session for user %s: %v", username, err)
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}
