//go:build !linux

package utils

import (
	"fmt"
)

// MatchParentPermissions is a no-op on non-linux systems.
func MatchParentPermissions(path string) error {
	return nil
}

// GetFileOwner returns placeholder values on non-linux systems.
func GetFileOwner(path string) (uid, gid int, username, groupname string, err error) {
	return 0, 0, "-", "-", nil
}

// ChangeOwner is not supported on non-linux systems.
func ChangeOwner(path string, uid, gid int) error {
	return fmt.Errorf("owner değiştirme bu işletim sisteminde desteklenmiyor")
}
