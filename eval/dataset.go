package eval

import (
	"bufio"
	"encoding/json"
	"os"
)

// Entry mirrors Recorder record for dataset export (minimal fields).
type Entry struct {
	Provider string         `json:"provider"`
	Model    string         `json:"model"`
	Messages []any          `json:"messages"`
	Response string         `json:"response"`
	Expected any            `json:"expected"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// BuildDataset reads a recorder NDJSON file and writes a filtered JSON array for evaluation tools.
func BuildDataset(inputPath, outputPath string, transform func(map[string]any) *Entry) error {
	in, err := os.Open(inputPath)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	defer w.Flush()
	w.WriteString("[")
	first := true
	s := bufio.NewScanner(in)
	for s.Scan() {
		var m map[string]any
		if err := json.Unmarshal([]byte(s.Text()), &m); err != nil {
			continue
		}
		e := transform(m)
		if e == nil {
			continue
		}
		b, _ := json.Marshal(e)
		if !first {
			w.WriteString(",")
		} else {
			first = false
		}
		w.Write(b)
	}
	w.WriteString("]")
	return s.Err()
}
