package engine

import (
	"testing"
	"time"

	"github.com/dbaseqp/Quotient/tests/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDivisorRotationTime(t *testing.T) {
	for numIPs, want := range map[string]time.Duration{
		"":     0,
		"zero": 0,
		"0":    0,
		"-3":   0,
		"5":    5*time.Second + divisorRotationMargin,
		"10":   10*time.Second + divisorRotationMargin,
	} {
		assert.Equal(t, want, divisorRotationTime(numIPs), "NUM_IPS=%q", numIPs)
	}
}

// An overrun round must still be followed by Divisor's rotation time before the next round starts.
func TestStart_OverrunRoundIsFollowedBySettleGap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	redis, pg := testutil.StartContainers(t)
	se := NewTestEngine(t, redis, pg, 3)
	t.Setenv("NUM_IPS", "4")
	want := 4*time.Second + divisorRotationMargin

	var starts []time.Time
	var firstEnd time.Time
	second := make(chan struct{})
	se.roundFunc = func() error {
		starts = append(starts, time.Now())
		switch len(starts) {
		case 1:
			// Collection ran past the next round's start, as with a missing result.
			se.NextRoundStartTime = time.Now().Add(-2 * time.Second)
			firstEnd = time.Now()
		case 2:
			close(second)
		}
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		se.Start()
	}()

	select {
	case <-second:
	case <-time.After(want + 15*time.Second):
		t.Fatal("second round never ran")
	}
	require.NoError(t, se.ResetScores())
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start did not return after ResetScores")
	}

	gap := starts[1].Sub(firstEnd)
	require.GreaterOrEqual(t, gap, want-50*time.Millisecond,
		"round after an overrun started %s after the previous round ended; want ≥ %s", gap, want)
}
