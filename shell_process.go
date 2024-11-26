package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"

	"webshell/strace"
	"webshell/ttyrec"
)

type ShellProcess struct {
	cmd    *exec.Cmd
	tty    *os.File
	reader io.Reader
	once   sync.Once
	rec    *ttyrec.Recorder
}

func (sp *ShellProcess) Read(b []byte) (n int, err error) {
	return sp.reader.Read(b)
}

func (sp *ShellProcess) Write(b []byte) (n int, err error) {
	return sp.tty.Write(b)
}

func (sp *ShellProcess) Start(command string, args ...string) error {
	var err error

	// Start the shell
	sp.cmd = exec.Command(command, args...)
	sp.cmd.Env = filterEnv(os.Environ())
	sp.cmd.Dir = config.HomeDir

	// TODO: move to params
	if config.User != nil {
		logger.Info(fmt.Sprintf("Running %s as %s", shell, config.User.Username))
		runAs(sp.cmd, config.User)
	}

	tty, err := pty.Start(sp.cmd)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to start %s: %s", shell, err))
	}
	sp.tty = tty
	sp.reader = tty

	return err
}

func (sp *ShellProcess) WithAuditing() error {
	// TODO: check shell is running
	straceAudit := strace.NewStraceLogger(auditLogger)
	if err := straceAudit.Attach(sp.cmd.Process.Pid); err != nil {
		logger.Error(fmt.Sprintf("Syscall auditing failed to start: %v", err))
		return errors.New("syscall auditing failed to start")
	}

	logger.Info("Syscall auditing is enabled")
	return nil
}

func (sp *ShellProcess) WithTTYRecorder(recorder *ttyrec.Recorder) error {
	// TODO: check shell is running
	sp.reader = io.TeeReader(sp.tty, recorder)
	return nil
}

func (sp *ShellProcess) Kill() error {

	sp.once.Do(func() {
		logger.Info(fmt.Sprintf("Killing process %d", sp.cmd.Process.Pid))
		// TODO: should we send SIGTERM instead? The go docs say kill will not kill
		// any processes this proc has started...
		if err := sp.cmd.Process.Kill(); err != nil {
			logger.Error(fmt.Sprintf("Failed to stop process: %s", err))
		}

		if _, err := sp.cmd.Process.Wait(); err != nil {
			logger.Error(fmt.Sprintf("Failed to wait process: %s", err))
		}

		if err := sp.tty.Close(); err != nil {
			logger.Error(fmt.Sprintf("Failed to close tty: %s", err))
		}

		if sp.rec != nil {
			if err := sp.rec.Save(); err != nil {
				logger.Error(fmt.Sprintf("Failed to save audit: %s", err))
			}
			sp.rec.Close()
		}
	})

	return nil
}
