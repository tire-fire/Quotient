package engine

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestPauseResumeEngine(t *testing.T) {
	se := &ScoringEngine{EnginePauseWg: &sync.WaitGroup{}}
	se.PauseEngine()
	if !se.IsEnginePaused {
		t.Fatal("engine should be paused")
	}
	se.ResumeEngine()
	if se.IsEnginePaused {
		t.Fatal("engine should be resumed")
	}
}

func TestGetActiveTasks(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	se := &ScoringEngine{RedisClient: client}

	ctx := context.Background()
	task1, _ := json.Marshal(map[string]any{"runner_id": "r1", "status_text": "running"})
	task2, _ := json.Marshal(map[string]any{"runner_id": "r2", "status_text": "failed"})
	client.Set(ctx, "task:1", task1, time.Minute)
	client.Set(ctx, "task:2", task2, time.Minute)

	tasks, err := se.GetActiveTasks()
	if err != nil {
		t.Fatalf("GetActiveTasks error: %v", err)
	}
	if len(tasks["running"].([]any)) != 1 || len(tasks["failed"].([]any)) != 1 {
		t.Fatalf("unexpected task counts: %v", tasks)
	}
}
