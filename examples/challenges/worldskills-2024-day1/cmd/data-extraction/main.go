package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/worldskills-2024-day1/internal/appconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	_ "github.com/lib/pq"
)

var invalidParameterCharacter = regexp.MustCompile(`[^A-Za-z0-9_.-]`)

type extractor struct {
	database *sql.DB
	ssm      *ssm.Client
	prefix   string
}

func main() {
	configPath := flag.String("config", "config.ini", "configuration file")
	flag.Parse()

	settings, err := appconfig.Load(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	databaseURL, err := settings.Required("database_url")
	if err != nil {
		log.Fatal(err)
	}
	interval, err := settings.Duration("poll_interval", 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	awsConfig, err := config.LoadDefaultConfig(ctx, config.WithRegion(settings.Value("region", "us-east-1")))
	if err != nil {
		log.Fatal(err)
	}
	database, err := sql.Open("postgres", databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	database.SetMaxOpenConns(2)

	app := &extractor{
		database: database,
		ssm:      ssm.NewFromConfig(awsConfig),
		prefix:   strings.TrimRight(settings.Value("result_prefix", "/worldskills/day1/results"), "/"),
	}
	for {
		extracted, err := app.run(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("extracted %d records", extracted)
		if settings.Bool("run_once") {
			return
		}
		time.Sleep(interval)
	}
}

func (e *extractor) run(ctx context.Context) (int, error) {
	rows, err := e.database.QueryContext(ctx,
		"SELECT source_id, message FROM DataProcessingAPP WHERE source_id IS NOT NULL ORDER BY id")
	if err != nil {
		return 0, fmt.Errorf("query processed data: %w", err)
	}
	defer rows.Close()

	extracted := 0
	for rows.Next() {
		var id, message string
		if err := rows.Scan(&id, &message); err != nil {
			return 0, fmt.Errorf("read processed data: %w", err)
		}
		name := e.prefix + "/" + invalidParameterCharacter.ReplaceAllString(id, "_")
		if _, err := e.ssm.PutParameter(ctx, &ssm.PutParameterInput{
			Name:      aws.String(name),
			Overwrite: aws.Bool(true),
			Tier:      types.ParameterTierIntelligentTiering,
			Type:      types.ParameterTypeString,
			Value:     aws.String(message),
		}); err != nil {
			return 0, fmt.Errorf("write %s: %w", name, err)
		}
		extracted++
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return extracted, nil
}
