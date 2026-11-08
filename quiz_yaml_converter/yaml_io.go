package quiz_yaml_converter

import (
	"fmt"
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

	// メッセージの形式を -validate の出力とそろえるため，validateQuizItem と同じ criteriaKeyErrors を使う．
	var errs []string
	for i, item := range data {
		for _, e := range criteriaKeyErrors(item, i+1) {
			errs = append(errs, e.Error())
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("invalid criteria keys:\n%s", strings.Join(errs, "\n"))
	}

	return data, nil
}

// YAMLファイルを読み込んでQuizItemのスライスに変換する．
// criteria の不正なキーはエラーにせず，Criteria に記録したまま返す．
// ファイルが存在しない場合のエラーは errors.Is(err, fs.ErrNotExist) で判定できる．
func decodeYAMLFile(yamlFilePath string) ([]QuizItem, error) {
	yamlData, err := os.ReadFile(yamlFilePath)
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
