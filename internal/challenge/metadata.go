package challenge

import (
	"bytes"
	"fmt"
	"image/jpeg"

	"codeberg.org/megakuul/cloudjam/pkg/challenge/api"
)

func ValidateDiagrams(diagrams map[string][]byte) error {
	if len(diagrams) > api.MaxDiagramCount {
		return fmt.Errorf("more than %d diagrams are not supported", api.MaxDiagramCount)
	}
	for name, diagram := range diagrams {
		if len(diagram) > api.MaxDiagramSize {
			return fmt.Errorf("diagram %q is larger than %d bytes", name, api.MaxDiagramSize)
		}
		if _, err := jpeg.DecodeConfig(bytes.NewReader(diagram)); err != nil {
			return fmt.Errorf("diagram %q is not a valid JPEG: %w", name, err)
		}
	}
	return nil
}
