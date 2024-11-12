package main

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

func checkFileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func chown(f *os.File, user *user.User) error {
	uid, _ := strconv.Atoi(user.Uid)
	gid, _ := strconv.Atoi(user.Gid)
	return f.Chown(uid, gid)
}

func isPathSafe(filename string) bool {
	return strings.HasPrefix(filepath.Clean(filename), config.HomeDir)
}
