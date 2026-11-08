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
// merged は <<: *anchor などのマージで取り込んだキーかどうか．（行番号は取り込み元のマッピングの行になる．）
type unknownCriteriaKey struct {
	name   string
	line   int
	merged bool
}

// YAMLの criteria オブジェクトを読み込む．
// ok / ng / repeat はそのまま読み込み，それ以外のキーは不正なキーとして記録する．
// <<: *anchor などのマージで取り込んだマッピングのキーも同じ規則で確認し，取り込み元のキーの行番号を記録する．
// criteria 以外のフィールドの未定義のキーは従来どおり無視するため，Decoder.KnownFields は使わない．
func (c *Criteria) UnmarshalYAML(node *yaml.Node) error {
	type plain Criteria // UnmarshalYAMLを持たない型に変換し，再帰呼び出しを防ぐ
	if err := node.Decode((*plain)(c)); err != nil {
		return err
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	collector := unknownKeyCollector{seenNames: map[string]bool{}}
	collector.collect(node, false)
	c.unknownKeys = collector.keys
	return nil
}

// criteria のマッピングから不正なキーを集める．
//
// YAMLのマージの規則（直接書いたキーは取り込んだキーより優先され，<<: [*a, *b] では前のものが優先される）
// に合わせて，優先されるキーから順に確認し，同じ名前のキーは最初の1つだけを記録する．
// これにより，読み込まれる値が1つだけのキーを重複して報告しない．
type unknownKeyCollector struct {
	keys      []unknownCriteriaKey
	seenNames map[string]bool
	visited   map[*yaml.Node]bool // 確認済みのマッピング．<< がある場合だけ作る．（循環参照による無限ループを防ぐ．）
}

// マッピングの直接のキーを確認してから，マージのキー（<<）が指すマッピングを順に確認する．
func (u *unknownKeyCollector) collect(mapping *yaml.Node, merged bool) {
	var merges []*yaml.Node
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := mapping.Content[i]
		if key.Tag == "!!merge" {
			merges = append(merges, mapping.Content[i+1])
			continue
		}
		u.record(key, merged)
	}
	if len(merges) == 0 {
		return
	}
	if u.visited == nil {
		u.visited = map[*yaml.Node]bool{}
	}
	u.visited[mapping] = true
	for _, value := range merges {
		u.collectMerge(value)
	}
}

// マージのキー（<<）の値をたどり，取り込むマッピングのキーを確認する．
// 値はマッピング，エイリアス，またはそれらのシーケンス（<<: [*a, *b]）のいずれか．
// （不正な値や未定義のアンカーは，ここに来る前に yaml.v3 がエラーにする．）
func (u *unknownKeyCollector) collectMerge(value *yaml.Node) {
	switch value.Kind {
	case yaml.AliasNode:
		u.collectMerge(value.Alias)
	case yaml.MappingNode:
		if u.visited[value] {
			return
		}
		u.visited[value] = true
		u.collect(value, true)
	case yaml.SequenceNode:
		for _, elem := range value.Content {
			u.collectMerge(elem)
		}
	}
}

// ok / ng / repeat 以外のキーで，まだ記録していない名前のものを記録する．
func (u *unknownKeyCollector) record(key *yaml.Node, merged bool) {
	if u.seenNames[key.Value] {
		return
	}
	u.seenNames[key.Value] = true
	switch key.Value {
	case "ok", "ng", "repeat":
		return
	}
	u.keys = append(u.keys, unknownCriteriaKey{name: key.Value, line: key.Line, merged: merged})
}

// 不正なキーごとのエラーメッセージを返す．問題番号は含まない（ValidationError.Error() で付ける）．
// マージで取り込んだキーは，行番号が別の問題などの行を指すことがあるため，その旨を書き添える．
func (c *Criteria) unknownKeyErrors() []string {
	if c == nil {
		return nil
	}
	var errs []string
	for _, key := range c.unknownKeys {
		location := fmt.Sprintf("%d行目", key.line)
		if key.merged {
			location += "，<< で取り込み"
		}
		errs = append(errs, fmt.Sprintf("不正なcriteriaキー: '%s'（%s） (使用可能: ok, ng, repeat)", key.name, location))
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
		}
		// Starts with 「 but no 」, add 」
		return item + "」"
	}
	// Ends with 」 but doesn't start with 「
	if strings.HasSuffix(item, "」") {
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
