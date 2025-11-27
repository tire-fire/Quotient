package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertEventually asserts that a condition becomes true within a timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	require.Fail(t, "condition not met within timeout", msgAndArgs...)
}

// AssertNever asserts that a condition never becomes true within a timeout
func AssertNever(t *testing.T, condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			require.Fail(t, "condition became true when it should not", msgAndArgs...)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// AssertRedisKeyExists checks if a Redis key exists
func AssertRedisKeyExists(t *testing.T, container *RedisContainer, key string) {
	t.Helper()
	ctx := context.Background()
	exists, err := container.Client.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Greater(t, exists, int64(0), "expected Redis key %s to exist", key)
}

// AssertRedisKeyNotExists checks if a Redis key does not exist
func AssertRedisKeyNotExists(t *testing.T, container *RedisContainer, key string) {
	t.Helper()
	ctx := context.Background()
	exists, err := container.Client.Exists(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(0), exists, "expected Redis key %s to not exist", key)
}

// AssertQueueLength checks the length of a Redis list
func AssertQueueLength(t *testing.T, container *RedisContainer, queue string, expectedLen int64) {
	t.Helper()
	ctx := context.Background()
	length, err := container.Client.LLen(ctx, queue).Result()
	require.NoError(t, err)
	assert.Equal(t, expectedLen, length, "expected queue %s to have length %d, got %d", queue, expectedLen, length)
}

// AssertScoreMonotonicity ensures scores are non-decreasing
func AssertScoreMonotonicity(t *testing.T, scores []int) {
	t.Helper()
	for i := 1; i < len(scores); i++ {
		if scores[i] < scores[i-1] {
			require.Fail(t, "score decreased", "score at index %d (%d) is less than previous score (%d)", i, scores[i], scores[i-1])
		}
	}
}

// AssertValidTaskResult validates that a result matches its task
func AssertValidTaskResult(t *testing.T, taskTeamID uint, taskServiceName string, resultTeamID uint, resultServiceName string) {
	t.Helper()
	assert.Equal(t, taskTeamID, resultTeamID, "result team ID does not match task")
	assert.Equal(t, taskServiceName, resultServiceName, "result service name does not match task")
}
