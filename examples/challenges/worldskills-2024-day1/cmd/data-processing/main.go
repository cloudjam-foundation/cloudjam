package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"time"

	"codeberg.org/megakuul/cloudjam/examples/challenges/worldskills-2024-day1/internal/appconfig"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	_ "github.com/lib/pq"
)

type processor struct {
	database        *sql.DB
	dynamodb        *dynamodb.Client
	ssm             *ssm.Client
	table           string
	statusParameter string
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

	app := &processor{
		database:        database,
		dynamodb:        dynamodb.NewFromConfig(awsConfig),
		ssm:             ssm.NewFromConfig(awsConfig),
		table:           settings.Value("feedback_table", "FeedbackTable"),
		statusParameter: settings.Value("batch_status_parameter", "/worldskills/day1/batch-success"),
	}
	for {
		processed, err := app.run(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("processed %d records", processed)
		if settings.Bool("run_once") {
			return
		}
		time.Sleep(interval)
	}
}

func (p *processor) run(ctx context.Context) (int, error) {
	if _, err := p.database.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS DataProcessingAPP (
			id SERIAL PRIMARY KEY,
			message VARCHAR(255)
		);
		ALTER TABLE DataProcessingAPP ADD COLUMN IF NOT EXISTS source_id VARCHAR(255);
		CREATE UNIQUE INDEX IF NOT EXISTS dataprocessingapp_source_id ON DataProcessingAPP(source_id);
	`); err != nil {
		return 0, fmt.Errorf("prepare database: %w", err)
	}

	processed := 0
	paginator := dynamodb.NewScanPaginator(p.dynamodb, &dynamodb.ScanInput{TableName: aws.String(p.table)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, fmt.Errorf("scan %s: %w", p.table, err)
		}
		for _, item := range page.Items {
			id, idOK := item["id"].(*ddbtypes.AttributeValueMemberS)
			message, messageOK := item["message"].(*ddbtypes.AttributeValueMemberS)
			if !idOK || !messageOK {
				continue
			}
			result, err := p.database.ExecContext(ctx,
				"INSERT INTO DataProcessingAPP (message, source_id) VALUES ($1, $2) ON CONFLICT (source_id) DO NOTHING",
				message.Value, id.Value)
			if err != nil {
				return 0, fmt.Errorf("store %s: %w", id.Value, err)
			}
			if rows, err := result.RowsAffected(); err == nil {
				processed += int(rows)
			}
		}
	}
	_, err := p.ssm.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(p.statusParameter),
		Overwrite: aws.Bool(true),
		Tier:      ssmtypes.ParameterTierIntelligentTiering,
		Type:      ssmtypes.ParameterTypeString,
		Value:     aws.String("success"),
	})
	if err != nil {
		return 0, fmt.Errorf("record batch status: %w", err)
	}
	return processed, nil
}
