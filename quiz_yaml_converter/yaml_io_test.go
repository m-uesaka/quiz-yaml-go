package quiz_yaml_converter

import (
	"errors"
	"io/fs"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadYAMLData(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []QuizItem
	}{
		{
			name: "all criteria sections",
			content: `- question: 問題
  answer: 答え
  criteria:
    ok: [ok1, ok2]
    ng: [ng1]
    repeat: [rep1]
`,
			expected: []QuizItem{{
				Question: "問題",
				Answer:   "答え",
				Criteria: &Criteria{OK: []string{"ok1", "ok2"}, NG: []string{"ng1"}, Repeat: []string{"rep1"}},
			}},
		},
		{
			name:     "no criteria",
			content:  "- question: 問題\n  answer: 答え\n",
			expected: []QuizItem{{Question: "問題", Answer: "答え"}},
		},
		{
			name:     "null criteria",
			content:  "- question: 問題\n  answer: 答え\n  criteria:\n",
			expected: []QuizItem{{Question: "問題", Answer: "答え"}},
		},
		{
			name:     "empty criteria",
			content:  "- question: 問題\n  answer: 答え\n  criteria: {}\n",
			expected: []QuizItem{{Question: "問題", Answer: "答え", Criteria: &Criteria{}}},
		},
		{
			name:     "criteria with only empty lists",
			content:  "- question: 問題\n  answer: 答え\n  criteria:\n    ok: []\n",
			expected: []QuizItem{{Question: "問題", Answer: "答え", Criteria: &Criteria{OK: []string{}}}},
		},
		{
			name:     "unknown item key is ignored",
			content:  "- question: 問題\n  answer: 答え\n  id: 12\n  memo: メモ\n",
			expected: []QuizItem{{Question: "問題", Answer: "答え"}},
		},
		{
			name: "merge key in criteria",
			content: `- question: 問題1
  answer: 答え1
  criteria: &common
    ng: [ng1]
- question: 問題2
  answer: 答え2
  criteria:
    <<: *common
    ok: [ok1]
`,
			expected: []QuizItem{
				{Question: "問題1", Answer: "答え1", Criteria: &Criteria{NG: []string{"ng1"}}},
				{Question: "問題2", Answer: "答え2", Criteria: &Criteria{OK: []string{"ok1"}, NG: []string{"ng1"}}},
			},
		},
		{
			name:     "empty file",
			content:  "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := writeTempYAML(t, tt.content)

			// Act
			data, err := LoadYAMLData(path)

			// Assert
			if err != nil {
				t.Fatalf("LoadYAMLData() error = %v", err)
			}
			if !reflect.DeepEqual(data, tt.expected) {
				t.Errorf("LoadYAMLData() = %+v, want %+v", data, tt.expected)
			}
		})
	}
}

// criteria に ok / ng / repeat 以外のキーがある場合，読み込みがエラーになることを確認する．
func TestLoadYAMLDataUnknownCriteriaKeys(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantErrs []string
	}{
		{
			name:     "unknown criteria key",
			content:  "- question: 問題\n  answer: 答え\n  criteria:\n    ok: [ok1]\n    maybe: [m1]\n",
			wantErrs: []string{"問題 1: 不正なcriteriaキー: 'maybe'（5行目） (使用可能: ok, ng, repeat)"},
		},
		{
			name:     "misspelled criteria key",
			content:  "- question: 問題\n  answer: 答え\n  criteria:\n    OK: [ok1]\n",
			wantErrs: []string{"問題 1: 不正なcriteriaキー: 'OK'（4行目）"},
		},
		{
			name:     "unknown keys in multiple items",
			content:  "- question: 問題1\n  answer: 答え1\n- question: 問題2\n  answer: 答え2\n  criteria:\n    maybe: [m1]\n    close: [c1]\n",
			wantErrs: []string{"問題 2: 不正なcriteriaキー: 'maybe'（6行目）", "問題 2: 不正なcriteriaキー: 'close'（7行目）"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := writeTempYAML(t, tt.content)

			// Act
			_, err := LoadYAMLData(path)

			// Assert
			if err == nil {
				t.Fatal("LoadYAMLData() error = nil, want error")
			}
			for _, want := range tt.wantErrs {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("LoadYAMLData() error = %q, want it to contain %q", err, want)
				}
			}
		})
	}
}

// 存在しないファイルを読み込んだ場合，fs.ErrNotExist として判定できるエラーを返すことを確認する．
func TestLoadYAMLDataMissingFile(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "missing.yaml")

	// Act
	_, err := LoadYAMLData(path)

	// Assert
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("LoadYAMLData() error = %v, want fs.ErrNotExist", err)
	}
}
