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
	exampleRoot := filepath.Join(moduleRoot, "examples", "challenges", "borealis-field-observatory")
	temporary, err := os.MkdirTemp("", "borealis-field-observatory-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temporary)

	artifacts := []artifact{
		{
			name:        "borealis-api.zip",
			binary:      "borealis-api",
			packagePath: "api",
			files: map[string]string{
				"Dockerfile": "FROM public.ecr.aws/docker/library/alpine:3.23\n\nRUN addgroup -S app && adduser -S -G app app\nCOPY --chown=app:app borealis-api /usr/local/bin/borealis-api\nUSER app\nEXPOSE 8080\nENTRYPOINT [\"/usr/local/bin/borealis-api\"]\n",
				"README.md":  "# Borealis field API\n\nBuild with `docker build -t borealis-api .`. The image listens on port 8080 and provides `GET /health`, `POST /observations`, and `GET /observations/{observation_id}`.\n\nRequired environment variables: `IOT_ENDPOINT`, `IOT_TOPIC`, `APPCONFIG_APPLICATION`, `APPCONFIG_ENVIRONMENT`, `APPCONFIG_PROFILE`, `ATHENA_DATABASE`, `ATHENA_TABLE`, and `ATHENA_WORKGROUP`.\n",
			},
		},
		{
			name:        "borealis-probe.zip",
			binary:      "borealis-probe",
			packagePath: "probe",
			files: map[string]string{
				"README.md": "# Borealis probe\n\nRun `./borealis-probe -endpoint URL` to submit an attention observation and wait for it through Athena.\n",
			},
		},
	}
	if err := os.MkdirAll(filepath.Join(exampleRoot, "generated"), 0o755); err != nil {
		panic(err)
	}
	for _, item := range artifacts {
		binary := filepath.Join(temporary, item.binary)
		command := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", binary, "./examples/challenges/borealis-field-observatory/cmd/"+item.packagePath)
		command.Dir = moduleRoot
		command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
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
