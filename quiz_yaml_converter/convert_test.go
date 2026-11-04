package quiz_yaml_converter

import (
	"testing"
)

func TestDetectOutputFormat(t *testing.T) {
	tests := []struct {
		name         string
		outputFile   string
		templateFile string
		expected     OutputFormat
	}{
		{"CSV with .csv extension", "output.csv", "", FormatCSV},
		{"CSV with .CSV extension", "output.CSV", "", FormatCSV},
		{"Template with template file", "output.txt", "template.txt", FormatTemplate},
		{"Template without extension", "output", "", FormatTemplate},
		{"Template with .html", "output.html", "", FormatTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectOutputFormat(tt.outputFile, tt.templateFile)
			if result != tt.expected {
				t.Errorf("DetectOutputFormat(%q, %q) = %v, want %v", tt.outputFile, tt.templateFile, result, tt.expected)
			}
		})
	}
}
