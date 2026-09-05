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
	name        string
	binary      string
	packagePath string
	files       map[string]string
}

func main() {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		panic(err)
	}
	exampleRoot := filepath.Join(moduleRoot, "examples", "challenges", "worldskills-2024-day1")
	temporary, err := os.MkdirTemp("", "worldskills-day1-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temporary)

	artifacts := []artifact{
		{
			name:        "DataProcessingAPP.zip",
			binary:      "DataProcessingAPP",
			packagePath: "data-processing",
			files: map[string]string{
				"Dockerfile": "FROM public.ecr.aws/amazonlinux/amazonlinux:2\nCOPY DataProcessingAPP /usr/local/bin/DataProcessingAPP\nCOPY config.ini /etc/worldskills/config.ini\nENTRYPOINT [\"/usr/local/bin/DataProcessingAPP\", \"-config\", \"/etc/worldskills/config.ini\"]\n",
				"README.md":  "# DataProcessingAPP\n\nThe application copies new FeedbackTable records into PostgreSQL. Configure `DATABASE_URL`, and optionally `REGION`, `FEEDBACK_TABLE`, `BATCH_STATUS_PARAMETER`, `POLL_INTERVAL`, and `RUN_ONCE`. Set `RUN_ONCE=true` for an AWS Batch job. Environment variables override config.ini.\n",
				"config.ini": "[application]\nregion=us-east-1\nfeedback_table=FeedbackTable\ndatabase_url=\nbatch_status_parameter=/worldskills/day1/batch-success\npoll_interval=10s\nrun_once=true\n",
			},
		},
		{
			name:        "DataExtractionAPP.zip",
			binary:      "DataExtractionAPP",
			packagePath: "data-extraction",
			files: map[string]string{
				"Dockerfile": "FROM public.ecr.aws/amazonlinux/amazonlinux:2\nCOPY DataExtractionAPP /usr/local/bin/DataExtractionAPP\nCOPY config.ini /etc/worldskills/config.ini\nENTRYPOINT [\"/usr/local/bin/DataExtractionAPP\", \"-config\", \"/etc/worldskills/config.ini\"]\n",
				"README.md":  "# DataExtractionAPP\n\nThe application copies processed PostgreSQL records to Parameter Store. Configure `DATABASE_URL`, and optionally `REGION`, `RESULT_PREFIX`, `POLL_INTERVAL`, and `RUN_ONCE`. Keep `RUN_ONCE=false` for an ECS service. Environment variables override config.ini.\n",
				"config.ini": "[application]\nregion=us-east-1\ndatabase_url=\nresult_prefix=/worldskills/day1/results\npoll_interval=10s\nrun_once=false\n",
			},
		},
		{
			name:        "day1-probe.zip",
			binary:      "day1-probe",
			packagePath: "day1-probe",
			files: map[string]string{
				"README.md": "# day1-probe\n\nRun `./day1-probe -ingest-url URL` with competition-account AWS credentials to send a unique code and wait for its Parameter Store result.\n",
			},
		},
	}

	if err := os.MkdirAll(filepath.Join(exampleRoot, "generated"), 0o755); err != nil {
		panic(err)
	}
	for _, item := range artifacts {
		binary := filepath.Join(temporary, item.binary)
		packagePath := "./examples/challenges/worldskills-2024-day1/cmd/" + item.packagePath
		command := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", binary, packagePath)
		command.Dir = moduleRoot
		command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			panic(err)
		}
		if err := writeArchive(filepath.Join(exampleRoot, "generated", item.name), binary, item.files); err != nil {
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

func writeArchive(target, binary string, files map[string]string) error {
	var output bytes.Buffer
	archive := zip.NewWriter(&output)

	if err := addFile(archive, filepath.Base(binary), binary); err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(files)) {
		content := files[name]
		writer, err := archive.Create(name)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(writer, content); err != nil {
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
