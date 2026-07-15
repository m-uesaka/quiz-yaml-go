// 1問1ファイルのMarkdownクイズをQuizItemに変換するためのパーサーです．
// frontmatter分離＋行ベースの単純なステートマシンで実装しており，
// 外部のMarkdownパーサーライブラリには依存しません．
package quiz_yaml_converter

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatter部分のみを表す構造体．
// Title / Date はパース対象だがQuizItemには対応フィールドが無いため，
// パース後は意図的に破棄する（title/dateを保持する要件は無い）．
type quizFrontmatter struct {
	Title string   `yaml:"title"`
	Date  string   `yaml:"date"`
	Tags  []string `yaml:"tags"`
}

// markdownSections はMarkdown本文から抽出した各セクションの内容を保持する．
type markdownSections struct {
	question string
	answer   string
	spell    string
	comments []string
	ok       []string
	ng       []string
	close    []string
}

// splitFrontmatter は "---" で囲まれたfrontmatterと本文を分離する．
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	const delim = "---"
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != delim {
		return "", "", fmt.Errorf("frontmatterが見つかりません（先頭が'---'ではありません）")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == delim {
			return strings.Join(lines[1:i], "\n"), strings.Join(lines[i+1:], "\n"), nil
		}
	}
	return "", "", fmt.Errorf("frontmatterの終端'---'が見つかりません")
}

// trimBlankLines は先頭と末尾の空行のみを取り除く．
func trimBlankLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}

// joinLines はセクション本文の行を改行を保持したまま1つの文字列にする．
func joinLines(lines []string) string {
	return strings.Join(trimBlankLines(lines), "\n")
}

// parseBulletList は箇条書き（"- "）の行を1行1要素の[]stringに変換する．
func parseBulletList(lines []string) []string {
	var items []string
	for _, line := range trimBlankLines(lines) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		items = append(items, strings.TrimPrefix(trimmed, "- "))
	}
	return items
}

// parseParagraphs は空行区切りの段落に分割し，各段落内の改行は保持したまま
// 1段落1要素の[]stringに変換する．Commentセクションのように箇条書きではない
// が複数件になりうる自由記述をQuizItemの[]stringフィールドに落とし込むための
// 変換．
func parseParagraphs(lines []string) []string {
	var paragraphs []string
	var current []string
	flush := func() {
		if len(current) > 0 {
			paragraphs = append(paragraphs, strings.Join(current, "\n"))
			current = nil
		}
	}
	for _, line := range trimBlankLines(lines) {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return paragraphs
}

// parseMarkdownSections はfrontmatterを除いたMarkdown本文を
// Question / Answer / Spell / Criteria(OK/NG/Close) / Comment の
// 各セクションに分割する．
func parseMarkdownSections(body string) (markdownSections, error) {
	const (
		sectionNone = iota
		sectionQuestion
		sectionAnswer
		sectionSpell
		sectionCriteria
		sectionCriteriaOK
		sectionCriteriaNG
		sectionCriteriaClose
		sectionComment
	)

	buffers := map[int][]string{}
	current := sectionNone

	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "### "):
			switch strings.TrimSpace(strings.TrimPrefix(line, "### ")) {
			case "OK":
				current = sectionCriteriaOK
			case "NG":
				current = sectionCriteriaNG
			case "Close":
				current = sectionCriteriaClose
			default:
				return markdownSections{}, fmt.Errorf("未知の見出しです: %q", line)
			}
			continue
		case strings.HasPrefix(line, "## "):
			switch strings.TrimSpace(strings.TrimPrefix(line, "## ")) {
			case "Question":
				current = sectionQuestion
			case "Answer":
				current = sectionAnswer
			case "Spell":
				current = sectionSpell
			case "Criteria":
				current = sectionCriteria
			case "Comment":
				current = sectionComment
			default:
				return markdownSections{}, fmt.Errorf("未知の見出しです: %q", line)
			}
			continue
		}
		if current != sectionNone {
			buffers[current] = append(buffers[current], line)
		}
	}

	return markdownSections{
		question: joinLines(buffers[sectionQuestion]),
		answer:   joinLines(buffers[sectionAnswer]),
		spell:    joinLines(buffers[sectionSpell]),
		comments: parseParagraphs(buffers[sectionComment]),
		ok:       parseBulletList(buffers[sectionCriteriaOK]),
		ng:       parseBulletList(buffers[sectionCriteriaNG]),
		close:    parseBulletList(buffers[sectionCriteriaClose]),
	}, nil
}

// buildCriteria は OK/NG/Close を criteria マップに変換する．
// "Close" は既存YAMLの "repeat" に対応する名前変換を行う．
// OK/NG/Closeが全て空の場合は nil を返す．
func buildCriteria(ok, ng, close []string) map[string][]string {
	criteria := map[string][]string{}
	if len(ok) > 0 {
		criteria["ok"] = ok
	}
	if len(ng) > 0 {
		criteria["ng"] = ng
	}
	if len(close) > 0 {
		criteria["repeat"] = close
	}
	if len(criteria) == 0 {
		return nil
	}
	return criteria
}

// ParseMarkdownFile は1問分のMarkdownファイルをQuizItemに変換する．
func ParseMarkdownFile(mdFilePath string) (QuizItem, error) {
	raw, err := os.ReadFile(mdFilePath)
	if err != nil {
		return QuizItem{}, fmt.Errorf("failed to read markdown file: %w", err)
	}

	fmText, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return QuizItem{}, fmt.Errorf("%s: %w", mdFilePath, err)
	}

	var fm quizFrontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return QuizItem{}, fmt.Errorf("%s: failed to parse frontmatter: %w", mdFilePath, err)
	}

	sections, err := parseMarkdownSections(body)
	if err != nil {
		return QuizItem{}, fmt.Errorf("%s: %w", mdFilePath, err)
	}

	item := QuizItem{
		Question: sections.question,
		Answer:   sections.answer,
		Spell:    sections.spell,
		Tags:     fm.Tags,
		Comments: sections.comments,
		Criteria: buildCriteria(sections.ok, sections.ng, sections.close),
	}
	return item, nil
}

// collectMarkdownFiles はdirPath以下の*.mdファイルのパス一覧を返す．
// recursiveがtrueの場合はサブディレクトリも再帰的に辿る．
func collectMarkdownFiles(dirPath string, recursive bool) ([]string, error) {
	if !recursive {
		paths, err := filepath.Glob(filepath.Join(dirPath, "*.md"))
		if err != nil {
			return nil, fmt.Errorf("failed to glob markdown files: %w", err)
		}
		return paths, nil
	}

	var paths []string
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.EqualFold(filepath.Ext(path), ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk markdown directory: %w", err)
	}
	return paths, nil
}

// AggregateMarkdownDir は指定ディレクトリ以下の*.mdファイルをすべて読み込み，
// QuizItemのスライスとして返す．ファイル名でソートすることで，実行するたびに
// 出力順序が安定するようにする．recursiveがtrueの場合はサブディレクトリも
// 再帰的に辿る．
func AggregateMarkdownDir(dirPath string, recursive bool) ([]QuizItem, error) {
	paths, err := collectMarkdownFiles(dirPath, recursive)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	items := make([]QuizItem, 0, len(paths))
	var errs []string
	for _, p := range paths {
		item, err := ParseMarkdownFile(p)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		items = append(items, item)
	}
	if len(errs) > 0 {
		return items, fmt.Errorf("一部のファイルの変換に失敗しました:\n%s", strings.Join(errs, "\n"))
	}
	return items, nil
}
