package quiz_yaml_converter

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// YAMLファイルからデータを読み込む．
// criteria に ok / ng / repeat 以外のキーがある場合はエラーになる．
// criteria 以外のフィールドの未定義のキーは無視する．
func LoadYAMLData(yamlFilePath string) ([]QuizItem, error) {
	data, err := decodeYAMLFile(yamlFilePath)
	if err != nil {
		return nil, err
	}

	var errs []string
	for i, item := range data {
		errs = append(errs, item.Criteria.unknownKeyErrors(fmt.Sprintf("問題 %d: ", i+1))...)
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid criteria keys:\n%s", strings.Join(errs, "\n"))
	}

	return data, nil
}

// YAMLファイルを読み込んでQuizItemのスライスに変換する．
// criteria の不正なキーはエラーにせず，Criteria に記録したまま返す．
func decodeYAMLFile(yamlFilePath string) ([]QuizItem, error) {
	yamlFile, err := os.Open(yamlFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open YAML file: %w", err)
	}
	defer yamlFile.Close()

	yamlData, err := io.ReadAll(yamlFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %w", err)
	}

	var data []QuizItem
	if err := yaml.Unmarshal(yamlData, &data); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return data, nil
}

// QuizItemのスライスをYAMLファイルとして書き出す．
func SaveYAMLData(items []QuizItem, yamlFilePath string) error {
	out, err := yaml.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}
	if err := os.WriteFile(yamlFilePath, out, 0o644); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}
	return nil
}
