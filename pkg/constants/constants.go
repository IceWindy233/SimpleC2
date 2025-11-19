package constants

const (
	// Beacon Commands
	CmdShell   = "shell"
	CmdSleep   = "sleep"
	CmdBrowse  = "browse"
	CmdDownload = "download"
	CmdUpload  = "upload"
	CmdExit    = "exit"
	
	// File Operations
	ChunkSize = 1024 * 1024 // 1MB
)

var ValidCommands = map[string]struct{}{
	CmdShell:    {},
	CmdSleep:    {},
	CmdBrowse:   {},
	CmdDownload: {},
	CmdUpload:   {},
	CmdExit:     {},
}