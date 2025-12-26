package handler

import "testing"

func TestGetAdminUIConfig_EnvTokenEnables(t *testing.T) {
	withGlobalConfig(t, &Config{})
	t.Setenv("ADMIN_UI_TOKEN", "env-token")

	cfg := GetAdminUIConfig()
	if cfg.Token != "env-token" {
		t.Fatalf("expected token from env, got %q", cfg.Token)
	}
	if !cfg.Enabled {
		t.Fatal("expected admin ui enabled when token is set")
	}
	if cfg.BasePath != "/v1/admin" {
		t.Fatalf("expected default base path, got %q", cfg.BasePath)
	}
}

func TestGetAdminUIConfig_ExplicitDisable(t *testing.T) {
	withGlobalConfig(t, &Config{
		AdminUI: &AdminUIConfig{
			Enabled: true,
			Token:   "config-token",
		},
	})
	t.Setenv("ADMIN_UI_TOKEN", "env-token")
	t.Setenv("ADMIN_UI_ENABLED", "false")

	cfg := GetAdminUIConfig()
	if cfg.Enabled {
		t.Fatal("expected admin ui disabled when env says false")
	}
}

func TestGetAdminUIConfig_DisableWithoutToken(t *testing.T) {
	withGlobalConfig(t, &Config{
		AdminUI: &AdminUIConfig{
			Enabled: true,
			Token:   "",
		},
	})

	cfg := GetAdminUIConfig()
	if cfg.Enabled {
		t.Fatal("expected admin ui disabled when token is empty")
	}
}

func withGlobalConfig(t *testing.T, cfg *Config) {
	t.Helper()
	previous := getGlobalConfig()
	previousPath := getConfigPath()
	previousLoaded := getConfigLoadedAt()
	setGlobalConfig(cfg, previousPath, previousLoaded)
	t.Cleanup(func() {
		setGlobalConfig(previous, previousPath, previousLoaded)
	})
}
