package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "conf-*.toml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestSetConfig_LoadValid(t *testing.T) {
	conf := &ConfigSettings{}
	path := filepath.Join("..", "..", "config", "event.conf.example")
	if err := conf.SetConfig(path); err != nil {
		t.Fatalf("failed to load example config: %v", err)
	}
	if len(conf.Box) != 3 {
		t.Fatalf("expected 3 boxes, got %d", len(conf.Box))
	}
}

func TestSetConfig_MissingRequired(t *testing.T) {
	config := `[RequiredSettings]
EventType="rvb"
DBConnectURL="postgres://user:pass@localhost/db"
BindAddress="0.0.0.0"`
	path := writeTempConfig(t, config)
	conf := &ConfigSettings{}
	if err := conf.SetConfig(path); err == nil {
		t.Fatal("expected error for missing event name")
	}
}

func TestSetConfig_DuplicateBox(t *testing.T) {
	config := `[RequiredSettings]
EventName="test"
EventType="rvb"
DBConnectURL="postgres://user:pass@localhost/db"
BindAddress="0.0.0.0"

[[Team]]
Name="team1"
Pw="pw"

[[Box]]
Name="box1"
IP="10.0.0.1"

[[Box]]
Name="box1"
IP="10.0.0.2"`
	path := writeTempConfig(t, config)
	conf := &ConfigSettings{}
	if err := conf.SetConfig(path); err == nil || !strings.Contains(err.Error(), "duplicate box name") {
		t.Fatalf("expected duplicate box name error, got %v", err)
	}
}

func TestSetConfig_DefaultValues(t *testing.T) {
	config := `[RequiredSettings]
EventName="test"
EventType="rvb"
DBConnectURL="postgres://user:pass@localhost/db"
BindAddress="0.0.0.0"

[[Team]]
Name="team1"
Pw="pw"

[[Box]]
Name="box1"
IP="10.0.0.1"`
	path := writeTempConfig(t, config)
	conf := &ConfigSettings{}
	if err := conf.SetConfig(path); err != nil {
		t.Fatalf("failed to load config: %v", err)
	}
	if conf.MiscSettings.Delay != 60 {
		t.Errorf("default delay should be 60, got %d", conf.MiscSettings.Delay)
	}
	if conf.MiscSettings.Jitter != 5 {
		t.Errorf("default jitter should be 5, got %d", conf.MiscSettings.Jitter)
	}
	if conf.MiscSettings.Timeout != 30 {
		t.Errorf("default timeout should be 30, got %d", conf.MiscSettings.Timeout)
	}
	if conf.MiscSettings.Points != 1 {
		t.Errorf("default points should be 1, got %d", conf.MiscSettings.Points)
	}
	if conf.MiscSettings.SlaThreshold != 5 {
		t.Errorf("default sla threshold should be 5, got %d", conf.MiscSettings.SlaThreshold)
	}
	if conf.MiscSettings.SlaPenalty != 5 {
		t.Errorf("default sla penalty should be 5, got %d", conf.MiscSettings.SlaPenalty)
	}
}

func TestWatchConfigReloads(t *testing.T) {
	content := `[RequiredSettings]
EventName="first"
EventType="rvb"
DBConnectURL="postgres://user:pass@localhost/db"
BindAddress="0.0.0.0"

[[Team]]
Name="team1"
Pw="pw"

[[Box]]
Name="box1"
IP="10.0.0.1"`
	path := writeTempConfig(t, content)
	conf := &ConfigSettings{}
	if err := conf.SetConfig(path); err != nil {
		t.Fatalf("initial load failed: %v", err)
	}
	if err := conf.WatchConfig(path); err != nil {
		t.Fatalf("watch setup failed: %v", err)
	}
	// update file
	updated := strings.Replace(content, "first", "second", 1)
	os.WriteFile(path, []byte(updated), 0o644)
	time.Sleep(1500 * time.Millisecond)
	if conf.RequiredSettings.EventName != "second" {
		t.Fatalf("expected reload to update event name, got %s", conf.RequiredSettings.EventName)
	}
}
