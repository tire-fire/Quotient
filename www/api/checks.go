package api

import (
	"encoding/json"
	"net/http"
	"quotient/engine/db"
)

type checkInfo struct {
	Box        string        `json:"box"`
	Name       string        `json:"name"`
	Type       string        `json:"type"`
	Enabled    bool          `json:"enabled"`
	LastResult map[uint]bool `json:"last_results"`
}

func GetChecks(w http.ResponseWriter, r *http.Request) {
	lastRound, err := db.GetLastRound()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	results := make(map[string]map[uint]bool)
	for _, check := range lastRound.Checks {
		if results[check.ServiceName] == nil {
			results[check.ServiceName] = make(map[uint]bool)
		}
		results[check.ServiceName][check.TeamID] = check.Result
	}

	var data []checkInfo
	for _, box := range conf.Box {
		for _, r := range box.Runners {
			info := checkInfo{
				Box:        box.Name,
				Name:       r.GetName(),
				Type:       r.GetType(),
				Enabled:    r.Runnable(),
				LastResult: results[r.GetName()],
			}
			data = append(data, info)
		}
	}

	d, _ := json.Marshal(data)
	w.Write(d)
}
