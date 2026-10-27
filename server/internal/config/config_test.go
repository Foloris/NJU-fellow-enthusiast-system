package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileAndProcessPrecedence(t *testing.T) {
	t.Setenv("CAMPUS_CONFIG_TEST_EXISTING", "process-value")
	const fresh = "CAMPUS_CONFIG_TEST_FRESH"
	previous, exists := os.LookupEnv(fresh)
	os.Unsetenv(fresh)
	t.Cleanup(func() {
		if exists {
			os.Setenv(fresh, previous)
		} else {
			os.Unsetenv(fresh)
		}
	})
	file := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(file, []byte("\ufeff# config\nCAMPUS_CONFIG_TEST_EXISTING=file-value\nCAMPUS_CONFIG_TEST_FRESH='password=with#specials'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnv(file); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("CAMPUS_CONFIG_TEST_EXISTING") != "process-value" || os.Getenv(fresh) != "password=with#specials" {
		t.Fatal("environment values or precedence changed")
	}
}

func TestConfigurationRejectsUnsafeProduction(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("JWT_SECRET", "valid-testing-secret-with-more-than-32-bytes")
	t.Setenv("ADMIN_PASSWORD", "Testing123!")
	t.Setenv("APP_ENV", "production")
	t.Setenv("SEED_DEMO", "true")
	t.Setenv("DB_PORT", "3306")
	if _, err := Load(); err == nil {
		t.Fatal("production accepted demo accounts")
	}
	t.Setenv("SEED_DEMO", "false")
	if cfg, err := Load(); err != nil || !cfg.Production {
		t.Fatalf("valid production config rejected: %v", err)
	}
	t.Setenv("JWT_SECRET", "short")
	if _, err := Load(); err == nil {
		t.Fatal("accepted weak secret")
	}
}

func TestEmailConfiguration(t *testing.T) {
	base := Config{CampusEmailDomain: "smail.nju.edu.cn", SMTPHost: "smtp.qq.com", SMTPPort: "465", SMTPTLSMode: "tls", SMTPUsername: "sender@qq.com", SMTPPassword: "application-password"}
	if err := base.validateEmail(); err != nil || base.SMTPFrom != "sender@qq.com" {
		t.Fatalf("valid SMTP config: %v", err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.CampusEmailDomain = "nju.edu.cn,smail.nju.edu.cn" },
		func(c *Config) { c.SMTPPort = "25bad" }, func(c *Config) { c.SMTPTLSMode = "none" },
		func(c *Config) { c.SMTPPassword = "" }, func(c *Config) { c.SMTPFrom = "sender@qq.com\r\nBcc: bad@example.com" },
	} {
		c := base
		change(&c)
		if c.validateEmail() == nil {
			t.Fatal("invalid email configuration accepted")
		}
	}
}
