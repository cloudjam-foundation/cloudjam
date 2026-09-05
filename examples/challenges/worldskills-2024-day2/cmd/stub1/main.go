package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const defaultTable = "cloudraiser-iot"

type server struct {
	dynamodb *dynamodb.Client
	table    string
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	table := os.Getenv("TABLE_NAME")
	if table == "" {
		table = defaultTable
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	s := &server{dynamodb: dynamodb.NewFromConfig(cfg), table: table}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("GET /get_value", s.getValue)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(httpServer.ListenAndServe())
}

func health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *server) getValue(w http.ResponseWriter, r *http.Request) {
	no := r.URL.Query().Get("no")
	if no == "" {
		http.Error(w, "missing no", http.StatusBadRequest)
		return
	}

	output, err := s.dynamodb.GetItem(r.Context(), &dynamodb.GetItemInput{
		TableName: aws.String(s.table),
		Key: map[string]types.AttributeValue{
			"No": &types.AttributeValueMemberS{Value: no},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if len(output.Item) == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	response := map[string]string{}
	for name, value := range output.Item {
		if stringValue, ok := value.(*types.AttributeValueMemberS); ok {
			response[name] = stringValue.Value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("write response: %v", err)
	}
}
