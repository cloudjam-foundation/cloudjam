package observation

type Reading struct {
	ObservationID string  `json:"observation_id"`
	Habitat       string  `json:"habitat"`
	HumidityPct   float64 `json:"humidity_pct"`
	CO2PPM        int     `json:"co2_ppm"`
	RecordedAt    string  `json:"recorded_at"`
	Condition     string  `json:"condition,omitempty"`
}

type Limits struct {
	MinimumHumidityPct float64 `json:"minimum_humidity_pct"`
	MaximumHumidityPct float64 `json:"maximum_humidity_pct"`
	MaximumCO2PPM      int     `json:"maximum_co2_ppm"`
}
