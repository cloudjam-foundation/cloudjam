package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/borealis-field-observatory/internal/observation"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/appconfigdata"
	"github.com/aws/aws-sdk-go-v2/service/athena"
	athenatypes "github.com/aws/aws-sdk-go-v2/service/athena/types"
	"github.com/aws/aws-sdk-go-v2/service/iotdataplane"
)

var observationID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

type server struct {
	iot       *iotdataplane.Client
	appconfig *appconfigdata.Client
	athena    *athena.Client

	topic         string
	application   string
	environment   string
	configuration string
	database      string
	table         string
	workgroup     string

	limitsLock sync.Mutex
	limits     observation.Limits
	limitsRead time.Time
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	iotEndpoint := required("IOT_ENDPOINT")
	s := &server{
		iot: iotdataplane.NewFromConfig(cfg, func(options *iotdataplane.Options) {
			options.BaseEndpoint = aws.String("https://" + strings.TrimPrefix(iotEndpoint, "https://"))
		}),
		appconfig:     appconfigdata.NewFromConfig(cfg),
		athena:        athena.NewFromConfig(cfg),
		topic:         required("IOT_TOPIC"),
		application:   required("APPCONFIG_APPLICATION"),
		environment:   required("APPCONFIG_ENVIRONMENT"),
		configuration: required("APPCONFIG_PROFILE"),
		database:      required("ATHENA_DATABASE"),
		table:         required("ATHENA_TABLE"),
		workgroup:     required("ATHENA_WORKGROUP"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("POST /observations", s.submit)
	mux.HandleFunc("GET /observations/{observation_id}", s.find)
	log.Printf("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func (s *server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *server) submit(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var reading observation.Reading
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&reading); err != nil {
		writeError(w, http.StatusBadRequest, "invalid observation")
		return
	}
	if !valid(reading) {
		writeError(w, http.StatusBadRequest, "invalid observation")
		return
	}

	limits, err := s.readLimits(r.Context())
	if err != nil {
		log.Printf("read AppConfig: %v", err)
		writeError(w, http.StatusServiceUnavailable, "configuration unavailable")
		return
	}
	reading.Condition = "nominal"
	if reading.HumidityPct < limits.MinimumHumidityPct || reading.HumidityPct > limits.MaximumHumidityPct || reading.CO2PPM > limits.MaximumCO2PPM {
		reading.Condition = "attention"
	}
	payload, err := json.Marshal(reading)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encode observation")
		return
	}
	if _, err := s.iot.Publish(r.Context(), &iotdataplane.PublishInput{
		Payload: payload,
		Qos:     1,
		Topic:   aws.String(s.topic),
	}); err != nil {
		log.Printf("publish observation: %v", err)
		writeError(w, http.StatusBadGateway, "publish failed")
		return
	}
	writeJSON(w, http.StatusAccepted, reading)
}

func (s *server) find(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("observation_id")
	if !observationID.MatchString(id) {
		writeError(w, http.StatusBadRequest, "invalid observation id")
		return
	}
	reading, found, err := s.query(r.Context(), id)
	if err != nil {
		log.Printf("query observation: %v", err)
		writeError(w, http.StatusBadGateway, "query failed")
		return
	}
	if !found {
		writeError(w, http.StatusNotFound, "observation not found")
		return
	}
	writeJSON(w, http.StatusOK, reading)
}

func (s *server) readLimits(ctx context.Context) (observation.Limits, error) {
	s.limitsLock.Lock()
	defer s.limitsLock.Unlock()
	if time.Since(s.limitsRead) < time.Minute {
		return s.limits, nil
	}
	session, err := s.appconfig.StartConfigurationSession(ctx, &appconfigdata.StartConfigurationSessionInput{
		ApplicationIdentifier:                aws.String(s.application),
		ConfigurationProfileIdentifier:       aws.String(s.configuration),
		EnvironmentIdentifier:                aws.String(s.environment),
		RequiredMinimumPollIntervalInSeconds: aws.Int32(60),
	})
	if err != nil {
		return observation.Limits{}, err
	}
	configuration, err := s.appconfig.GetLatestConfiguration(ctx, &appconfigdata.GetLatestConfigurationInput{
		ConfigurationToken: session.InitialConfigurationToken,
	})
	if err != nil {
		return observation.Limits{}, err
	}
	if err := json.Unmarshal(configuration.Configuration, &s.limits); err != nil {
		return observation.Limits{}, err
	}
	if s.limits.MinimumHumidityPct >= s.limits.MaximumHumidityPct || s.limits.MaximumCO2PPM <= 0 {
		return observation.Limits{}, errors.New("invalid limits")
	}
	s.limitsRead = time.Now()
	return s.limits, nil
}

func (s *server) query(ctx context.Context, id string) (observation.Reading, bool, error) {
	query := fmt.Sprintf(`SELECT observation_id, habitat, humidity_pct, co2_ppm, recorded_at, condition FROM %s.%s WHERE observation_id = '%s' LIMIT 1`, s.database, s.table, id)
	started, err := s.athena.StartQueryExecution(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		WorkGroup:   aws.String(s.workgroup),
	})
	if err != nil {
		return observation.Reading{}, false, err
	}
	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		execution, err := s.athena.GetQueryExecution(ctx, &athena.GetQueryExecutionInput{QueryExecutionId: started.QueryExecutionId})
		if err != nil {
			return observation.Reading{}, false, err
		}
		state := execution.QueryExecution.Status.State
		switch state {
		case athenatypes.QueryExecutionStateSucceeded:
			return s.queryResult(ctx, started.QueryExecutionId)
		case athenatypes.QueryExecutionStateFailed, athenatypes.QueryExecutionStateCancelled:
			return observation.Reading{}, false, fmt.Errorf("Athena query %s: %s", state, aws.ToString(execution.QueryExecution.Status.StateChangeReason))
		}
		time.Sleep(500 * time.Millisecond)
	}
	return observation.Reading{}, false, errors.New("Athena query timed out")
}

func (s *server) queryResult(ctx context.Context, executionID *string) (observation.Reading, bool, error) {
	result, err := s.athena.GetQueryResults(ctx, &athena.GetQueryResultsInput{QueryExecutionId: executionID})
	if err != nil {
		return observation.Reading{}, false, err
	}
	if len(result.ResultSet.Rows) < 2 || len(result.ResultSet.Rows[1].Data) < 6 {
		return observation.Reading{}, false, nil
	}
	data := result.ResultSet.Rows[1].Data
	humidity, err := strconv.ParseFloat(aws.ToString(data[2].VarCharValue), 64)
	if err != nil {
		return observation.Reading{}, false, err
	}
	co2, err := strconv.Atoi(aws.ToString(data[3].VarCharValue))
	if err != nil {
		return observation.Reading{}, false, err
	}
	return observation.Reading{
		ObservationID: aws.ToString(data[0].VarCharValue),
		Habitat:       aws.ToString(data[1].VarCharValue),
		HumidityPct:   humidity,
		CO2PPM:        co2,
		RecordedAt:    aws.ToString(data[4].VarCharValue),
		Condition:     aws.ToString(data[5].VarCharValue),
	}, true, nil
}

func valid(reading observation.Reading) bool {
	if !observationID.MatchString(reading.ObservationID) || strings.TrimSpace(reading.Habitat) == "" {
		return false
	}
	if reading.HumidityPct < 0 || reading.HumidityPct > 100 || reading.CO2PPM < 0 {
		return false
	}
	_, err := time.Parse(time.RFC3339, reading.RecordedAt)
	return err == nil
}

func required(name string) string {
	value := os.Getenv(name)
	if value == "" {
		log.Fatalf("%s is required", name)
	}
	return value
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}
