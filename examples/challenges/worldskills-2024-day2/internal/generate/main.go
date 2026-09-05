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
	name   string
	binary string
	files  map[string]string
}

func main() {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		panic(err)
	}
	exampleRoot := filepath.Join(moduleRoot, "examples", "challenges", "worldskills-2024-day2")
	temporary, err := os.MkdirTemp("", "worldskills-day2-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(temporary)

	artifacts := []artifact{
		{
			name:   "stub1.zip",
			binary: "stub1",
			files: map[string]string{
				"Dockerfile": "FROM public.ecr.aws/amazonlinux/amazonlinux:2\nCOPY stub1 /usr/local/bin/stub1\nEXPOSE 8080\nENTRYPOINT [\"/usr/local/bin/stub1\"]\n",
				"README.md":  "# stub1\n\nReference blood-pressure query service. It listens on `PORT` (default: `8080`) and reads `TABLE_NAME` (default: `cloudraiser-iot`).\n",
			},
		},
		{
			name:   "day2-probe.zip",
			binary: "day2-probe",
			files: map[string]string{
				"README.md": "# day2-probe\n\nRun `./day2-probe -blood-pressure-ingest URL -blood-pressure-query URL -enterprise-api URL` to exercise both pipelines.\n",
			},
		},
	}

	for _, item := range artifacts {
		binary := filepath.Join(temporary, item.binary)
		packagePath := "./examples/challenges/worldskills-2024-day2/cmd/" + item.binary
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
