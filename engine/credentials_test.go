package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"quotient/engine/config"
	"quotient/engine/db"
)

// helper to stub getTeams during a test
func withTeams(t *testing.T, teams []db.TeamSchema) func() {
	t.Helper()
	old := getTeams
	getTeams = func() ([]db.TeamSchema, error) { return teams, nil }
	return func() { getTeams = old }
}

func TestLoadCredentialsCopiesFiles(t *testing.T) {
	cleanup := withTeams(t, []db.TeamSchema{{ID: 1}, {ID: 2}})
	defer cleanup()

	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	os.MkdirAll("config/credlists", 0o755)
	os.MkdirAll("submissions/pcrs", 0o755)

	content := "user1,pass1\nuser2,pass2\n"
	os.WriteFile("config/credlists/list.csv", []byte(content), 0o644)

	conf := &config.ConfigSettings{
		CredlistSettings: config.CredlistConfig{
			Credlist: []config.Credlist{{CredlistName: "list", CredlistPath: "list.csv", CredlistExplainText: "username,password"}},
		},
	}

	se := &ScoringEngine{Config: conf, CredentialsMutex: map[uint]*sync.Mutex{}}

	if err := se.LoadCredentials(); err != nil {
		t.Fatalf("LoadCredentials failed: %v", err)
	}

	for _, id := range []uint{1, 2} {
		path := filepath.Join("submissions/pcrs", fmt.Sprintf("%d/list.csv", id))
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected credlist copied for team %d", id)
		}
	}
}

func TestUpdateCredentials(t *testing.T) {
	cleanup := withTeams(t, []db.TeamSchema{{ID: 1}})
	defer cleanup()

	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	os.MkdirAll("config/credlists", 0o755)
	os.MkdirAll("submissions/pcrs/1", 0o755)

	content := "user1,pass1\nuser2,pass2\n"
	os.WriteFile("config/credlists/list.csv", []byte(content), 0o644)
	// copy initial file
	os.WriteFile("submissions/pcrs/1/list.csv", []byte(content), 0o644)

	conf := &config.ConfigSettings{
		CredlistSettings: config.CredlistConfig{
			Credlist: []config.Credlist{{CredlistName: "list", CredlistPath: "list.csv", CredlistExplainText: "username,password"}},
		},
	}
	se := &ScoringEngine{Config: conf, CredentialsMutex: map[uint]*sync.Mutex{1: {}}}

	updated, err := se.UpdateCredentials(1, "list.csv", []string{"user2"}, []string{"new"})
	if err != nil {
		t.Fatalf("UpdateCredentials failed: %v", err)
	}
	if updated != 1 {
		t.Fatalf("expected 1 updated record, got %d", updated)
	}
	data, _ := os.ReadFile("submissions/pcrs/1/list.csv")
	if !strings.Contains(string(data), "user2,new") {
		t.Fatalf("credlist not updated: %s", string(data))
	}
}

func TestGetCredlists(t *testing.T) {
	cleanup := withTeams(t, []db.TeamSchema{{ID: 1}})
	defer cleanup()

	tmp := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmp)
	defer os.Chdir(oldWd)

	os.MkdirAll("config/credlists", 0o755)
	content := "user1,pass1\n"
	os.WriteFile("config/credlists/list.csv", []byte(content), 0o644)

	conf := &config.ConfigSettings{
		CredlistSettings: config.CredlistConfig{
			Credlist: []config.Credlist{{CredlistName: "list", CredlistPath: "list.csv", CredlistExplainText: "username,password"}},
		},
	}
	se := &ScoringEngine{Config: conf, CredentialsMutex: map[uint]*sync.Mutex{1: {}}}

	v, err := se.GetCredlists()
	if err != nil {
		t.Fatalf("GetCredlists failed: %v", err)
	}
	lists := v.([]any)
	if len(lists) != 1 {
		t.Fatalf("expected 1 credlist, got %d", len(lists))
	}
	val := reflect.ValueOf(lists[0])
	name := val.FieldByName("Name").String()
	if name != "list" {
		t.Fatalf("unexpected name: %v", name)
	}
}
