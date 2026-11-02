package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	BuildMode       string
	DBPath          string
	APIPrefix       string
	ServerPort      string
	DefaultResolver string
}

func Load(path string) (*Config, error) {
	cfg := &Config{
		BuildMode:       "static",
		DBPath:          "./data/shortcuts.db",
		APIPrefix:       "/api",
		ServerPort:      "8080",
		DefaultResolver: "",
	}

	file, err := os.Open(path)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "BUILD_MODE":
				cfg.BuildMode = value
			case "DB_PATH":
				cfg.DBPath = value
			case "API_PREFIX":
				cfg.APIPrefix = value
			case "SERVER_PORT":
				cfg.ServerPort = value
			case "DEFAULT_RESOLVER":
				cfg.DefaultResolver = value
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	if v := os.Getenv("BUILD_MODE"); v != "" {
		cfg.BuildMode = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("API_PREFIX"); v != "" {
		cfg.APIPrefix = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		cfg.ServerPort = v
	}
	if v := os.Getenv("DEFAULT_RESOLVER"); v != "" {
		cfg.DefaultResolver = v
	}

	return cfg, nil
}
