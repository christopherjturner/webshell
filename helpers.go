package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"syscall"
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

func runAs(cmd *exec.Cmd, user *user.User) {
	uid, _ := strconv.ParseInt(user.Uid, 10, 32)
	gid, _ := strconv.ParseInt(user.Gid, 10, 32)

	cmd.SysProcAttr = &syscall.SysProcAttr{}
	cmd.SysProcAttr.Credential = &syscall.Credential{Uid: uint32(uid), Gid: uint32(gid)}
	cmd.Env = append(cmd.Env, fmt.Sprintf("HOME=%s", user.HomeDir))
}

var restrictedEnvVars = map[string]bool{
	"AUDIT_UPLOAD_URL": true,
}

// Removes any restricted keys from the parent environment
func filterEnv(o []string) []string {
	environ := []string{}
	for _, e := range o {
		key, _, _ := strings.Cut(e, "=")
		if _, found := restrictedEnvVars[key]; !found {
			environ = append(environ, e)
		}
	}
	return environ
}
