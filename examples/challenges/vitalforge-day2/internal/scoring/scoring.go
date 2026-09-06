package scoring

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
)

type Check struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Points float64 `json:"points"`
}

type Result struct {
	Data    map[string]any `json:"data,omitempty"`
	Check   Check          `json:"check"`
	OK      bool           `json:"ok"`
	Detail  string         `json:"detail"`
	Latency int64          `json:"latency_ms"`
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
	for _, check := range Checks {
		method, path, group, status, limit := http.MethodGet, "", "member", 200, int64(800)
		var body any
		fields := map[string]any{}
		switch check.ID {
		case "health":
			path, group = "/health", ""
			fields["status"], fields["service"] = "ok", "wellness-api"
		case "members-unauth":
			path, group, status = "/api/members", "", 401
		case "members-coach":
			path, group = "/api/members", "coach"
			fields["total"], fields["members"] = float64(3), "array"
		case "thresholds-version":
			path = "/api/config/thresholds/version"
			fields["version"] = "non-empty string"
		case "meal-create":
			method, path, status = http.MethodPost, "/api/meals", 201
			body = map[string]any{"user_id": "member-demo-1", "calories": 650, "protein_g": 42, "description": "CloudJam scoring meal"}
			fields["meal_id"], fields["user_id"], fields["calories"] = "non-empty string", "member-demo-1", float64(650)
		case "workout-create":
			method, path, status = http.MethodPost, "/api/workouts", 201
			body = map[string]any{"user_id": "member-demo-1", "duration_min": 45, "calories_burned": 320, "activity": "cardio"}
			fields["workout_id"], fields["user_id"], fields["calories_burned"] = "non-empty string", "member-demo-1", float64(320)
		case "dashboard-member":
			path, limit = "/api/members/member-demo-1/dashboard", 500
			fields["user_id"], fields["analytics.thresholds_version"] = "member-demo-1", "non-empty string"
			for _, name := range []string{"calorie_balance", "protein_progress_pct", "sleep_score"} {
				fields["analytics."+name] = "number"
			}
		case "dashboard-forbidden":
			path, status = "/api/members/member-demo-2/dashboard", 403
		case "sleep-ingest", "sleep-too-short":
			method, path, group, status, limit = http.MethodPost, "/sleep", "", 202, 1000
			duration := 8 * time.Hour
			if check.ID == "sleep-too-short" {
				duration, status = time.Hour, 422
			} else {
				fields["session_id"], fields["user_id"], fields["duration_hours"], fields["thresholds_version"] = "non-empty string", "member-demo-1", float64(8), "non-empty string"
			}
			ended := time.Now().UTC()
			body = map[string]any{"user_id": "member-demo-1", "started_at": ended.Add(-duration).Format(time.RFC3339), "ended_at": ended.Format(time.RFC3339)}
		case "export-queue":
			method, path, group, status = http.MethodPost, "/api/members/member-demo-1/export", "coach", 202
			body = map[string]any{}
			fields["user_id"], fields["status"] = "member-demo-1", "queued"
		case "export-latest":
			path = "/api/export/member-demo-1/latest"
			fields["user_id"], fields["generated_at"], fields["thresholds_version"] = "member-demo-1", "non-empty string", "non-empty string"
			for _, name := range []string{"meals_count", "workouts_count", "avg_sleep_hours"} {
				fields[name] = "number"
			}
		}
		result := Result{Check: check}
		if err := ctx.Err(); err != nil {
			result.Detail = "FAIL | " + err.Error() + "; no request sent"
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
			subject := "member-demo-1"
			if group == "coach" {
				subject = "scoring-coach"
			}
			request.Headers["X-Dev-Sub"] = []string{subject}
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
			if check.ID != "export-latest" || err != nil || response.StatusCode != 404 || attempt >= 9 {
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
				var value any = data
				for _, part := range strings.Split(field, ".") {
					object, ok := value.(map[string]any)
					if !ok {
						value = nil
						break
					}
					value = object[part]
				}
				valid := false
				switch want {
				case "number":
					_, valid = value.(float64)
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
		result.OK = detail == "ok"
		if result.OK {
			result.Data = data
		}
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
