//go:generate go run .

package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"time"
)

type artifact struct {
	name         string
	binary       string
	packagePath  string
	architecture string
	files        map[string]string
}

func main() {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		panic(err)
	}
	exampleRoot := filepath.Join(moduleRoot, "examples", "challenges", "helios-cold-chain")
	temporary, err := os.MkdirTemp("", "helios-cold-chain-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temporary)

	artifacts := []artifact{
		{
			name:         "helios-ingest.zip",
			binary:       "bootstrap",
			packagePath:  "ingest",
			architecture: "arm64",
			files: map[string]string{
				"README.md": "# Helios ingestion function\n\nDeploy `bootstrap` with the `provided.al2023` runtime on arm64. Set `EVENT_BUS_NAME` to the custom event bus name and expose a Function URL.\n",
			},
		},
		{
			name:         "helios-relay.zip",
			binary:       "bootstrap",
			packagePath:  "relay",
			architecture: "arm64",
			files: map[string]string{
				"README.md": "# Helios EventBridge relay\n\nDeploy `bootstrap` with the `provided.al2023` runtime on arm64. Set `QUEUE_URL` to the telemetry queue and configure the function as the EventBridge rule target.\n",
			},
		},
		{
			name:         "helios-orchestrator.zip",
			binary:       "bootstrap",
			packagePath:  "orchestrator",
			architecture: "arm64",
			files: map[string]string{
				"README.md": "# Helios queue orchestrator\n\nDeploy `bootstrap` with the `provided.al2023` runtime on arm64. Set `STATE_MACHINE_ARN`, connect the telemetry SQS queue with partial batch responses enabled, and use a visibility timeout longer than the function timeout.\n",
			},
		},
		{
			name:         "helios-analyzer.zip",
			binary:       "bootstrap",
			packagePath:  "analyzer",
			architecture: "arm64",
			files: map[string]string{
				"README.md": "# Helios analyzer\n\nDeploy `bootstrap` with the `provided.al2023` runtime on arm64. Set `RESULT_BUCKET`, `KMS_KEY_ID`, `LIMITS_SECRET`, and `ALERT_TOPIC_ARN`. Invoke it from an Express Step Functions workflow.\n",
			},
		},
		{
			name:         "helios-query.zip",
			binary:       "bootstrap",
			packagePath:  "query",
			architecture: "arm64",
			files: map[string]string{
				"README.md": "# Helios query function\n\nDeploy `bootstrap` with the `provided.al2023` runtime on arm64. Set `RESULT_BUCKET` and expose a Function URL.\n",
			},
		},
		{
			name:         "helios-probe.zip",
			binary:       "helios-probe",
			packagePath:  "probe",
			architecture: "amd64",
			files: map[string]string{
				"README.md": "# Helios probe\n\nRun `./helios-probe -ingest-url URL -query-url URL` to submit safe and excursion readings and verify both results.\n",
			},
		},
	}

	if err := os.MkdirAll(filepath.Join(exampleRoot, "generated"), 0o755); err != nil {
		panic(err)
	}
	for _, item := range artifacts {
		binary := filepath.Join(temporary, item.binary+"-"+item.packagePath)
		packagePath := "./examples/challenges/helios-cold-chain/cmd/" + item.packagePath
		command := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", binary, packagePath)
		command.Dir = moduleRoot
		command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+item.architecture)
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			panic(err)
		}
		if err := writeArchive(filepath.Join(exampleRoot, "generated", item.name), item.binary, binary, item.files); err != nil {
			panic(err)
		}
	}
}

func findModuleRoot() (string, error) {
	directory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory, nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			return "", fmt.Errorf("go.mod not found")
		}
		directory = parent
	}
}

func writeArchive(target, binaryName, binary string, files map[string]string) error {
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	if err := addFile(archive, binaryName, binary); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		writer, err := archive.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, files[name]); err != nil {
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return os.WriteFile(target, output.Bytes(), 0o644)
}

func addFile(archive *zip.Writer, name, source string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = name
	header.Method = zip.Deflate
	header.SetMode(0o755)
	header.Modified = time.Unix(0, 0).UTC()
	writer, err := archive.CreateHeader(header)
	if err != nil {
		return err
	}
	file, err := os.Open(source)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(writer, file)
	return err
}
