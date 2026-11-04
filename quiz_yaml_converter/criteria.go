package quiz_yaml_converter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// 1問分の判定基準を表す構造体．
// YAMLの criteria オブジェクトに対応する．ok / ng / repeat 以外のキーは読み込み時に記録され，
// LoadYAMLData ではエラー，ValidateYAMLFile ではバリデーションエラーとして報告される．
type Criteria struct {
	OK     []string `yaml:"ok,omitempty"`     // 別解（正解として扱う答え）
	NG     []string `yaml:"ng,omitempty"`     // 誤答として扱う答え
	Repeat []string `yaml:"repeat,omitempty"` // もう一度回答を求める答え

	unknownKeys []unknownCriteriaKey // YAMLに書かれていた ok / ng / repeat 以外のキー
}

// criteria に書かれていた不正なキーと，その行番号．
type unknownCriteriaKey struct {
	name string
	line int
}

// YAMLの criteria オブジェクトを読み込む．
// ok / ng / repeat はそのまま読み込み，それ以外のキーは不正なキーとして記録する．
// <<: *anchor などのマージで取り込んだマッピングのキーも同じ規則で確認し，取り込んだ側のキーの行番号を記録する．
// criteria 以外のフィールドの未定義のキーは従来どおり無視するため，Decoder.KnownFields は使わない．
func (c *Criteria) UnmarshalYAML(node *yaml.Node) error {
	type plain Criteria // UnmarshalYAMLを持たない型に変換し，再帰呼び出しを防ぐ
	if err := node.Decode((*plain)(c)); err != nil {
		return err
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	c.collectUnknownKeys(node, map[*yaml.Node]bool{})
	return nil
}

// マッピングのキーのうち ok / ng / repeat 以外のものを不正なキーとして記録する．
// マージのキー（<<）は，その値が指すマッピングを再帰的に確認する．
// visited は確認済みのマッピングで，循環参照による無限ループと，同じマッピングを複数回取り込んだ場合の重複を防ぐ．
func (c *Criteria) collectUnknownKeys(mapping *yaml.Node, visited map[*yaml.Node]bool) {
	if visited[mapping] {
		return
	}
	visited[mapping] = true
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		if key.Tag == "!!merge" {
			c.collectUnknownKeysInMerge(mapping.Content[i+1], visited)
			continue
		}
		switch key.Value {
		case "ok", "ng", "repeat":
			continue
		}
		c.unknownKeys = append(c.unknownKeys, unknownCriteriaKey{name: key.Value, line: key.Line})
	}
}

// マージのキー（<<）の値をたどり，取り込むマッピングのキーを確認する．
// 値はマッピング，エイリアス，またはそれらのシーケンス（<<: [*a, *b]）のいずれか．
func (c *Criteria) collectUnknownKeysInMerge(value *yaml.Node, visited map[*yaml.Node]bool) {
	switch value.Kind {
	case yaml.AliasNode:
		if value.Alias != nil {
			c.collectUnknownKeysInMerge(value.Alias, visited)
		}
	case yaml.MappingNode:
		c.collectUnknownKeys(value, visited)
	case yaml.SequenceNode:
		for _, elem := range value.Content {
			c.collectUnknownKeysInMerge(elem, visited)
		}
	}
}

// 不正なキーごとのエラーメッセージを返す．prefixは「問題 1: 」のような問題の位置を表す文字列．
func (c *Criteria) unknownKeyErrors(prefix string) []string {
	if c == nil {
		return nil
	}
	var errs []string
	for _, key := range c.unknownKeys {
		errs = append(errs, fmt.Sprintf("%s不正なcriteriaキー: '%s'（%d行目） (使用可能: ok, ng, repeat)", prefix, key.name, key.line))
	}
	return errs
}

// 判定基準が1件も含まれていない（nilを含む）かどうかを返す．
func (c *Criteria) IsEmpty() bool {
	return c == nil || (len(c.OK) == 0 && len(c.NG) == 0 && len(c.Repeat) == 0)
}

// 必要に応じて「」を追加する．
//
// - 既に"「"で始まり"」"で終わっている場合はそのまま返す
// - "「"で始まっているが"」"で終わっていない場合
//   - "」"がどこかに含まれている場合はそのまま返す
//   - 含まれていない場合は"」"を追加して返す
//
// - "」"を含むが"「"で始まっていない場合は"「"を追加して返す
// - どちらも含まれていない場合は"「"と"」"を最初と最後に追加して返す．
func AddQuotesIfNeeded(item string) string {
	// Already starts with 「 and ends with 」
	if strings.HasPrefix(item, "「") && strings.HasSuffix(item, "」") {
		return item
	}
	// Starts with 「 but doesn't end with 」
	if strings.HasPrefix(item, "「") {
		// Contains 」 somewhere (like 「美術館」（おまけ））
		if strings.Contains(item, "」") {
			return item
		} else {
			// Starts with 「 but no 」, add 」
			return item + "」"
		}
	}
	// Ends with 」 but doesn't start with 「
	if !strings.HasPrefix(item, "「") && strings.HasSuffix(item, "」") {
		return "「" + item
	}
	// No quotes at all, add both
	return "「" + item + "」"
}

// 正誤判定の文字列をフォーマットするための補助関数
// 別解などの単語に適切に「」を追加して羅列し，最後に指定された文字列を追加する．
func formatCriteriaSection(items []string, suffix string) string {
	if len(items) == 0 {
		return ""
	}

	var formattedItems []string
	for _, item := range items {
		formattedItems = append(formattedItems, AddQuotesIfNeeded(item))
	}
	return strings.Join(formattedItems, "") + suffix
}

// 正誤判定のフォーマットを行う．
//
// 形式は「別解1」「別解2」／「誤答1」は誤答／「もう一度1」はもう一度という形式で返す．
// 空のリストは出力に含めない．Criteriaがnilの場合は空文字を返す．
// テンプレートからは {{formatCriteria .Criteria}} のように使用する．
func (c *Criteria) Format() string {
	if c == nil {
		return ""
	}

	var parts []string
	if len(c.OK) > 0 {
		parts = append(parts, formatCriteriaSection(c.OK, ""))
	}
	if len(c.NG) > 0 {
		parts = append(parts, formatCriteriaSection(c.NG, "は誤答"))
	}
	if len(c.Repeat) > 0 {
		parts = append(parts, formatCriteriaSection(c.Repeat, "はもう一度"))
	}

	return strings.Join(parts, "／")
}
