package main

import (
	"fmt"
	"testing"

	"quotient/engine"
)

func TestCreateRunner(t *testing.T) {
	cases := []struct {
		typ    string
		expect string
	}{
		{"Ping", "*checks.Ping"},
		{"Custom", "*checks.Custom"},
		{"unknown", ""},
	}

	for _, c := range cases {
		task := &engine.Task{ServiceType: c.typ, CheckData: []byte("{}")}
		r, err := createRunner(task)
		if c.expect == "" {
			if err == nil {
				t.Fatalf("expected error for %s", c.typ)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if typ := fmt.Sprintf("%T", r); typ != c.expect {
			t.Fatalf("want %s got %s", c.expect, typ)
		}
	}
}
