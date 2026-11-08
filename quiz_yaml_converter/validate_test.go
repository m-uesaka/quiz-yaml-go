package quiz_yaml_converter

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestValidateYAMLFileValid(t *testing.T) {
	// Arrange
	path := writeTempYAML(t, `- question: 問題1
  answer: 答え1
  criteria:
    ok: [ok1]
    ng: [ng1]
    repeat: [rep1]
- question: 問題2
  answer: 答え2
`)

	// Act
	result := ValidateYAMLFile(path)

	// Assert
	if !result.IsValid || len(result.Errors) != 0 {
		t.Errorf("ValidateYAMLFile() = %+v, want valid", result)
	}
	if result.Items != 2 {
		t.Errorf("Items = %d, want 2", result.Items)
	}
}

func TestValidateYAMLFileInvalidCriteria(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantErrors []string
	}{
		{
			name:       "empty criteria entries",
			content:    "- question: 問題\n  answer: 答え\n  criteria:\n    ok: [ok1, \"\"]\n    ng: [\" \"]\n    repeat: [\"\"]\n",
			wantErrors: []string{"問題 1: criteria.ok[1] が空です", "問題 1: criteria.ng[0] が空です", "問題 1: criteria.repeat[0] が空です"},
		},
		{
			name:       "unknown criteria key",
			content:    "- question: 問題\n  answer: 答え\n  criteria:\n    maybe: [m1]\n",
			wantErrors: []string{"問題 1: 不正なcriteriaキー: 'maybe'（4行目） (使用可能: ok, ng, repeat)"},
		},
		{
			name:       "unknown criteria key is reported with other errors",
			content:    "- question: 問題1\n  answer: 答え1\n  criteria:\n    maybe: [m1]\n- question: 問題2\n  answer: 答え2\n- question: 問題3\n  answer: \"\"\n",
			wantErrors: []string{"問題 1: 不正なcriteriaキー: 'maybe'（4行目）", "問題 3: 答え (answer) が空です"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := writeTempYAML(t, tt.content)

			// Act
			result := ValidateYAMLFile(path)

			// Assert
			if result.IsValid {
				t.Fatalf("ValidateYAMLFile() IsValid = true, want false")
			}
			if result.Items == 0 {
				t.Errorf("Items = 0, want the number of items in the file")
			}
			if len(result.Errors) != len(tt.wantErrors) {
				t.Fatalf("Errors = %q, want %d errors", result.Errors, len(tt.wantErrors))
			}
			for i, want := range tt.wantErrors {
				if !strings.Contains(result.Errors[i], want) {
					t.Errorf("Errors[%d] = %q, want it to contain %q", i, result.Errors[i], want)
				}
			}
		})
	}
}

// ファイルが存在しない場合，os.Stat を使わずに従来と同じメッセージを返すことを確認する．
func TestValidateYAMLFileMissingFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "missing.yaml")

	// Act
	result := ValidateYAMLFile(path)

	// Assert
	want := []string{"ファイルが存在しません: " + path}
	if result.IsValid {
		t.Errorf("IsValid = true, want false")
	}
	if !reflect.DeepEqual(result.Errors, want) {
		t.Errorf("Errors = %q, want %q", result.Errors, want)
	}
	wantDetails := []ValidationError{{Message: "ファイルが存在しません: " + path}}
	if !reflect.DeepEqual(result.Details, wantDetails) {
		t.Errorf("Details = %+v, want %+v", result.Details, wantDetails)
	}
}

func TestValidateYAMLFileFileLevelErrors(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantPrefix string
	}{
		{name: "empty file", content: "", wantPrefix: "YAMLファイルにクイズデータが含まれていません"},
		{name: "broken YAML", content: "- question: [\n", wantPrefix: "YAMLファイルの読み込みエラー: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := writeTempYAML(t, tt.content)

			// Act
			result := ValidateYAMLFile(path)

			// Assert
			if result.IsValid {
				t.Fatalf("IsValid = true, want false")
			}
			if len(result.Errors) != 1 || !strings.HasPrefix(result.Errors[0], tt.wantPrefix) {
				t.Fatalf("Errors = %q, want one error with prefix %q", result.Errors, tt.wantPrefix)
			}
			if len(result.Details) != 1 || result.Details[0].Item != 0 {
				t.Errorf("Details = %+v, want one file-level error (Item 0)", result.Details)
			}
		})
	}
}

// Errors と Details が同じ順序・同じ内容になることを確認する．
func TestValidateYAMLFileDetailsMatchErrors(t *testing.T) {
	// Arrange
	path := writeTempYAML(t, "- question: \"\"\n  answer: \"\"\n  tags: [\"\"]\n")

	// Act
	result := ValidateYAMLFile(path)

	// Assert
	want := []ValidationError{
		{Item: 1, Field: "question", Message: "問題文 (question) が空です"},
		{Item: 1, Field: "answer", Message: "答え (answer) が空です"},
		{Item: 1, Field: "tags[0]", Message: "tags[0] が空です"},
	}
	if !reflect.DeepEqual(result.Details, want) {
		t.Errorf("Details = %+v, want %+v", result.Details, want)
	}
	wantErrors := []string{"問題 1: 問題文 (question) が空です", "問題 1: 答え (answer) が空です", "問題 1: tags[0] が空です"}
	if !reflect.DeepEqual(result.Errors, wantErrors) {
		t.Errorf("Errors = %q, want %q", result.Errors, wantErrors)
	}
}

func TestValidationErrorFormat(t *testing.T) {
	tests := []struct {
		name string
		err  ValidationError
		want string
	}{
		{name: "item-level error", err: ValidationError{Item: 1, Field: "answer", Message: "答え (answer) が空です"}, want: "問題 1: 答え (answer) が空です"},
		{name: "item number over 9", err: ValidationError{Item: 12, Field: "criteria.ok[0]", Message: "criteria.ok[0] が空です"}, want: "問題 12: criteria.ok[0] が空です"},
		{name: "file-level error", err: ValidationError{Message: "YAMLファイルにクイズデータが含まれていません"}, want: "YAMLファイルにクイズデータが含まれていません"},
		{name: "item-level error with source", err: ValidationError{Item: 3, Field: "answer", Message: "答え (answer) が空です", Source: "md/sub/c.md"}, want: "問題 3 (md/sub/c.md): 答え (answer) が空です"},
		{name: "file-level error with source", err: ValidationError{Message: "読み込めません", Source: "md/c.md"}, want: "md/c.md: 読み込めません"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			gotError := tt.err.Error()
			gotString := tt.err.String()

			// Assert
			if gotError != tt.want {
				t.Errorf("Error() = %q, want %q", gotError, tt.want)
			}
			if gotString != tt.want {
				t.Errorf("String() = %q, want %q", gotString, tt.want)
			}
		})
	}
}

func TestValidateItemsValid(t *testing.T) {
	tests := []struct {
		name  string
		items []QuizItem
	}{
		{name: "required fields only", items: []QuizItem{{Question: "問題", Answer: "答え"}}},
		{
			name: "all fields",
			items: []QuizItem{{
				Question: "問題",
				Answer:   "答え",
				Spell:    "Answer",
				Tags:     []string{"タグ"},
				Comments: []string{"コメント"},
				Criteria: &Criteria{OK: []string{"ok1"}, NG: []string{"ng1"}, Repeat: []string{"rep1"}},
			}},
		},
		{name: "empty criteria", items: []QuizItem{{Question: "問題", Answer: "答え", Criteria: &Criteria{}}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := ValidateItems(tt.items)

			// Assert
			if got != nil {
				t.Errorf("ValidateItems() = %+v, want nil", got)
			}
		})
	}
}

func TestValidateItemsInvalid(t *testing.T) {
	tests := []struct {
		name  string
		items []QuizItem
		want  []ValidationError
	}{
		{
			name:  "nil",
			items: nil,
			want:  []ValidationError{{Message: "YAMLファイルにクイズデータが含まれていません"}},
		},
		{
			name:  "empty slice",
			items: []QuizItem{},
			want:  []ValidationError{{Message: "YAMLファイルにクイズデータが含まれていません"}},
		},
		{
			name:  "empty question",
			items: []QuizItem{{Question: "", Answer: "答え"}},
			want:  []ValidationError{{Item: 1, Field: "question", Message: "問題文 (question) が空です"}},
		},
		{
			name:  "whitespace-only answer",
			items: []QuizItem{{Question: "問題", Answer: " \n"}},
			want:  []ValidationError{{Item: 1, Field: "answer", Message: "答え (answer) が空です"}},
		},
		{
			name: "empty entries in criteria, comments and tags",
			items: []QuizItem{{
				Question: "問題",
				Answer:   "答え",
				Tags:     []string{"タグ", ""},
				Comments: []string{" "},
				Criteria: &Criteria{OK: []string{"ok1", ""}, NG: []string{" "}, Repeat: []string{""}},
			}},
			want: []ValidationError{
				{Item: 1, Field: "criteria.ok[1]", Message: "criteria.ok[1] が空です"},
				{Item: 1, Field: "criteria.ng[0]", Message: "criteria.ng[0] が空です"},
				{Item: 1, Field: "criteria.repeat[0]", Message: "criteria.repeat[0] が空です"},
				{Item: 1, Field: "comments[0]", Message: "comments[0] が空です"},
				{Item: 1, Field: "tags[1]", Message: "tags[1] が空です"},
			},
		},
		{
			name: "item numbers are 1-based",
			items: []QuizItem{
				{Question: "問題1", Answer: "答え1"},
				{Question: "問題2", Answer: ""},
			},
			want: []ValidationError{{Item: 2, Field: "answer", Message: "答え (answer) が空です"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := ValidateItems(tt.items)

			// Assert
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ValidateItems() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// criteria の不正なキーは，YAMLから読み込んだデータで確認する（不正なキーの記録方法は UnmarshalYAML に任せる）．
func TestValidateItemsUnknownCriteriaKey(t *testing.T) {
	// Arrange
	items, err := decodeYAMLFile(writeTempYAML(t, "- question: 問題\n  answer: 答え\n  criteria:\n    maybe: [m1]\n"))
	if err != nil {
		t.Fatalf("decodeYAMLFile() error = %v", err)
	}

	// Act
	got := ValidateItems(items)

	// Assert
	want := []ValidationError{{Item: 1, Field: "criteria", Message: "不正なcriteriaキー: 'maybe'（4行目） (使用可能: ok, ng, repeat)"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ValidateItems() = %+v, want %+v", got, want)
	}
}

func TestCheckNonEmptyValid(t *testing.T) {
	tests := []struct {
		name   string
		values []string
	}{
		{name: "nil", values: nil},
		{name: "empty slice", values: []string{}},
		{name: "non-empty values", values: []string{"a", " b ", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := checkNonEmpty(1, "tags", tt.values)

			// Assert
			if got != nil {
				t.Errorf("checkNonEmpty() = %+v, want nil", got)
			}
		})
	}
}

func TestCheckNonEmptyInvalid(t *testing.T) {
	tests := []struct {
		name   string
		index  int
		field  string
		values []string
		want   []ValidationError
	}{
		{
			name:   "empty string",
			index:  1,
			field:  "tags",
			values: []string{""},
			want:   []ValidationError{{Item: 1, Field: "tags[0]", Message: "tags[0] が空です"}},
		},
		{
			name:   "whitespace-only values among others",
			index:  3,
			field:  "criteria.ok",
			values: []string{"ok1", " ", "ok2", "\t"},
			want: []ValidationError{
				{Item: 3, Field: "criteria.ok[1]", Message: "criteria.ok[1] が空です"},
				{Item: 3, Field: "criteria.ok[3]", Message: "criteria.ok[3] が空です"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := checkNonEmpty(tt.index, tt.field, tt.values)

			// Assert
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("checkNonEmpty() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
