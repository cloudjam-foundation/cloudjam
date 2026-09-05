package telemetry

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var shipmentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,63}$`)

type Reading struct {
	ShipmentID   string  `json:"shipment_id"`
	Sequence     int64   `json:"sequence"`
	TemperatureC float64 `json:"temperature_c"`
	RecordedAt   string  `json:"recorded_at"`
}

func (r Reading) Validate() error {
	if !shipmentIDPattern.MatchString(r.ShipmentID) {
		return fmt.Errorf("shipment_id must contain 1-64 letters, numbers, or hyphens")
	}
	if r.Sequence <= 0 {
		return fmt.Errorf("sequence must be positive")
	}
	if r.TemperatureC < -80 || r.TemperatureC > 80 {
		return fmt.Errorf("temperature_c must be between -80 and 80")
	}
	if _, err := time.Parse(time.RFC3339, r.RecordedAt); err != nil {
		return fmt.Errorf("recorded_at must be RFC3339: %w", err)
	}
	return nil
}

func (r Reading) Key() string {
	return "processed/" + r.ShipmentID + "/" + strconv.FormatInt(r.Sequence, 10) + ".json"
}

type Result struct {
	Reading
	Status      string `json:"status"`
	ProcessedAt string `json:"processed_at"`
}
