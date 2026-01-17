//go:build windows

package state

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = kernel32.NewProc("LockFileEx")
	procUnlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
)

// lockFile acquires an exclusive lock on the file (Windows implementation)
func lockFile(file *os.File) error {
	var overlapped syscall.Overlapped

	ret, _, err := procLockFileEx.Call(
		uintptr(file.Fd()),
		uintptr(LOCKFILE_EXCLUSIVE_LOCK),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}

// unlockFile releases the lock on the file (Windows implementation)
func unlockFile(file *os.File) error {
	var overlapped syscall.Overlapped

	ret, _, err := procUnlockFileEx.Call(
		uintptr(file.Fd()),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}
