//go:build windows

package installer

import (
	"fmt"
	"strings"
	"syscall"
	"unsafe"
)

const (
	hkeyCurrentUser = 0x80000001
	regKeyPath      = "Environment"
	regValueName    = "Path"

	keyQueryValue = 0x0001
	keySetValue   = 0x0002
	keyRead       = 0x20019
	keyWrite      = 0x20006

	regSZ        = 1
	regExpandSZ  = 2
	hwndBroadcast = 0xffff
	wmSettingChange = 0x001a
	smtoAbortIfHung = 0x0002
)

var (
	advapi32             = syscall.NewLazyDLL("advapi32.dll")
	procRegOpenKeyExW    = advapi32.NewProc("RegOpenKeyExW")
	procRegQueryValueExW = advapi32.NewProc("RegQueryValueExW")
	procRegSetValueExW   = advapi32.NewProc("RegSetValueExW")
	procRegCloseKey      = advapi32.NewProc("RegCloseKey")

	user32                     = syscall.NewLazyDLL("user32.dll")
	procSendMessageTimeoutW    = user32.NewProc("SendMessageTimeoutW")
)

// WindowsPathManager implements PathManager on Windows using the Windows Registry.
type WindowsPathManager struct{}

// NewPlatformPathManager creates a PathManager for the current platform.
func NewPlatformPathManager(homeDir string, userShell string) PathManager {
	return &WindowsPathManager{}
}

// GetPath retrieves the current User PATH environment variable from the Windows Registry.
func (m *WindowsPathManager) GetPath() (string, error) {
	var hKey syscall.Handle
	keyNamePtr, err := syscall.UTF16PtrFromString(regKeyPath)
	if err != nil {
		return "", err
	}

	ret, _, _ := procRegOpenKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(keyNamePtr)),
		0,
		uintptr(keyRead),
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return "", fmt.Errorf("failed to open registry key: %w", syscall.Errno(ret))
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	valNamePtr, err := syscall.UTF16PtrFromString(regValueName)
	if err != nil {
		return "", err
	}

	var valType uint32
	var valSize uint32

	// First probe the size
	ret, _, _ = procRegQueryValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(valNamePtr)),
		0,
		uintptr(unsafe.Pointer(&valType)),
		0,
		uintptr(unsafe.Pointer(&valSize)),
	)
	if ret != 0 {
		// Key might not exist or empty
		return "", nil
	}

	if valSize == 0 {
		return "", nil
	}

	buf := make([]uint16, valSize/2+1)
	ret, _, _ = procRegQueryValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(valNamePtr)),
		0,
		uintptr(unsafe.Pointer(&valType)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&valSize)),
	)
	if ret != 0 {
		return "", fmt.Errorf("failed to query registry value: %w", syscall.Errno(ret))
	}

	return syscall.UTF16ToString(buf), nil
}

// SetPath writes the new User PATH to the Windows Registry.
func (m *WindowsPathManager) SetPath(newPath string) error {
	var hKey syscall.Handle
	keyNamePtr, err := syscall.UTF16PtrFromString(regKeyPath)
	if err != nil {
		return err
	}

	ret, _, _ := procRegOpenKeyExW.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(keyNamePtr)),
		0,
		uintptr(keyWrite),
		uintptr(unsafe.Pointer(&hKey)),
	)
	if ret != 0 {
		return fmt.Errorf("failed to open registry key for writing: %w", syscall.Errno(ret))
	}
	defer procRegCloseKey.Call(uintptr(hKey))

	valNamePtr, err := syscall.UTF16PtrFromString(regValueName)
	if err != nil {
		return err
	}

	utf16Val, err := syscall.UTF16FromString(newPath)
	if err != nil {
		return err
	}

	ret, _, _ = procRegSetValueExW.Call(
		uintptr(hKey),
		uintptr(unsafe.Pointer(valNamePtr)),
		0,
		uintptr(regExpandSZ),
		uintptr(unsafe.Pointer(&utf16Val[0])),
		uintptr(len(utf16Val)*2),
	)
	if ret != 0 {
		return fmt.Errorf("failed to set registry value: %w", syscall.Errno(ret))
	}

	return nil
}

// AddEntry safely appends a directory to User PATH.
func (m *WindowsPathManager) AddEntry(entry string) error {
	curPath, err := m.GetPath()
	if err != nil {
		return err
	}

	var entries []string
	if curPath != "" {
		entries = strings.Split(curPath, ";")
	}

	updated := AddPathEntry(entries, entry, "windows")
	newPath := strings.Join(updated, ";")

	if err := m.SetPath(newPath); err != nil {
		return err
	}

	_ = m.BroadcastChange()
	return nil
}

// RemoveEntries safely removes specific directories from User PATH while preserving unrelated entries.
func (m *WindowsPathManager) RemoveEntries(toRemove []string) error {
	curPath, err := m.GetPath()
	if err != nil {
		return err
	}

	if curPath == "" {
		return nil
	}

	entries := strings.Split(curPath, ";")
	filtered := FilterPathList(entries, toRemove, "windows")
	newPath := strings.Join(filtered, ";")

	if err := m.SetPath(newPath); err != nil {
		return err
	}

	_ = m.BroadcastChange()
	return nil
}

// BroadcastChange broadcasts WM_SETTINGCHANGE so new shells and Explorer pick up the new environment variables.
func (m *WindowsPathManager) BroadcastChange() error {
	envStr, err := syscall.UTF16PtrFromString("Environment")
	if err != nil {
		return err
	}

	var result uintptr
	procSendMessageTimeoutW.Call(
		uintptr(hwndBroadcast),
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(envStr)),
		uintptr(smtoAbortIfHung),
		5000,
		uintptr(unsafe.Pointer(&result)),
	)

	return nil
}
