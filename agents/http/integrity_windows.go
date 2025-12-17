//go:build windows
// +build windows

package main

import (
	"log"
	"golang.org/x/sys/windows"
)

// checkHighIntegrity determines if the current process is running with high integrity (e.g., Administrator on Windows).
func checkHighIntegrity() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		log.Printf("Error during AllocateAndInitializeSid: %v", err)
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0) // Pseudo handle for current process token
	isMember, err := token.IsMember(sid)
	if err != nil {
		log.Printf("Error during IsMember: %v", err)
		return false
	}
	return isMember
}
