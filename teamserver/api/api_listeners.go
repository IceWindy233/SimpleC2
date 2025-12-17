package api

import (
	"archive/zip"
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"simplec2/pkg/config"
	"simplec2/pkg/logger"
	"simplec2/pkg/pki"
	"strconv"
	"strings" // Import strings

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// CreateListenerRequest defines the structure for the listener creation API request body.
type CreateListenerRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Config string `json:"config"`
}

// CreateListener godoc
// @Summary Generate listener configuration
// @Description Generates a ZIP package containing configuration and certificates for a new listener.
// @Tags listeners
// @Accept  json
// @Produce  application/zip
// @Param listener body CreateListenerRequest true "Listener details"
// @Success 200 {file} binary
// @Failure 400 {object} gin.H{"error": string} "Invalid request body"
// @Failure 500 {object} gin.H{"error": string} "Internal server error"
// @Router /listeners [post]
func (a *API) CreateListener(c *gin.Context) {
	var req CreateListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	// 1. Load CA
	caCertPath := a.Config.GRPC.Certs.CACert
	// Assuming ca.key is in the same directory as ca.crt
	caKeyPath := filepath.Join(filepath.Dir(caCertPath), "ca.key")

	caCertPEM, err := os.ReadFile(caCertPath)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to read CA certificate", err.Error()))
		return
	}
	caKeyPEM, err := os.ReadFile(caKeyPath)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to read CA private key", err.Error()))
		return
	}

	// 2. Generate Keys & Certs


	// mTLS Client Cert
	clientPriv, clientCert, err := pki.GenerateCert(pki.CertConfig{
		CommonName: "SimpleC2 Listener - " + req.Name,
		IsClient:   true,
	}, caCertPEM, caKeyPEM)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to generate client certificate", err.Error()))
		return
	}

	// Parse generated certificate to extract Serial Number
	block, _ := pem.Decode(clientCert)
	if block == nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to decode generated certificate", "PEM decode failed"))
		return
	}
	parsedCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to parse generated certificate", err.Error()))
		return
	}
	
	// Record issued certificate
	if err := a.ListenerService.RecordIssuedCertificate(c.Request.Context(), parsedCert.SerialNumber.String(), parsedCert.Subject.CommonName, req.Name); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to record issued certificate", err.Error()))
		return
	}

	// 3. Generate listener.yaml
	// Parse the raw JSON config from request to get port
	var configMap map[string]interface{}
	if err := json.Unmarshal([]byte(req.Config), &configMap); err != nil {
		configMap = make(map[string]interface{})
	}
	portStr := ":8888" // Default listener port
	if p, ok := configMap["port"]; ok {
		switch v := p.(type) {
		case float64: // JSON numbers are float64 by default
			if v == 0 {
				portStr = ":8888" // Explicitly use default if 0 is provided
			} else {
				portStr = fmt.Sprintf(":%d", int(v))
			}
		case string: // Could be ":8080" or "8080"
			trimmed := strings.TrimSpace(v)
			if trimmed == "0" || trimmed == ":0" || trimmed == "" {
				portStr = ":8888"
			} else if !strings.HasPrefix(trimmed, ":") {
				portStr = ":" + trimmed
			} else {
				portStr = trimmed
			}
		default:
			// Invalid port type, stick with default
			portStr = ":8888"
		}
	}
	// Ensure portStr is never empty or just ":" after processing
	if portStr == "" || portStr == ":" {
		portStr = ":8888"
	}

	// We need to fetch the API Key. In a real scenario, we might generate a new specific API Key for this listener.
	// For now, let's use the TeamServer's configured API Key (or the one from config).
	// NOTE: Ideally, we should generate a unique API Key for each listener for better security/revocation.
	apiKey, _ := a.Config.Auth.GetAPIKey()
	
	listenerCfg := config.ListenerConfig{
		TeamServer: struct {
			Host string `yaml:"host"`
			Port string `yaml:"port"`
		}{
			Host: "localhost", // Users should probably update this manually or we detect TS public IP
			Port: a.Config.GRPC.Port,
		},
		Listener: struct {
			Name string `yaml:"name"`
			Port string `yaml:"port"`
		}{
			Name: req.Name,
			Port: portStr,
		},
		Auth: struct {
			APIKey           string `yaml:"api_key,omitempty"`
			EncryptedAPIKey  *config.EncryptedAPIKey `yaml:"encrypted_api_key,omitempty"`
		}{
			APIKey: apiKey, // In production, encrypt this!
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
	
	yamlData, err := yaml.Marshal(&listenerCfg)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to marshal listener config", err.Error()))
		return
	}


	// 4. Create ZIP
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	files := map[string][]byte{
		"listener.yaml":        yamlData,
		"certs/client.crt":     clientCert,
		"certs/client.key":     clientPriv,
		"certs/ca.crt":         caCertPEM,
		//"certs/listener_rsa.key": rsaPriv, // Removed
		//"listener.pub":         rsaPub, // Removed
	}

	for name, content := range files {
		f, err := zipWriter.Create(name)
		if err != nil {
			Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to create zip entry", err.Error()))
			return
		}
		_, err = f.Write(content)
		if err != nil {
			Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to write zip entry", err.Error()))
			return
		}
	}

	if err := zipWriter.Close(); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to close zip", err.Error()))
		return
	}

	// 5. Return response
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"listener_%s.zip\"", req.Name))
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// GetListeners godoc
// @Summary Get all listeners
// @Description Retrieves a list of all active listeners.
// @Tags listeners
// @Produce  json
// @Success 200 {object} StandardResponse{data=[]data.Listener}
// @Failure 500 {object} StandardResponse
// @Router /listeners [get]
func (a *API) GetListeners(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid 'page' parameter", "must be an integer"))
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid 'limit' parameter", "must be an integer"))
		return
	}

	listeners, total, err := a.ListenerService.ListListeners(c.Request.Context(), page, limit)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to retrieve listeners", err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}
	Respond(c, http.StatusOK, NewSuccessResponse(listeners, meta))
}

// DeleteListener godoc
// @Summary Delete a listener
// @Description Stops and deletes a listener by its name.
// @Tags listeners
// @Produce  json
// @Param name path string true "The name of the listener to delete"
// @Success 204 "No Content"
// @Failure 404 {object} StandardResponse
// @Failure 500 {object} StandardResponse
// @Router /listeners/{name} [delete]
func (a *API) DeleteListener(c *gin.Context) {
	listenerName := c.Param("name")

	// Get listener info before deletion for event broadcasting
	listener, err := a.ListenerService.GetListener(c.Request.Context(), listenerName)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Listener not found", err.Error()))
		return
	}

	err = a.ListenerService.DeleteListener(c.Request.Context(), listenerName)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to delete listener", err.Error()))
		return
	}

	// Broadcast LISTENER_STOPPED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "LISTENER_STOPPED",
		Payload: listener,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling LISTENER_STOPPED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted LISTENER_STOPPED event for %s", listenerName)
		}
	}

	c.Status(http.StatusNoContent)
}

// StartListener sends a start command to the listener.
func (a *API) StartListener(c *gin.Context) {
	name := c.Param("name")
	if err := a.ListenerService.StartListener(c.Request.Context(), name); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to start listener", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(gin.H{"message": "Start command sent"}, nil))
}

// StopListener sends a stop command to the listener.
func (a *API) StopListener(c *gin.Context) {
	name := c.Param("name")
	if err := a.ListenerService.StopListener(c.Request.Context(), name); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to stop listener", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(gin.H{"message": "Stop command sent"}, nil))
}

// RestartListener sends a restart command to the listener.
func (a *API) RestartListener(c *gin.Context) {
	name := c.Param("name")
	if err := a.ListenerService.RestartListener(c.Request.Context(), name); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to restart listener", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(gin.H{"message": "Restart command sent"}, nil))
}
