package main

import (
	"flag"
	"log/slog"
	"os"
	"strconv"
)

type Config struct {
	HomeDir  string
	Port     int
	Once     bool
	Token    string
	LogLevel *slog.LevelVar
}

func LoadConfig() Config {
	cfg := Config{}

	flag.BoolVar(&cfg.Once, "once", false, "Single use service, only accepts one connection")
	flag.IntVar(&cfg.Port, "port", 8080, "Port to listen on")

	homeDir, _ := os.UserHomeDir()
	flag.StringVar(&cfg.HomeDir, "home", homeDir, "Home directory for file access")

	flag.StringVar(&cfg.Token, "token", "no-token", "Token to access service")

	debug := flag.Bool("debug", false, "Debug level logging")

	flag.Parse()

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
