package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Address       string
	DBHost        string
	DBPort        string
	DBName        string
	DBUser        string
	DBPassword    string
	JWTSecret     string
	AdminUsername string
	AdminPassword string
	SeedDemo      bool
	Production    bool
	UploadDir     string
	ClientDir     string
}

// LoadEnv supports KEY=value, comments and quoted values. Process variables take precedence.
func LoadEnv(filename string) error {
	f, err := os.Open(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(strings.TrimPrefix(key, "export "))
		if !ok || key == "" || strings.ContainsAny(key, " \t") {
			return fmt.Errorf("invalid environment setting: %q", key)
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func Load() (Config, error) {
	if err := LoadEnv(".env"); err != nil {
		return Config{}, err
	}
	c := Config{
		Address: ":" + env("PORT", "3001"), DBHost: env("DB_HOST", "127.0.0.1"), DBPort: env("DB_PORT", "3306"),
		DBName: env("DB_NAME", "campus_fellow"), DBUser: env("DB_USER", "campus"), DBPassword: os.Getenv("DB_PASSWORD"),
		JWTSecret: os.Getenv("JWT_SECRET"), AdminUsername: env("ADMIN_USERNAME", "admin"), AdminPassword: os.Getenv("ADMIN_PASSWORD"),
		SeedDemo: os.Getenv("SEED_DEMO") == "true", Production: os.Getenv("APP_ENV") == "production",
		UploadDir: env("UPLOAD_DIR", "server/uploads"), ClientDir: env("CLIENT_DIST_DIR", "client/dist"),
	}
	if len(c.JWTSecret) < 32 || c.JWTSecret == "replace-this-with-a-long-random-string" {
		return c, errors.New("JWT_SECRET must be a non-placeholder secret of at least 32 bytes")
	}
	if len(c.AdminPassword) < 8 || len(c.AdminPassword) > 72 {
		return c, errors.New("ADMIN_PASSWORD must be 8 to 72 bytes")
	}
	if c.Production && c.SeedDemo {
		return c, errors.New("SEED_DEMO must be false in production")
	}
	if port, err := strconv.Atoi(c.DBPort); err != nil || port < 1 || port > 65535 {
		return c, errors.New("DB_PORT is invalid")
	}
	for _, p := range []*string{&c.UploadDir, &c.ClientDir} {
		absolute, err := filepath.Abs(*p)
		if err != nil {
			return c, err
		}
		*p = absolute
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
