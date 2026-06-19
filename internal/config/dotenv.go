package config

import (
	"bufio"
	"os"
	"strings"
)

// LoadDotEnv reads a .env-style file and populates os.Environ for any KEY=VALUE
// pairs not already present in the environment. It mirrors the common behaviour
// of dotenv loaders: quoted values are unquoted, blank lines and '#' comment
// lines are skipped, and a '#' inside an unquoted value is preserved as long as
// it is not preceded by whitespace (so passwords containing '#' stay intact).
//
// Existing environment variables take precedence over the file, so real shell
// exports always win. A missing file is treated as a no-op (local dev may have
// none). Path defaults to ".env" when envFile is empty.
func LoadDotEnv(envFile string) error {
	if envFile == "" {
		envFile = ".env"
	}

	file, err := os.Open(envFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Strip a leading "export " if present (shell-style).
		line = strings.TrimPrefix(line, "export ")

		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key == "" {
			continue
		}
		// Keys must be identifiers; skip anything malformed.
		if !isValidEnvKey(key) {
			continue
		}

		value := strings.TrimSpace(line[eq+1:])
		value = parseEnvValue(value)

		// Real environment wins over the file — do not clobber existing vars.
		if _, ok := os.LookupEnv(key); ok {
			continue
		}
		if err := os.Setenv(key, value); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// parseEnvValue strips surrounding single/double quotes and trims an inline
// trailing comment (" # ..." ) only when the value is unquoted.
func parseEnvValue(value string) string {
	if len(value) >= 2 {
		first := value[0]
		last := value[len(value)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return value[1 : len(value)-1]
		}
	}

	// Unquoted value: drop an inline comment starting with " #" (whitespace
	// required, so "a#b" stays intact).
	if idx := strings.Index(value, " #"); idx != -1 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
