//go:build windows

package credential

import (
	"errors"
	"strings"
	"syscall"
	"unsafe"
)

var ErrInvalidCredentialKey = errors.New("credential service, account, and secret are required")

const (
	credTypeGeneric         = 1
	credPersistLocalMachine = 2
)

type credential struct {
	Flags              uint32
	Type               uint32
	TargetName         *uint16
	Comment            *uint16
	LastWritten        [2]uint32
	CredentialBlobSize uint32
	CredentialBlob     *byte
	Persist            uint32
	AttributeCount     uint32
	Attributes         uintptr
	TargetAlias        *uint16
	UserName           *uint16
}

var (
	advapi32    = syscall.NewLazyDLL("advapi32.dll")
	credWriteW  = advapi32.NewProc("CredWriteW")
	credReadW   = advapi32.NewProc("CredReadW")
	credDeleteW = advapi32.NewProc("CredDeleteW")
	credFree    = advapi32.NewProc("CredFree")
)

func Save(service, account, secret string) error {
	if invalid(service, account) || secret == "" {
		return ErrInvalidCredentialKey
	}
	target, _ := syscall.UTF16PtrFromString(service + ":" + account)
	user, _ := syscall.UTF16PtrFromString(account)
	blob := []byte(secret)
	entry := credential{Type: credTypeGeneric, TargetName: target, UserName: user, CredentialBlobSize: uint32(len(blob)), CredentialBlob: &blob[0], Persist: credPersistLocalMachine}
	result, _, callErr := credWriteW.Call(uintptr(unsafe.Pointer(&entry)), 0)
	if result == 0 {
		return callErr
	}
	return nil
}

func Load(service, account string) (string, error) {
	if invalid(service, account) {
		return "", ErrInvalidCredentialKey
	}
	target, _ := syscall.UTF16PtrFromString(service + ":" + account)
	var entry *credential
	result, _, callErr := credReadW.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&entry)))
	if result == 0 {
		return "", callErr
	}
	defer credFree.Call(uintptr(unsafe.Pointer(entry)))
	return string(unsafe.Slice(entry.CredentialBlob, entry.CredentialBlobSize)), nil
}

func Delete(service, account string) error {
	if invalid(service, account) {
		return ErrInvalidCredentialKey
	}
	target, _ := syscall.UTF16PtrFromString(service + ":" + account)
	result, _, callErr := credDeleteW.Call(uintptr(unsafe.Pointer(target)), credTypeGeneric, 0)
	if result == 0 {
		return callErr
	}
	return nil
}

func invalid(service, account string) bool {
	return service == "" || account == "" || strings.ContainsRune(service, 0) || strings.ContainsRune(account, 0)
}
