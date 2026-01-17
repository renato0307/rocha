//go:build unix

package state

import (
	"os"

	"golang.org/x/sys/unix"
)

// lockFile acquires an exclusive lock on the file (Unix implementation)
func lockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

// unlockFile releases the lock on the file (Unix implementation)
func unlockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
