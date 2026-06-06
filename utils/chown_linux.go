//go:build linux

package utils

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
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

// GetFileOwner returns the owner UID, GID and their names for a file
func GetFileOwner(path string) (uid, gid int, username, groupname string, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, "", "", err
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, "", "", fmt.Errorf("unable to get file ownership info")
	}

	uid = int(stat.Uid)
	gid = int(stat.Gid)

	// Try to resolve username
	u, err := user.LookupId(strconv.Itoa(uid))
	if err == nil {
		username = u.Username
	} else {
		username = strconv.Itoa(uid)
	}

	// Try to resolve group name
	g, err := user.LookupGroupId(strconv.Itoa(gid))
	if err == nil {
		groupname = g.Name
	} else {
		groupname = strconv.Itoa(gid)
	}

	return uid, gid, username, groupname, nil
}

// ChangeOwner changes the owner of a file or directory
func ChangeOwner(path string, uid, gid int) error {
	return os.Chown(path, uid, gid)
}
