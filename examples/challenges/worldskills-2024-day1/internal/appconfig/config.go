package appconfig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config map[string]string

func Load(path string) (Config, error) {
	config := Config{}
	file, err := os.Open(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") ||
				strings.HasPrefix(line, "[") {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if found {
				config[strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	return config, nil
}

func (c Config) Value(key, fallback string) string {
	if value := os.Getenv(strings.ToUpper(key)); value != "" {
		return value
	}
	if value := c[strings.ToLower(key)]; value != "" {
		return value
	}
	return fallback
}

func (c Config) Required(key string) (string, error) {
	value := c.Value(key, "")
	if value == "" {
		return "", fmt.Errorf("%s is required", strings.ToUpper(key))
	}
	return value, nil
}

func (c Config) Duration(key string, fallback time.Duration) (time.Duration, error) {
	value := c.Value(key, fallback.String())
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", strings.ToUpper(key), err)
	}
	return duration, nil
}

func (c Config) Bool(key string) bool {
	switch strings.ToLower(c.Value(key, "false")) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}
