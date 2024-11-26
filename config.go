package main

import (
	"flag"
	"log/slog"
	"os"
	"os/user"
	"strconv"
	"time"
)

type Config struct {
	HomeDir    string
	Port       int
	Once       bool
	Token      string
	LogLevel   *slog.LevelVar
	User       *user.User
	AuditTTY   bool
	AuditPath  string
	AuditExec  bool
	Replay     bool
	ReplayFile string
	Grace      time.Duration
}

func LoadConfig() Config {
	cfg := Config{}
	homeDir, _ := os.UserHomeDir()

	debug := flag.Bool("debug", false, "Debug level logging")

	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&cfg.HomeDir, "home", homeDir, "Home directory for file access")
	flag.StringVar(&cfg.Token, "token", "no-token", "Token to access service")
	username := flag.String("user", "", "User to run shell as")

	// Only allows the first unique user to connect to the shell. When they disconnect the server will exit.
	flag.BoolVar(&cfg.Once, "once", false, "Single use service, only accepts one connection")

	// Grace period for once mode, how long do we give the user to reconnect if their connection drops.
	// This uses user input and the ping message from the term to work out what's active. If we set this too short
	// its possible to kill the server while a user is connected. Default ping interval is 5s.
	graceSecs := flag.Int("grace", 20, "Seconds to wait after disconnecting before stopping server. Used with -once.")

	// Turns on various auditing capabilities.
	flag.BoolVar(&cfg.AuditTTY, "audit-tty", false, "Record users tty session for auditing")
	flag.BoolVar(&cfg.AuditExec, "audit-exec", false, "Record all commands executed by user")
	flag.StringVar(&cfg.AuditPath, "audit-path", "/tmp", "Directory to write audit logs to")
	audit := flag.Bool("audit", false, "Enabled all auditing")

	// Replayer is still work-in-progress
	flag.BoolVar(&cfg.Replay, "replay", false, "Enabled replay of audit files")
	flag.StringVar(&cfg.ReplayFile, "replay-file", "", "Path to audit file to replay")

	flag.Parse()

	// Validate username
	if *username != "" {
		user, err := user.Lookup(*username)
		if err != nil {
			println("Invalid user")
			os.Exit(1)
		} else {
			cfg.User = user
		}
	}

	// Audit shortcut
	if *audit {
		cfg.AuditTTY = true
		cfg.AuditExec = true
	}

	// Debug logging
	cfg.LogLevel = new(slog.LevelVar)
	if *debug {
		cfg.LogLevel.Set(slog.LevelDebug)
	}

	cfg.Grace = time.Duration(*graceSecs) * time.Second

	return cfg
}

func LoadConfigFromEnv() Config {
	cfg := LoadConfig()

	port, ok := os.LookupEnv("PORT")
	if ok {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Port = p
		}
	}

	if token, ok := os.LookupEnv("TOKEN"); ok {
		cfg.Token = token
	}

	if home, ok := os.LookupEnv("HOMEDIR"); ok {
		cfg.HomeDir = home
	}

	return cfg
}
