package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"quotient/engine/db"
	"slices"
	"strconv"
	"time"
)

func GetCredlists(w http.ResponseWriter, r *http.Request) {
	req_roles := r.Context().Value("roles").([]string)
	if !slices.Contains(req_roles, "admin") && !conf.MiscSettings.EasyPCR {
		w.WriteHeader(http.StatusForbidden)
		data := map[string]any{"error": "PCR self service not allowed"}
		d, _ := json.Marshal(data)
		w.Write(d)
		return
	}

	credlists, err := eng.GetCredlists()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		data := map[string]any{"error": "Error getting credlists"}
		d, _ := json.Marshal(data)
		w.Write(d)
		slog.Error("Error getting credlists", "request_id", r.Context().Value("request_id"), "error", err.Error())
		return
	}

	d, _ := json.Marshal(credlists)
	w.Write(d)
}

func GetPcrs(w http.ResponseWriter, r *http.Request) {
	// Placeholder for future team PCR functionality
	w.WriteHeader(http.StatusNotImplemented)
}

func AdminGetPcrs(w http.ResponseWriter, r *http.Request) {
	req_roles := r.Context().Value("roles").([]string)
	if !slices.Contains(req_roles, "admin") {
		w.WriteHeader(http.StatusForbidden)
		data := map[string]any{"error": "Forbidden"}
		d, _ := json.Marshal(data)
		w.Write(d)
		return
	}

	teams, err := db.GetTeams()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		d, _ := json.Marshal(map[string]any{"error": "Error getting teams"})
		w.Write(d)
		return
	}

	type pcrInfo struct {
		TeamID   uint   `json:"team_id"`
		TeamName string `json:"team_name"`
		Credlist string `json:"credlist"`
		Updated  string `json:"updated"`
	}

	var data []pcrInfo

	for _, team := range teams {
		for _, cred := range conf.CredlistSettings.Credlist {
			filePath := filepath.Join("submissions/pcrs", fmt.Sprint(team.ID), cred.CredlistPath)
			fi, err := os.Stat(filePath)
			if err != nil {
				continue
			}
			data = append(data, pcrInfo{
				TeamID:   team.ID,
				TeamName: team.Name,
				Credlist: cred.CredlistPath,
				Updated:  fi.ModTime().Format(time.RFC3339),
			})
		}
	}

	d, _ := json.Marshal(data)
	w.Write(d)
}

func CreatePcr(w http.ResponseWriter, r *http.Request) {
	// get teamid from request
	// get username,password from request
	// somehow determine which credlist to change
	type Form struct {
		TeamID       string   `json:"team_id"`
		CredlistPath string   `json:"credlist_id"`
		Usernames    []string `json:"usernames"`
		Passwords    []string `json:"passwords"`
	}

	var form Form

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		slog.Error("Failed to decode PCR json", "request_id", r.Context().Value("request_id"), "error", err.Error())
		return
	}

	req_roles := r.Context().Value("roles").([]string)
	if !slices.Contains(req_roles, "admin") {
		if conf.MiscSettings.EasyPCR {
			me, err := db.GetTeamByUsername(r.Context().Value("username").(string))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if form.TeamID != fmt.Sprint(me.ID) {
				w.WriteHeader(http.StatusForbidden)
				data := map[string]any{"error": "PCR not allowed"}
				d, _ := json.Marshal(data)
				w.Write(d)
				return
			}
		} else {
			w.WriteHeader(http.StatusForbidden)
			data := map[string]any{"error": "PCR not allowed"}
			d, _ := json.Marshal(data)
			w.Write(d)
			return
		}
	}

	id, err := strconv.Atoi(form.TeamID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	updatedCount, err := eng.UpdateCredentials(uint(id), form.CredlistPath, form.Usernames, form.Passwords)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		data := map[string]any{"error": "Error updating PCR"}
		d, _ := json.Marshal(data)
		w.Write(d)
		slog.Error("Error updating PCR", "request_id", r.Context().Value("request_id"), "error", err.Error())
		return
	}

	data := map[string]any{
		"message": "PCR updated successfully",
		"count":   updatedCount,
	}
	d, _ := json.Marshal(data)
	w.Write(d)
}

func DownloadPcrFile(w http.ResponseWriter, r *http.Request) {
	req_roles := r.Context().Value("roles").([]string)
	if !slices.Contains(req_roles, "admin") {
		w.WriteHeader(http.StatusForbidden)
		data := map[string]any{"error": "Forbidden"}
		d, _ := json.Marshal(data)
		w.Write(d)
		return
	}

	teamStr := r.PathValue("team")
	fileName := r.PathValue("file")
	teamID, err := strconv.Atoi(teamStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filePath := filepath.Join("submissions/pcrs", fmt.Sprint(teamID), fileName)
	if !PathIsInDir("submissions/pcrs", filePath) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	file, err := os.Open(filePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	if _, err := io.Copy(w, file); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
