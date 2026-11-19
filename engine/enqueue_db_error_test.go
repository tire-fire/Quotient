package engine

import (
	"testing"

	"github.com/dbaseqp/Quotient/engine/checks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A DB read that fails while a round enqueues its checks (Postgres restarting at round start) must
// fail the round so the loop retries it under the same ID. Skipping the affected checks stored the
// round without them, so teams lost those checks' points for an infrastructure blip.
func TestRvb_EnqueueReadFailureRetriesRound(t *testing.T) {
	se, redis, pg, teams := collectSetup(t, 2, "web")
	collectRunner(t, redis, func(task Task, push func(checks.Result)) { push(passed(task)) })

	pg.Exec(t, "ALTER TABLE team_service_check_schemas RENAME TO team_service_check_schemas_away")
	t.Cleanup(func() {
		pg.Exec(t, `DO $$ BEGIN
			IF to_regclass('team_service_check_schemas_away') IS NOT NULL THEN
				ALTER TABLE team_service_check_schemas_away RENAME TO team_service_check_schemas;
			END IF; END $$`)
	})

	require.Error(t, se.rvb(), "a failed read while enqueueing must fail the round")
	assert.Equal(t, uint(1), se.CurrentRound, "the round ID is not used up")
	round, err := pg.DB.GetLastRound()
	require.NoError(t, err)
	assert.Zero(t, round.ID, "nothing is stored for the failed attempt")

	pg.Exec(t, "ALTER TABLE team_service_check_schemas_away RENAME TO team_service_check_schemas")
	require.NoError(t, se.rvb())
	round, mine := lastRoundChecks(t, pg, teams[1].ID)
	assert.Equal(t, uint(1), round.ID, "the retry stores the same round ID")
	assert.Len(t, round.Checks, 2, "with every team's check")
	require.Len(t, mine, 1)
	assert.True(t, mine[0].Result)
}
