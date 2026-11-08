package quiz_yaml_converter

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

// 警告の有無にかかわらず変換が行われ，警告が返されることを確認する．
func TestConvertWithWarnings(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		outputName   string
		withTemplate bool
		wantWarnings []ValidationError
		wantContent  string
	}{
		{
			name:        "valid data to CSV",
			content:     "- question: 問題\n  answer: 答え\n",
			outputName:  "quiz.csv",
			wantContent: "答え",
		},
		{
			name:         "empty answer to CSV",
			content:      "- question: 問題\n  answer: \"\"\n",
			outputName:   "quiz.csv",
			wantWarnings: []ValidationError{{Item: 1, Field: "answer", Message: "答え (answer) が空です"}},
			wantContent:  "問題",
		},
		{
			name:         "empty list to CSV",
			content:      "[]\n",
			outputName:   "quiz.csv",
			wantWarnings: []ValidationError{{Message: "YAMLファイルにクイズデータが含まれていません"}},
			wantContent:  "question,answer,spell,criteria",
		},
		{
			name:         "empty question with template",
			content:      "- question: \"\"\n  answer: 答え\n  tags: [\"\"]\n",
			outputName:   "quiz.txt",
			withTemplate: true,
			wantWarnings: []ValidationError{
				{Item: 1, Field: "question", Message: "問題文 (question) が空です"},
				{Item: 1, Field: "tags[0]", Message: "tags[0] が空です"},
			},
			wantContent: "[答え]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			input := writeFile(t, dir, "quiz.yaml", tt.content)
			output := filepath.Join(dir, tt.outputName)
			tmpl := ""
			if tt.withTemplate {
				tmpl = writeFile(t, dir, "custom.tmpl", "{{range .Items}}[{{.Answer}}]{{end}}")
			}

			// Act
			warnings, err := ConvertWithWarnings(input, output, tmpl)

			// Assert
			if err != nil {
				t.Fatalf("ConvertWithWarnings() error = %v", err)
			}
			if !reflect.DeepEqual(warnings, tt.wantWarnings) {
				t.Errorf("warnings = %+v, want %+v", warnings, tt.wantWarnings)
			}
			got := readFile(t, output)
			if !strings.Contains(got, tt.wantContent) {
				t.Errorf("output = %q, want it to contain %q", got, tt.wantContent)
			}
		})
	}
}

// criteria の不正なキーは警告ではなくエラーになり，出力ファイルを作らないことを確認する．
func TestConvertWithWarningsErrors(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		outputName string
		wantErr    string
	}{
		{
			name:       "unknown criteria key",
			content:    "- question: 問題\n  answer: \"\"\n  criteria:\n    maybe: [m1]\n",
			outputName: "quiz.csv",
			wantErr:    "不正なcriteriaキー: 'maybe'",
		},
		{
			name:       "template is required for non-CSV output",
			content:    "- question: 問題\n  answer: 答え\n",
			outputName: "quiz.txt",
			wantErr:    "template file is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			input := writeFile(t, dir, "quiz.yaml", tt.content)
			output := filepath.Join(dir, tt.outputName)

			// Act
			warnings, err := ConvertWithWarnings(input, output, "")

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ConvertWithWarnings() error = %v, want it to contain %q", err, tt.wantErr)
			}
			if warnings != nil {
				t.Errorf("warnings = %+v, want nil", warnings)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Errorf("output file should not be created: %v", statErr)
			}
		})
	}
}

// Markdownの集約時も警告を返し，警告があってもYAMLファイルを書き出すことを確認する．
func TestConvertMarkdownDirToYAMLWithWarnings(t *testing.T) {
	const validMD = "---\ntitle: A\n---\n## Question\n\n問題文A\n\n## Answer\n\n答えA\n"
	const emptyAnswerMD = "---\ntitle: B\n---\n## Question\n\n問題文B\n\n## Answer\n\n"

	tests := []struct {
		name         string
		files        map[string]string
		wantWarnings []ValidationError // Source は mdDir からの相対パスで書く
		wantContent  string
	}{
		{
			name:        "valid files",
			files:       map[string]string{"a.md": validMD},
			wantContent: "答えA",
		},
		{
			name:         "empty answer in the second file",
			files:        map[string]string{"a.md": validMD, "b.md": emptyAnswerMD},
			wantWarnings: []ValidationError{{Item: 2, Field: "answer", Message: "答え (answer) が空です", Source: "b.md"}},
			wantContent:  "問題文B",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mdDir := t.TempDir()
			for name, content := range tt.files {
				writeFile(t, mdDir, name, content)
			}
			output := filepath.Join(t.TempDir(), "quiz.yaml")
			want := withSourceDir(tt.wantWarnings, mdDir)

			// Act
			warnings, err := ConvertMarkdownDirToYAMLWithWarnings(mdDir, output, false)

			// Assert
			if err != nil {
				t.Fatalf("ConvertMarkdownDirToYAMLWithWarnings() error = %v", err)
			}
			if !reflect.DeepEqual(warnings, want) {
				t.Errorf("warnings = %+v, want %+v", warnings, want)
			}
			got := readFile(t, output)
			if !strings.Contains(got, tt.wantContent) {
				t.Errorf("output = %q, want it to contain %q", got, tt.wantContent)
			}
		})
	}
}

func TestConvertMarkdownDirToYAMLWithWarningsError(t *testing.T) {
	const brokenMD = "frontmatterなし\n"
	const emptyAnswerMD = "---\ntitle: B\n---\n## Question\n\n問題文B\n\n## Answer\n\n"

	tests := []struct {
		name         string
		files        map[string]string
		wantWarnings []ValidationError // Source は mdDir からの相対パスで書く
	}{
		{
			name:  "all files are broken",
			files: map[string]string{"a.md": brokenMD},
		},
		{
			name:         "warnings of parsed files are returned with the parse error",
			files:        map[string]string{"a.md": brokenMD, "sub/b.md": emptyAnswerMD},
			wantWarnings: []ValidationError{{Item: 1, Field: "answer", Message: "答え (answer) が空です", Source: filepath.Join("sub", "b.md")}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mdDir := t.TempDir()
			for name, content := range tt.files {
				if err := os.MkdirAll(filepath.Join(mdDir, filepath.Dir(name)), 0o755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				writeFile(t, mdDir, name, content)
			}
			output := filepath.Join(t.TempDir(), "quiz.yaml")
			want := withSourceDir(tt.wantWarnings, mdDir)

			// Act
			warnings, err := ConvertMarkdownDirToYAMLWithWarnings(mdDir, output, true)

			// Assert
			if err == nil {
				t.Fatal("ConvertMarkdownDirToYAMLWithWarnings() error = nil, want error")
			}
			if !reflect.DeepEqual(warnings, want) {
				t.Errorf("warnings = %+v, want %+v", warnings, want)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Errorf("output file should not be written: stat error = %v", statErr)
			}
		})
	}
}

// withSourceDir は warnings の各 Source の前に dir を付けたコピーを返す．
func withSourceDir(warnings []ValidationError, dir string) []ValidationError {
	if warnings == nil {
		return nil
	}
	out := make([]ValidationError, len(warnings))
	for i, w := range warnings {
		w.Source = filepath.Join(dir, w.Source)
		out[i] = w
	}
	return out
}

// 未対応のフォーマットを渡した場合に，別のフォーマットで書き出さずにエラーになることを確認する．
func TestWriteOutputUnsupportedFormat(t *testing.T) {
	// Arrange
	output := filepath.Join(t.TempDir(), "quiz.json")
	data := []QuizItem{{Question: "問題", Answer: "答え"}}

	// Act
	err := writeOutput(data, OutputFormat("json"), output, "")

	// Assert
	if err == nil || !strings.Contains(err.Error(), `unsupported output format: "json"`) {
		t.Fatalf("writeOutput() error = %v, want unsupported output format error", err)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Errorf("output file should not be written: stat error = %v", statErr)
	}
}
