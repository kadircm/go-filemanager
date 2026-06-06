//go:build linux

package utils

import (
	"os"
	"path/filepath"
	"syscall"
)

// MatchParentPermissions matches the UID and GID of the new file/dir with its parent directory on Linux.
func MatchParentPermissions(path string) error {
	parent := filepath.Dir(path)
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return err
	}

	stat, ok := parentInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return nil
	}

	uid := int(stat.Uid)
	gid := int(stat.Gid)

	return os.Chown(path, uid, gid)
}
