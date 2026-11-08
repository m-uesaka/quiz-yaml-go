// 1問1ファイルのMarkdownクイズをQuizItemに変換するためのパーサーです．
// frontmatter分離＋行ベースの単純なステートマシンで実装しており，
// 外部のMarkdownパーサーライブラリには依存しません．
//
// （パッケージの説明は doc.go に書くため，このコメントは package 句から離している．）

package quizconv

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"gopkg.in/yaml.v3"
)

// frontmatter部分のみを表す構造体．
// title / date などQuizItemに対応フィールドが無いキーは，yaml.v3が未知の
// フィールドとして無視するため定義しない．
type quizFrontmatter struct {
	Tags []string `yaml:"tags"`
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
// contentの改行コードは "\n" に正規化済みであることを前提とする．
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	const delim = "---"
	lines := strings.Split(content, "\n")
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
// 各セクションに分割する．bodyの改行コードは "\n" に正規化済みであることを
// 前提とする．
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

	for _, line := range strings.Split(body, "\n") {
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

// buildCriteria は OK/NG/Close を判定基準に変換する．
// "Close" は既存YAMLの "repeat" に対応する名前変換を行う．
// OK/NG/Closeが全て空の場合は nil を返す．
func buildCriteria(ok, ng, close []string) *Criteria {
	criteria := &Criteria{OK: ok, NG: ng, Repeat: close}
	if criteria.IsEmpty() {
		return nil
	}
	return criteria
}

// ParseMarkdownFile は1問分のMarkdownファイルをQuizItemに変換する．
func ParseMarkdownFile(mdFilePath string) (QuizItem, error) {
	raw, err := os.ReadFile(mdFilePath)
	if err != nil {
		return QuizItem{}, fmt.Errorf("Markdownファイルの読み込みに失敗しました: %w", err)
	}

	// 入力の正規化はここで1回だけ行い，以降の処理は BOM なし・"\n" 改行の文字列のみを扱う．
	// Windowsのメモ帳などで保存したファイルは先頭にUTF-8のBOMが付き，改行がCRLFになることがある．
	content := strings.TrimPrefix(string(raw), "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")

	fmText, body, err := splitFrontmatter(content)
	if err != nil {
		return QuizItem{}, fmt.Errorf("%s: %w", mdFilePath, err)
	}

	var fm quizFrontmatter
	if err := yaml.Unmarshal([]byte(fmText), &fm); err != nil {
		return QuizItem{}, fmt.Errorf("%s: frontmatterの解析に失敗しました: %w", mdFilePath, err)
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

// isMarkdownFile はファイル名の拡張子が ".md" かどうかを，大文字・小文字を
// 区別せずに判定する．再帰の有無にかかわらず同じ規則で判定するために使う．
func isMarkdownFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".md")
}

// isMarkdownFileEntry はディレクトリのエントリが読み込み対象のMarkdownファイルかどうかを返す．
// 拡張子が ".md" でも，ディレクトリと，ディレクトリを指すシンボリックリンクは対象外にする．
// （fs.DirEntry.IsDir はシンボリックリンクのリンク先を見ないため，リンクの場合は os.Stat で確認する．
// リンク切れなどで確認できない場合は対象に含め，読み込み時のエラーとして報告する．）
func isMarkdownFileEntry(path string, d fs.DirEntry) bool {
	if d.IsDir() || !isMarkdownFile(d.Name()) {
		return false
	}
	if d.Type()&fs.ModeSymlink != 0 {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return false
		}
	}
	return true
}

// collectMarkdownFiles はdirPath以下のMarkdownファイル（拡張子が ".md"．
// 大文字・小文字は区別しない）のパス一覧を返す．ディレクトリは拡張子に
// かかわらず対象外とする．recursiveがtrueの場合はサブディレクトリも
// 再帰的に辿る．
//
// 再帰の有無で振る舞いが変わらないように，収集の前に dirPath を確認し，
// 存在しない場合（パスの途中がファイルの場合を含む）・ディレクトリでない場合は
// エラーを返す．存在しない場合のエラーは元のエラーをラップするため，
// errors.Is(err, fs.ErrNotExist) で判定できる．
//
// dirPath がディレクトリへのシンボリックリンクの場合も辿れるように，
// リンク先を解決してから1回の filepath.WalkDir で収集する．（再帰しない場合は
// サブディレクトリを fs.SkipDir で飛ばす．）返すパスは dirPath を起点とする．
func collectMarkdownFiles(dirPath string, recursive bool) ([]string, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
			return nil, fmt.Errorf("Markdownディレクトリが見つかりません: %w", err)
		}
		return nil, fmt.Errorf("Markdownディレクトリを確認できません: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("Markdownディレクトリではありません（ファイルが指定されています）: %s", dirPath)
	}
	root, err := filepath.EvalSymlinks(dirPath)
	if err != nil {
		return nil, fmt.Errorf("Markdownディレクトリを確認できません: %w", err)
	}

	var paths []string
	err = filepath.WalkDir(root, func(walked string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && walked != root && !recursive {
			return fs.SkipDir
		}
		rel, err := filepath.Rel(root, walked)
		if err != nil {
			return err
		}
		path := filepath.Join(dirPath, rel)
		if isMarkdownFileEntry(path, d) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("Markdownディレクトリを読み込めません: %w", err)
	}
	return paths, nil
}

// AggregateMarkdownDir は指定ディレクトリ以下の*.mdファイルをすべて読み込み，
// QuizItemのスライスとして返す．パスでソートすることで，実行するたびに
// 出力順序が安定するようにする．recursiveがtrueの場合はサブディレクトリも
// 再帰的に辿る．
//
// ディレクトリが存在しない場合・ディレクトリでない場合に加えて，Markdownファイルが
// 1つも見つからない場合もエラーを返す．（パスの指定ミスに気づけるようにし，
// 既存の出力ファイルが空のリストで上書きされないようにするため．）
//
// 一部のファイルの変換に失敗した場合は，変換できたQuizItemと，各ファイルの
// エラーを errors.Join で集約したエラーの両方を返す．呼び出し側は errors.Is /
// errors.As で個々のエラーを扱える．
func AggregateMarkdownDir(dirPath string, recursive bool) ([]QuizItem, error) {
	items, _, err := aggregateMarkdownDir(dirPath, recursive)
	return items, err
}

// aggregateMarkdownDir は AggregateMarkdownDir と同じ処理を行い，各QuizItemの元になった
// ファイルのパスも返す．sources[i] は items[i] のファイルのパス．
func aggregateMarkdownDir(dirPath string, recursive bool) (items []QuizItem, sources []string, err error) {
	paths, err := collectMarkdownFiles(dirPath, recursive)
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("Markdownファイル（拡張子が .md のファイル）が見つかりません: %s", dirPath)
	}
	sort.Strings(paths)

	items = make([]QuizItem, 0, len(paths))
	sources = make([]string, 0, len(paths))
	var errs []error
	for _, p := range paths {
		item, err := ParseMarkdownFile(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		items = append(items, item)
		sources = append(sources, p)
	}
	if len(errs) > 0 {
		return items, sources, fmt.Errorf("一部のファイルの変換に失敗しました:\n%w", errors.Join(errs...))
	}
	return items, sources, nil
}
