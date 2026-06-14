package config

import (
	"testing"
)

func TestLoad_SessionSecret(t *testing.T) {
	secret := "test-secret-32-chars-long-enough!"
	t.Setenv("SESSION_SECRET", secret)

	cfg := Load()

	if cfg.SessionSecret != secret {
		t.Errorf("SessionSecret = %q, want %q", cfg.SessionSecret, secret)
	}
}

func TestLoad_DevMode_True(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret-32-chars-long-enough!")
	t.Setenv("DEV_MODE", "true")

	cfg := Load()

	if !cfg.DevMode {
		t.Error("expected DevMode=true when DEV_MODE=true")
	}
}

func TestLoad_DevMode_False(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret-32-chars-long-enough!")
	t.Setenv("DEV_MODE", "false")

	cfg := Load()

	if cfg.DevMode {
		t.Error("expected DevMode=false when DEV_MODE=false")
	}
}

func TestLoad_TrustedProxies_Parsed(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret-32-chars-long-enough!")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1, 172.16.0.0/12 , 192.168.1.1")

	cfg := Load()

	if len(cfg.TrustedProxies) != 3 {
		t.Fatalf("expected 3 trusted proxies, got %d: %v", len(cfg.TrustedProxies), cfg.TrustedProxies)
	}
	if cfg.TrustedProxies[0] != "10.0.0.1" {
		t.Errorf("proxy[0] = %q, want 10.0.0.1", cfg.TrustedProxies[0])
	}
	if cfg.TrustedProxies[1] != "172.16.0.0/12" {
		t.Errorf("proxy[1] = %q, want 172.16.0.0/12", cfg.TrustedProxies[1])
	}
	if cfg.TrustedProxies[2] != "192.168.1.1" {
		t.Errorf("proxy[2] = %q, want 192.168.1.1", cfg.TrustedProxies[2])
	}
}

func TestLoad_TrustedProxies_SingleEntry(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret-32-chars-long-enough!")
	t.Setenv("TRUSTED_PROXIES", "10.0.0.1")

	cfg := Load()

	if len(cfg.TrustedProxies) != 1 || cfg.TrustedProxies[0] != "10.0.0.1" {
		t.Errorf("expected [10.0.0.1], got %v", cfg.TrustedProxies)
	}
}

func TestLoad_ImprintFields(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret-32-chars-long-enough!")
	t.Setenv("IMPRINT_NAME", "Dr. Max Mustermann")
	t.Setenv("IMPRINT_STREET", "Musterstr. 1")
	t.Setenv("IMPRINT_ZIP", "12345")
	t.Setenv("IMPRINT_CITY", "Musterstadt")
	t.Setenv("IMPRINT_EMAIL", "max@example.com")

	cfg := Load()

	if cfg.ImprintName != "Dr. Max Mustermann" {
		t.Errorf("ImprintName = %q", cfg.ImprintName)
	}
	if cfg.ImprintStreet != "Musterstr. 1" {
		t.Errorf("ImprintStreet = %q", cfg.ImprintStreet)
	}
	if cfg.ImprintZip != "12345" {
		t.Errorf("ImprintZip = %q", cfg.ImprintZip)
	}
	if cfg.ImprintCity != "Musterstadt" {
		t.Errorf("ImprintCity = %q", cfg.ImprintCity)
	}
	if cfg.ImprintEmail != "max@example.com" {
		t.Errorf("ImprintEmail = %q", cfg.ImprintEmail)
	}
}

func TestLoad_ListenAddr(t *testing.T) {
	t.Setenv("SESSION_SECRET", "test-secret-32-chars-long-enough!")
	t.Setenv("LISTEN_ADDR", ":9090")

	cfg := Load()

	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q, want :9090", cfg.ListenAddr)
	}
}
