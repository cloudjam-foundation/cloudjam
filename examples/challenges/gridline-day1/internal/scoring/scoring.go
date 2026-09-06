package scoring

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
	"github.com/google/uuid"
)

type Check struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Points float64 `json:"points"`
}

type Result struct {
	Check   Check  `json:"check"`
	OK      bool   `json:"ok"`
	Detail  string `json:"detail"`
	Latency int64  `json:"latency_ms"`
	HeatID  string `json:"heat_id,omitempty"`
}

//go:embed checks.json
var definitions []byte

var Checks = func() []Check {
	var checks []Check
	if err := json.Unmarshal(definitions, &checks); err != nil {
		panic(err)
	}
	return checks
}()

func Run(ctx context.Context, send func(api.SendHTTPInput) (api.SendHTTPOutput, error)) []Result {
	results := make([]Result, 0, len(Checks))
	heatID := ""
	for _, check := range Checks {
		method, path, group, status, limit := http.MethodGet, "", "spectator", 200, int64(500)
		var body any
		fields := map[string]any{}
		dependent := false
		switch check.ID {
		case "health":
			path, group, limit = "/health", "", 800
			fields["status"] = "ok"
		case "tracks-unauth":
			path, group, status = "/api/tracks", "", 401
		case "tracks-spectator":
			path = "/api/tracks"
			fields["total"] = float64(6)
		case "track-detail":
			path = "/api/tracks/track-01"
			fields["track_id"], fields["name"] = "track-01", "string"
		case "rules-version":
			path = "/api/config/rules/version"
			fields["version"] = "string"
		case "heats-spectator-forbidden", "create-heat":
			method, path, status = http.MethodPost, "/api/heats", 403
			body = map[string]any{"track_id": "track-01", "label": "Scoring " + uuid.NewString(), "duration_seconds": 60}
			if check.ID == "create-heat" {
				group, status = "director", 201
				fields["heat_id"] = "non-empty string"
			}
		case "transponder-not-running", "transponder-unknown":
			method, path, group, status, dependent = http.MethodPost, "/transponder", "", 422, true
			transponder := "T-0001"
			if check.ID == "transponder-unknown" {
				transponder = "T-9999"
			}
			body = map[string]any{"transponder_id": transponder, "track_id": "track-01", "heat_id": heatID, "line": "sf", "ts": time.Now().UTC().Format(time.RFC3339Nano)}
		case "start-heat":
			method, path, group, status, limit, dependent = http.MethodPost, "/api/heats/"+url.PathEscape(heatID)+"/start", "director", 202, 1000, true
			body = map[string]any{}
		case "heat-running":
			path, dependent = "/api/heats/"+url.PathEscape(heatID), true
			fields["status"] = "RUNNING"
		case "leaderboard":
			path, limit, dependent = "/api/heats/"+url.PathEscape(heatID)+"/leaderboard", 300, true
			fields["heat_id"], fields["standings"] = heatID, "array"
		case "heat-not-found":
			path, status = "/api/heats/"+uuid.NewString(), 404
			fields["error"] = "string"
		}
		result := Result{Check: check, HeatID: heatID}
		if dependent && heatID == "" {
			result.Detail = "FAIL | prerequisite create-heat failed; no request sent"
			results = append(results, result)
			continue
		}
		payload, _ := json.Marshal(body)
		if body == nil {
			payload = nil
		}
		request := api.SendHTTPInput{Method: method, URL: path, Body: payload, TimeoutMillis: 5000,
			Headers: map[string][]string{"Accept": {"application/json"}, "X-CloudJam-Check": {check.ID}}}
		if body != nil {
			request.Headers["Content-Type"] = []string{"application/json"}
		}
		if group != "" {
			request.Headers["X-Dev-Sub"] = []string{"scoring-" + group}
			request.Headers["X-Dev-Groups"] = []string{group}
		}
		var response api.SendHTTPOutput
		var data map[string]any
		var err error
		for attempt := 0; ; attempt++ {
			started := time.Now()
			response, err = send(request)
			result.Latency = time.Since(started).Milliseconds()
			data = nil
			if err == nil {
				_ = json.Unmarshal(response.Body, &data)
			}
			if check.ID != "heat-running" || err != nil || response.StatusCode != 200 || data["status"] != "SCHEDULED" || attempt >= 9 {
				break
			}
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-time.After(time.Second):
			}
			if err != nil {
				break
			}
		}
		detail := "ok"
		switch {
		case err != nil:
			detail = err.Error()
		case response.StatusCode != status:
			detail = fmt.Sprintf("status %d, want %d", response.StatusCode, status)
		default:
			for field, want := range fields {
				value := data[field]
				valid := false
				switch want {
				case "string":
					_, valid = value.(string)
				case "array":
					_, valid = value.([]any)
				case "non-empty string":
					text, ok := value.(string)
					valid = ok && text != ""
				default:
					valid = value == want
				}
				if !valid {
					detail = fmt.Sprintf("$.%s = %v, want %v", field, value, want)
					break
				}
			}
			if detail == "ok" && result.Latency > limit {
				detail = fmt.Sprintf("latency %dms > %dms", result.Latency, limit)
			}
		}
		if check.ID == "create-heat" && response.StatusCode == 201 {
			if id, ok := data["heat_id"].(string); ok && id != "" {
				heatID = id
				result.HeatID = id
			}
		}
		result.OK = detail == "ok"
		outcome := "FAIL"
		if result.OK {
			outcome = "PASS"
		}
		result.Detail = fmt.Sprintf("%s | %s %s | HTTP %d | %dms | %s", outcome, method, path, response.StatusCode, result.Latency, detail)
		if !result.OK && len(response.Body) != 0 {
			text := strings.Join(strings.Fields(string(response.Body)), " ")
			result.Detail += fmt.Sprintf(" | response: %.240s", text)
		}
		results = append(results, result)
	}
	return results
}
