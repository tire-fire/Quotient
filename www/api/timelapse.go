package api

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"log/slog"
	"net/http"
	"slices"
	"sort"
	"strconv"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"

	"github.com/kettek/apng"

	"quotient/engine/db"
)

var (
	colorUp      = color.RGBA{0x3D, 0x99, 0x70, 0xFF}
	colorDown    = color.RGBA{0xAA, 0x07, 0x07, 0xFF}
	colorUnknown = color.RGBA{0x77, 0x77, 0x77, 0xFF}
	colorWhite   = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	colorBlack   = color.RGBA{0x00, 0x00, 0x00, 0xFF}
)

const (
	cellWidth   = 40
	cellHeight  = 30
	labelLeft   = 120 // width reserved for team name labels
	labelTop    = 60  // height reserved for service name labels + title
	titleHeight = 20  // height for the "Round N" title
	padding     = 10
)

func ExportTimelapse(w http.ResponseWriter, r *http.Request) {
	delayMs := 500
	if d := r.URL.Query().Get("delay"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 && parsed <= 10000 {
			delayMs = parsed
		}
	}

	rounds, err := db.GetAllRoundsWithChecks()
	if err != nil {
		slog.Error("failed to get rounds for timelapse", "error", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": "Failed to retrieve rounds"})
		return
	}

	if len(rounds) == 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "No rounds to generate timelapse"})
		return
	}

	teams, err := db.GetTeams()
	if err != nil {
		slog.Error("failed to get teams for timelapse", "error", err)
		WriteJSON(w, http.StatusInternalServerError, map[string]any{"error": "Failed to retrieve teams"})
		return
	}
	teams = slices.DeleteFunc(teams, func(team db.TeamSchema) bool { return !team.Active })

	if len(teams) == 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "No active teams"})
		return
	}

	// Collect unique service names across all rounds
	serviceSet := make(map[string]bool)
	for _, round := range rounds {
		for _, check := range round.Checks {
			serviceSet[check.ServiceName] = true
		}
	}
	services := make([]string, 0, len(serviceSet))
	for s := range serviceSet {
		services = append(services, s)
	}
	sort.Strings(services)

	if len(services) == 0 {
		WriteJSON(w, http.StatusBadRequest, map[string]any{"error": "No services found"})
		return
	}

	// Build APNG
	a := apng.APNG{
		Frames: make([]apng.Frame, len(rounds)),
	}

	for i, round := range rounds {
		img := renderHeatmapFrame(round, teams, services)
		delay := uint16(delayMs / 10) // APNG delay is in 1/100ths of a second
		if i == len(rounds)-1 {
			delay *= 3 // last frame holds 3x longer
		}
		a.Frames[i] = apng.Frame{
			Image:            img,
			DelayNumerator:   delay,
			DelayDenominator: 100,
		}
	}

	w.Header().Set("Content-Type", "image/apng")
	w.Header().Set("Content-Disposition", `attachment; filename="timelapse.png"`)

	if err := apng.Encode(w, a); err != nil {
		slog.Error("failed to encode timelapse APNG", "error", err)
	}
}

func renderHeatmapFrame(round db.RoundSchema, teams []db.TeamSchema, services []string) *image.RGBA {
	imgWidth := labelLeft + len(services)*cellWidth + padding
	imgHeight := labelTop + len(teams)*cellHeight + padding

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	// Fill background white
	draw.Draw(img, img.Bounds(), &image.Uniform{colorWhite}, image.Point{}, draw.Src)

	// Draw title "Round N"
	title := fmt.Sprintf("Round %d", round.ID)
	drawString(img, padding, titleHeight, title, colorBlack)

	// Draw service name labels across the top
	for j, svc := range services {
		x := labelLeft + j*cellWidth + 2
		y := labelTop - 5
		drawString(img, x, y, svc, colorBlack)
	}

	// Build lookup: teamID -> serviceName -> result
	checkMap := make(map[uint]map[string]bool)
	checkExists := make(map[uint]map[string]bool)
	for _, check := range round.Checks {
		if checkMap[check.TeamID] == nil {
			checkMap[check.TeamID] = make(map[string]bool)
			checkExists[check.TeamID] = make(map[string]bool)
		}
		checkMap[check.TeamID][check.ServiceName] = check.Result
		checkExists[check.TeamID][check.ServiceName] = true
	}

	// Draw rows
	for i, team := range teams {
		// Team name label
		y := labelTop + i*cellHeight + cellHeight/2 + 5
		drawString(img, padding, y, team.Name, colorBlack)

		// Cells
		for j, svc := range services {
			cellX := labelLeft + j*cellWidth
			cellY := labelTop + i*cellHeight

			var c color.RGBA
			if checkExists[team.ID] != nil && checkExists[team.ID][svc] {
				if checkMap[team.ID][svc] {
					c = colorUp
				} else {
					c = colorDown
				}
			} else {
				c = colorUnknown
			}

			rect := image.Rect(cellX, cellY, cellX+cellWidth-1, cellY+cellHeight-1)
			draw.Draw(img, rect, &image.Uniform{c}, image.Point{}, draw.Src)
		}
	}

	return img
}

func drawString(img *image.RGBA, x, y int, s string, col color.RGBA) {
	d := &font.Drawer{
		Dst:  img,
		Src:  &image.Uniform{col},
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}
	d.DrawString(s)
}
