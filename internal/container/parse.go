package container

import (
	"encoding/json"
	"strings"

	fctypes "fuclaw/internal/types"
)

func ParseOutputLine(line string) (*fctypes.ContainerOutput, bool, error) {
	if !strings.Contains(line, OutputStartMarker) {
		return nil, false, nil
	}
	startIdx := strings.Index(line, OutputStartMarker)
	if startIdx < 0 {
		return nil, false, nil
	}
	startIdx += len(OutputStartMarker)

	endIdx := strings.Index(line, OutputEndMarker)
	if endIdx < 0 || endIdx <= startIdx {
		return nil, false, nil
	}

	jsonStr := line[startIdx:endIdx]
	var output fctypes.ContainerOutput
	if err := json.Unmarshal([]byte(jsonStr), &output); err != nil {
		return nil, true, err
	}
	return &output, true, nil
}
