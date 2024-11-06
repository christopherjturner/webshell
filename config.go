package main

import (
	"flag"
	"log/slog"
	"os"
	"os/user"
	"strconv"
)

type Config struct {
	HomeDir   string
	Port      int
	Once      bool
	Token     string
	LogLevel  *slog.LevelVar
	User      *user.User
	AuditTTY  bool
	AuditPath string
	AuditExec bool
	Replay    bool
}

func LoadConfig() Config {
	cfg := Config{}
	homeDir, _ := os.UserHomeDir()

	debug := flag.Bool("debug", false, "Debug level logging")

	flag.BoolVar(&cfg.Once, "once", false, "Single use service, only accepts one connection")
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")
	flag.StringVar(&cfg.HomeDir, "home", homeDir, "Home directory for file access")
	flag.StringVar(&cfg.Token, "token", "no-token", "Token to access service")

	flag.BoolVar(&cfg.AuditTTY, "audit-tty", false, "Record users tty session for auditing")
	flag.BoolVar(&cfg.AuditExec, "audit-exec", false, "Record all commands executed by user")
	flag.StringVar(&cfg.AuditPath, "audit-path", "/tmp", "Directory to write audit logs to")

	// Replayer is still work-in-progress
	// flag.BoolVar(&cfg.Replay, "replay", false, "Enabled replay of audit files.")

	audit := flag.Bool("audit", false, "Enabled all auditing")
	username := flag.String("user", "", "User to run shell as")

	flag.Parse()

	// Validate username
	if *username != "" {
		user, err := user.Lookup(*username)
		if err != nil {
			println("Invalid username")
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
