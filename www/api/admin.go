package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"quotient/engine/db"
)

func PauseEngine(w http.ResponseWriter, r *http.Request) {
	type Form struct {
		Pause bool `json:"pause"`
	}

	var form Form
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	slog.Debug("pause engine requested", "wasEnginePausedWhenRequestIssued", eng.IsEnginePaused, "setPauseTo", form.Pause)
	if eng.IsEnginePaused && !form.Pause {
		eng.ResumeEngine()
	} else if !eng.IsEnginePaused && form.Pause {
		eng.PauseEngine()
	} else {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	d := []byte(`{"status": "success"}`)
	w.Write(d)
}

func ResetScores(w http.ResponseWriter, r *http.Request) {
	slog.Debug("reset scores requested")
	if err := eng.ResetScores(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d := []byte(`{"status": "success"}`)
	w.Write(d)
}

func ExportScores(w http.ResponseWriter, r *http.Request) {
	type TeamScore struct {
		TeamID        uint   `json:"team_id"`
		TeamName      string `json:"team_name"`
		ServicePoints int    `json:"service_points"`
		SlaViolations int    `json:"sla_violations"`
		TotalPoints   int    `json:"total_points"`
	}
	var data []TeamScore

	teams, err := db.GetTeams()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, team := range teams {
		servicePoints, slaCount, slaTotal, err := db.GetTeamScore(team.ID)
		if err != nil {
			slog.Error("failed to get team score", "team_id", team.ID, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		teamScore := TeamScore{
			TeamID:        team.ID,
			TeamName:      team.Name,
			ServicePoints: servicePoints,
			SlaViolations: slaCount,
			TotalPoints:   servicePoints - slaTotal,
		}

		data = append(data, teamScore)
	}

	d, _ := json.Marshal(data)
	w.Write(d)
}

func ExportConfig(w http.ResponseWriter, r *http.Request) {

}

func GetActiveTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := eng.GetActiveTasks()
	if err != nil {
		slog.Error("failed to get active tasks", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d, _ := json.Marshal(tasks)
	w.Write(d)
}

func GetEngine(w http.ResponseWriter, r *http.Request) {
	lastRound, err := db.GetLastRound()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	d, _ := json.Marshal(map[string]any{
		"last_round":         lastRound,
		"current_round_time": eng.CurrentRoundStartTime,
		"next_round_time":    eng.NextRoundStartTime,
		"start_time":         eng.Config.MiscSettings.StartTime,
		"stop_time":          eng.Config.MiscSettings.StopTime,
		"running":            !eng.IsEnginePaused,
	})
	w.Write(d)
}

func UpdateTeams(w http.ResponseWriter, r *http.Request) {
	type Form struct {
		Teams []struct {
			TeamID     int    `json:"id"`
			Identifier string `json:"identifier"`
			Active     bool   `json:"active"`
		} `json:"teams"`
	}

	var form Form
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, team := range form.Teams {
		if err := db.UpdateTeam(uint(team.TeamID), team.Identifier, team.Active); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	d := []byte(`{"status": "success"}`)
	w.Write(d)
}

func UpdateStartTime(w http.ResponseWriter, r *http.Request) {
	type Form struct {
		StartTime string `json:"start_time"`
	}

	var form Form
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t, err := time.Parse(time.RFC3339, form.StartTime)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eng.SetStartTime(t)

	d := []byte(`{"status": "success"}`)
	w.Write(d)
}

func UpdateStopTime(w http.ResponseWriter, r *http.Request) {
	type Form struct {
		StopTime string `json:"stop_time"`
	}

	var form Form
	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	t, err := time.Parse(time.RFC3339, form.StopTime)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	eng.SetStopTime(t)

	d := []byte(`{"status": "success"}`)
	w.Write(d)
}
