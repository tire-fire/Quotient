package checks

import (
	"testing"
	"time"
)

func TestServiceRunTimeout(t *testing.T) {
	s := Service{Timeout: 1}
	ch := make(chan Result, 1)
	s.Run(1, "team", 1, ch, func(teamID uint, id string, res Result, resp chan Result) {
		time.Sleep(2 * time.Second)
		res.Status = true
		resp <- res
	})
	result := <-ch
	if result.Error != "check timeout exceeded" {
		t.Fatalf("expected timeout error, got %v", result.Error)
	}
}

func TestPingVerifyDefaults(t *testing.T) {
	p := &Ping{}
	if err := p.Verify("box", "1.1.1.1", 5, 2, 3, 4); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if p.ServiceType != "Ping" || p.Display != "ping" || p.Count != 1 {
		t.Fatalf("defaults not applied: %+v", p)
	}
}
