package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// loadDotEnv searches for .env by walking up from cwd to the project root.
func loadDotEnv() {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	for {
		path := filepath.Join(cwd, ".env")
		file, err := os.Open(path)
		if err == nil {
			defer file.Close()

			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				key, value, ok := parseEnvLine(scanner.Text())
				if !ok || os.Getenv(key) != "" {
					continue
				}
				_ = os.Setenv(key, value)
			}
			return
		}

		parent := filepath.Dir(cwd)
		if parent == cwd {
			break
		}
		cwd = parent
	}
}

func parseEnvLine(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")
	key, value, ok := strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	if key == "" || strings.ContainsAny(key, " \t") {
		return "", "", false
	}
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	return key, value, true
}
