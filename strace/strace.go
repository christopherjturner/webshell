package strace

import (
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const SYSCALL_EXECVE = "execve"

var (
	errIgnoredMessage   = errors.New("ignored message")
	errInvalidTimestamp = errors.New("invalid timestamp")

	execveRx = regexp.MustCompile(`^\[pid\s+(\d+)\]\s+([\d\.]+)\s+execve\((.+)\)\s+=\s+\d+`)
)

type StraceExecve struct {
	Pid       string
	Timestamp time.Time
	Cmd       string
}

func filter(s string) bool {
	return strings.Contains(s, SYSCALL_EXECVE)
}

func parse(s string) (StraceExecve, error) {
	println(s)
	res := StraceExecve{}
	m := execveRx.FindStringSubmatch(s)
	if m == nil {
		return res, errIgnoredMessage
	}

	// Extract the fields.
	pid := m[1]
	timestamp := m[2]
	cmd := m[3]

	// Parse the UNIX timestamp
	bSecs, bNs, found := strings.Cut(timestamp, ".")
	if !found {
		return res, errInvalidTimestamp
	}
	secs, errSec := strconv.Atoi(string(bSecs))
	ns, errNs := strconv.Atoi(string(bNs))
	if errSec != nil || errNs != nil {
		return StraceExecve{}, errInvalidTimestamp
	}

	p := StraceExecve{
		Pid:       pid,
		Timestamp: time.Unix(int64(secs), int64(ns)),
		Cmd:       cmd,
	}

	return p, nil
}

type StraceLogger struct {
	buf    strings.Builder
	logger *slog.Logger
	pid    int
	cmd    *exec.Cmd
}

func NewStraceLogger(logger *slog.Logger) *StraceLogger {
	return &StraceLogger{
		logger: logger,
	}
}

func (s *StraceLogger) Attach(pid int) error {

	pathToStrace, err := exec.LookPath("strace")
	if err != nil {
		s.logger.Warn("strace is not installed. Command auditing will be is disabled!")
		return nil
	}

	cmd := exec.Command(pathToStrace, "-fttt", "-qqq", "-s", "2048", "-e", "trace=execve", "-p", fmt.Sprintf("%d", pid))
	cmd.Stderr = s
	s.cmd = cmd
	s.pid = pid

	if err := cmd.Start(); err != nil {
		return err
	}

	s.logger.Info(fmt.Sprintf("Attached to PID %d", pid))

	return nil
}

func (s *StraceLogger) Write(p []byte) (int, error) {
	for _, b := range p {
		s.buf.WriteByte(b)
		if b == '\n' {
			line := s.buf.String()
			if filter(line) {
				if execve, err := parse(line); err == nil {
					s.logger.Info(fmt.Sprintf("Audit: [PID %s] %s", execve.Pid, execve.Cmd))
				}
			}
			s.buf.Reset()
		}
	}

	return len(p), nil
}
