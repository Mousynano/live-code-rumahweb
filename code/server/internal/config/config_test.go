package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFileDoesNotOverrideExistingEnvironment(t *testing.T) {
	t.Setenv("EXISTING_VALUE", "from-environment")
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	if err := os.WriteFile(path, []byte("EXISTING_VALUE=from-file\nNEW_VALUE=loaded\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("EXISTING_VALUE"); got != "from-environment" {
		t.Fatalf("existing environment was overwritten: %q", got)
	}
	if got := os.Getenv("NEW_VALUE"); got != "loaded" {
		t.Fatalf("new value was not loaded: %q", got)
	}
}

func TestLoadDotEnvFileSupportsQuotedValues(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, ".env")
	if err := os.WriteFile(path, []byte("DOUBLE_QUOTED=\"hello world\"\nSINGLE_QUOTED='hello again'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("DOUBLE_QUOTED"); got != "hello world" {
		t.Fatalf("unexpected double quoted value: %q", got)
	}
	if got := os.Getenv("SINGLE_QUOTED"); got != "hello again" {
		t.Fatalf("unexpected single quoted value: %q", got)
	}
}
