package quiz_yaml_converter

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseMarkdownFile_FullySpecified(t *testing.T) {
	content := `---
title: 自作問題-ハヤカワ・ポケット・ミステリ
date: 2026-06-12
tags:
    - ガツオ
---
## Question

ミッキー・スピレインの『大いなる殺人』を第一冊として1953年に刊行された，
早川書房の翻訳ミステリーシリーズは何でしょう？

## Answer

ハヤカワ・ポケット・ミステリ

## Spell

Hayakawa Pocket Mystery

## Criteria

### OK

- 「ハヤカワ・ミステリ」

### NG

- 「ハヤカワ・ミステリ文庫」（別レーベル）

### Close

## Comment
`
	dir := t.TempDir()
	path := writeFile(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Answer != "ハヤカワ・ポケット・ミステリ" {
		t.Errorf("Answer = %q", item.Answer)
	}
	if item.Spell != "Hayakawa Pocket Mystery" {
		t.Errorf("Spell = %q", item.Spell)
	}
	if len(item.Tags) != 1 || item.Tags[0] != "ガツオ" {
		t.Errorf("Tags = %v", item.Tags)
	}
	wantCriteria := &Criteria{
		OK: []string{"「ハヤカワ・ミステリ」"},
		NG: []string{"「ハヤカワ・ミステリ文庫」（別レーベル）"},
	}
	if !reflect.DeepEqual(item.Criteria, wantCriteria) {
		t.Errorf("Criteria = %+v, want %+v（### Close が空なら Repeat は nil）", item.Criteria, wantCriteria)
	}
	if item.Comments != nil {
		t.Errorf("Comments = %v, want nil", item.Comments)
	}
}

func TestParseMarkdownFile_EmptyOptionalSections(t *testing.T) {
	content := `---
title: 自作問題-Vessel
date: 2026-03-20
---
## Question

デッキと階段が蜂の巣状に外周を取り囲む建築作品は何でしょう？

## Answer

Vessel

## Spell

## Criteria

### OK

### NG

### Close

## Comment
`
	dir := t.TempDir()
	path := writeFile(t, dir, "vessel.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Spell != "" {
		t.Errorf("Spell = %q, want empty", item.Spell)
	}
	if item.Comments != nil {
		t.Errorf("Comments = %v, want nil", item.Comments)
	}
	if item.Criteria != nil {
		t.Errorf("Criteria = %v, want nil", item.Criteria)
	}
	if item.Tags != nil {
		t.Errorf("Tags = %v, want nil (frontmatter has no tags key)", item.Tags)
	}
}

func TestParseMarkdownFile_MultilineQuestionWithoutBlankLine(t *testing.T) {
	content := `---
title: 自作問題-ジェプツンタンパ1世
date: 2026-06-12
tags:
    - ガツオ
---
## Question

1974年にモンゴルで化石が発見された恐竜に献名されているのは誰でしょう？
【通常】ジュンガルの首長ガルダンと対立したときには宗教指導者は誰でしょう？

## Answer

ジェプツンタンパ1世

## Spell

Jebtsundamba Khutuktu

## Criteria

### OK

- ウンドル・ゲゲン・ザナザバル（Öndör Gegeen Zanabazar）
- ジュニャーナヴァジュラ（Jñānavajra）

### NG

### Close

## Comment
`
	dir := t.TempDir()
	path := writeFile(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantQuestion := "1974年にモンゴルで化石が発見された恐竜に献名されているのは誰でしょう？\n" +
		"【通常】ジュンガルの首長ガルダンと対立したときには宗教指導者は誰でしょう？"
	if item.Question != wantQuestion {
		t.Errorf("Question = %q, want %q", item.Question, wantQuestion)
	}
}

func TestParseMarkdownFile_CommentWithMultipleLinesAndMarkdownLink(t *testing.T) {
	content := `---
title: 自作問題-Vessel
date: 2026-03-20
---
## Question

質問文

## Answer

Vessel

## Spell

## Criteria

### OK

### NG

### Close

## Comment

[4人が命を落としたNYの巨大アートが3年ぶりに公開再開 | ARTnews JAPAN](https://artnewsjapan.com/article/2792)
これは補足コメントの2行目である．

別の段落のコメント．
`
	dir := t.TempDir()
	path := writeFile(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantFirst := "[4人が命を落としたNYの巨大アートが3年ぶりに公開再開 | ARTnews JAPAN](https://artnewsjapan.com/article/2792)\n" +
		"これは補足コメントの2行目である．"
	wantSecond := "別の段落のコメント．"
	if len(item.Comments) != 2 {
		t.Fatalf("Comments = %v, want 2 elements", item.Comments)
	}
	if item.Comments[0] != wantFirst {
		t.Errorf("Comments[0] = %q, want %q", item.Comments[0], wantFirst)
	}
	if item.Comments[1] != wantSecond {
		t.Errorf("Comments[1] = %q, want %q", item.Comments[1], wantSecond)
	}
}

func TestParseMarkdownFile_CloseMapsToRepeat(t *testing.T) {
	content := `---
title: 自作問題-テスト
date: 2026-01-01
---
## Question

問題文

## Answer

答え

## Spell

## Criteria

### OK

- 別解1

### NG

### Close

- もう一度1
- もう一度2

## Comment
`
	dir := t.TempDir()
	path := writeFile(t, dir, "test.md", content)

	item, err := ParseMarkdownFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := &Criteria{OK: []string{"別解1"}, Repeat: []string{"もう一度1", "もう一度2"}}
	if !reflect.DeepEqual(item.Criteria, want) {
		t.Errorf("Criteria = %+v, want %+v（### NG が空なら NG は nil）", item.Criteria, want)
	}
}

// markdownFixture はAnswerだけが異なる最小限の1問分のMarkdownを返す．
func markdownFixture(answer string) string {
	return "---\ntitle: 自作問題-" + answer + "\ndate: 2026-01-01\n---\n" +
		"## Question\n\n問題文\n\n## Answer\n\n" + answer + "\n\n## Spell\n\n## Criteria\n\n### OK\n\n### NG\n\n### Close\n\n## Comment\n"
}

// CRLF改行のファイルでも，LF改行のファイルと同じ結果になることを確認する．
func TestParseMarkdownFile_CRLFLineEndings(t *testing.T) {
	// Arrange
	lf := `---
title: 自作問題-CRLF
date: 2026-01-01
tags:
    - タグ1
---
## Question

問題文1行目
問題文2行目

## Answer

答え

## Spell

Answer

## Criteria

### OK

- 別解1

### NG

- 誤答1

### Close

- もう一度1

## Comment

コメント1行目
コメント2行目

別の段落
`
	dir := t.TempDir()
	crlfPath := writeFile(t, dir, "crlf.md", strings.ReplaceAll(lf, "\n", "\r\n"))
	want := QuizItem{
		Question: "問題文1行目\n問題文2行目",
		Answer:   "答え",
		Spell:    "Answer",
		Tags:     []string{"タグ1"},
		Comments: []string{"コメント1行目\nコメント2行目", "別の段落"},
		Criteria: &Criteria{OK: []string{"別解1"}, NG: []string{"誤答1"}, Repeat: []string{"もう一度1"}},
	}

	// Act
	got, err := ParseMarkdownFile(crlfPath)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseMarkdownFile(CRLF) = %+v, want %+v", got, want)
	}
}

// QuizItemに対応しないfrontmatterのキー（title / date など）は無視され，
// 先頭にUTF-8のBOMが付いたファイル（CRLF改行を含む）でも，BOMなしのファイルと同じ結果になることを確認する．
func TestParseMarkdownFile_UTF8BOM(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{name: "LF", content: "\ufeff" + markdownFixture("答え")},
		{name: "CRLF", content: "\ufeff" + strings.ReplaceAll(markdownFixture("答え"), "\n", "\r\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			want, err := ParseMarkdownFile(writeFile(t, dir, "plain.md", markdownFixture("答え")))
			if err != nil {
				t.Fatalf("failed to parse file without BOM: %v", err)
			}
			path := writeFile(t, dir, "bom.md", tt.content)

			// Act
			got, err := ParseMarkdownFile(path)

			// Assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ParseMarkdownFile() = %+v, want %+v", got, want)
			}
		})
	}
}

// tagsだけが読み込まれることを確認する．
func TestParseMarkdownFile_IgnoresUnusedFrontmatterKeys(t *testing.T) {
	// Arrange
	content := "---\ntitle: 自作問題-テスト\ndate: 2026-01-01\nauthor: 誰か\ntags:\n    - タグ1\n---\n" +
		"## Question\n\n問題文\n\n## Answer\n\n答え\n"
	path := writeFile(t, t.TempDir(), "test.md", content)

	// Act
	item, err := ParseMarkdownFile(path)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(item.Tags, []string{"タグ1"}) {
		t.Errorf("Tags = %v, want [タグ1]", item.Tags)
	}
	if item.Answer != "答え" {
		t.Errorf("Answer = %q, want %q", item.Answer, "答え")
	}
}

// 拡張子の判定は，再帰の有無にかかわらず大文字・小文字を区別せず，
// 拡張子が ".md" のディレクトリと，ディレクトリを指すシンボリックリンクは対象外になることを確認する．
func TestCollectMarkdownFiles_ExtensionMatching(t *testing.T) {
	tests := []struct {
		name      string
		recursive bool
	}{
		{name: "再帰しない", recursive: false},
		{name: "再帰する", recursive: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			for _, name := range []string{"a.md", "B.MD", "c.Md", "d.txt", "e.markdown"} {
				writeFile(t, dir, name, "")
			}
			if err := os.Mkdir(filepath.Join(dir, "x.md"), 0o755); err != nil {
				t.Fatalf("failed to create directory: %v", err)
			}
			// ディレクトリを指すシンボリックリンクは対象外，ファイルを指すシンボリックリンクは対象にする．
			// リンク先はdirの外に置き，再帰する場合にリンク先が別に数えられないようにする．
			target := t.TempDir()
			targetFile := writeFile(t, target, "target.md", "")
			if err := os.Symlink(target, filepath.Join(dir, "link-to-dir.md")); err != nil {
				t.Skipf("symlinks are not supported: %v", err)
			}
			if err := os.Symlink(targetFile, filepath.Join(dir, "link-to-file.md")); err != nil {
				t.Skipf("symlinks are not supported: %v", err)
			}
			want := []string{
				filepath.Join(dir, "B.MD"),
				filepath.Join(dir, "a.md"),
				filepath.Join(dir, "c.Md"),
				filepath.Join(dir, "link-to-file.md"),
			}

			// Act
			got, err := collectMarkdownFiles(dir, tt.recursive)

			// Assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sort.Strings(got)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("collectMarkdownFiles(recursive=%v) = %v, want %v", tt.recursive, got, want)
			}
		})
	}
}

func TestAggregateMarkdownDir_SortedAndAggregated(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "c.md", markdownFixture("C"))
	writeFile(t, dir, "a.md", markdownFixture("A"))
	writeFile(t, dir, "b.md", markdownFixture("B"))

	items, err := AggregateMarkdownDir(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
	want := []string{"A", "B", "C"}
	for i, w := range want {
		if items[i].Answer != w {
			t.Errorf("items[%d].Answer = %q, want %q", i, items[i].Answer, w)
		}
	}
}

func TestAggregateMarkdownDir_Recursive(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}
	writeFile(t, dir, "top.md", markdownFixture("TOP"))
	writeFile(t, sub, "nested.md", markdownFixture("NESTED"))

	nonRecursive, err := AggregateMarkdownDir(dir, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(nonRecursive) != 1 {
		t.Fatalf("non-recursive len(items) = %d, want 1", len(nonRecursive))
	}

	recursive, err := AggregateMarkdownDir(dir, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(recursive) != 2 {
		t.Fatalf("recursive len(items) = %d, want 2", len(recursive))
	}
}

// ディレクトリへのシンボリックリンクを指定した場合も，再帰の有無にかかわらず
// リンク先のMarkdownファイルを収集し，パスは指定したリンクを起点とすることを確認する．
func TestCollectMarkdownFiles_SymlinkRoot(t *testing.T) {
	for _, recursive := range []bool{false, true} {
		t.Run(fmt.Sprintf("recursive=%v", recursive), func(t *testing.T) {
			// Arrange
			target := t.TempDir()
			writeFile(t, target, "top.md", markdownFixture("TOP"))
			sub := filepath.Join(target, "sub")
			if err := os.Mkdir(sub, 0o755); err != nil {
				t.Fatalf("failed to create subdirectory: %v", err)
			}
			writeFile(t, sub, "nested.md", markdownFixture("NESTED"))
			link := filepath.Join(t.TempDir(), "link")
			if err := os.Symlink(target, link); err != nil {
				t.Skipf("symlinks are not supported: %v", err)
			}
			want := []string{filepath.Join(link, "top.md")}
			if recursive {
				want = append(want, filepath.Join(link, "sub", "nested.md"))
			}

			// Act
			got, err := collectMarkdownFiles(link, recursive)

			// Assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sort.Strings(got)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("collectMarkdownFiles(recursive=%v) = %v, want %v", recursive, got, want)
			}
		})
	}
}

func TestAggregateMarkdownDir_PartialFailureReportsFileName(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "good.md", markdownFixture("OK"))
	brokenPath := writeFile(t, dir, "broken.md", "no frontmatter here")

	items, err := AggregateMarkdownDir(dir, false)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !strings.Contains(err.Error(), brokenPath) {
		t.Errorf("error message %q does not contain broken file path %q", err.Error(), brokenPath)
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1 (only the successfully parsed file)", len(items))
	}
}

// 複数のファイルの変換に失敗した場合，各ファイルのエラーが集約され，
// errors.As で個々のエラーを取り出せることを確認する．
func TestAggregateMarkdownDir_MultipleFailuresAreJoined(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	writeFile(t, dir, "good.md", markdownFixture("OK"))
	noFrontmatterPath := writeFile(t, dir, "no_frontmatter.md", "no frontmatter here")
	// tagsに文字列のリストではなくマッピングを指定し，yaml.v3の *yaml.TypeError を発生させる
	badTagsPath := writeFile(t, dir, "bad_tags.md", "---\ntags:\n    key: value\n---\n## Answer\n\n答え\n")

	// Act
	items, err := AggregateMarkdownDir(dir, false)

	// Assert
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if len(items) != 1 {
		t.Errorf("len(items) = %d, want 1 (only the successfully parsed file)", len(items))
	}
	for _, p := range []string{noFrontmatterPath, badTagsPath} {
		if !strings.Contains(err.Error(), p) {
			t.Errorf("error message %q does not contain file path %q", err.Error(), p)
		}
	}
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		t.Errorf("errors.As(err, *yaml.TypeError) = false, want true: %v", err)
	}
}

// Markdownファイル以外のファイルが混在していても，再帰の有無にかかわらず
// Markdownファイルだけが読み込まれることを確認する．
func TestAggregateMarkdownDir_IgnoresNonMarkdownFiles(t *testing.T) {
	tests := []struct {
		name      string
		recursive bool
	}{
		{name: "再帰しない", recursive: false},
		{name: "再帰する", recursive: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			writeFile(t, dir, "quiz.md", markdownFixture("答え"))
			writeFile(t, dir, "memo.txt", "Markdownではないファイル")

			// Act
			items, err := AggregateMarkdownDir(dir, tt.recursive)

			// Assert
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(items) != 1 || items[0].Answer != "答え" {
				t.Errorf("AggregateMarkdownDir(recursive=%v) = %+v, want 1 item with Answer %q", tt.recursive, items, "答え")
			}
		})
	}
}

// 存在しないディレクトリ・ファイル・Markdownファイルを含まないディレクトリを指定した場合，
// 再帰の有無にかかわらず，同じエラーになることを確認する．
func TestAggregateMarkdownDir_InvalidDirectory(t *testing.T) {
	tests := []struct {
		name string
		// pathはフィクスチャ用のディレクトリを受け取り，AggregateMarkdownDirに渡すパスを返す
		path         func(t *testing.T, dir string) string
		wantMsg      string
		wantNotExist bool
	}{
		{
			name: "存在しないディレクトリ",
			path: func(_ *testing.T, dir string) string {
				return filepath.Join(dir, "missing")
			},
			wantMsg:      "Markdownディレクトリが見つかりません",
			wantNotExist: true,
		},
		{
			name: "ディレクトリではなくファイル",
			path: func(t *testing.T, dir string) string {
				return writeFile(t, dir, "quiz.md", markdownFixture("答え"))
			},
			wantMsg: "Markdownディレクトリではありません",
		},
		{
			name: "パスの途中がファイル",
			path: func(t *testing.T, dir string) string {
				return filepath.Join(writeFile(t, dir, "quiz.md", markdownFixture("答え")), "sub")
			},
			wantMsg: "Markdownディレクトリが見つかりません",
		},
		{
			name: "空のディレクトリ",
			path: func(_ *testing.T, dir string) string {
				return dir
			},
			wantMsg: "Markdownファイル（拡張子が .md のファイル）が見つかりません",
		},
		{
			name: "Markdownファイル以外のファイルのみを含むディレクトリ",
			path: func(t *testing.T, dir string) string {
				writeFile(t, dir, "memo.txt", "")
				writeFile(t, dir, "quiz.markdown", markdownFixture("答え"))
				return dir
			},
			wantMsg: "Markdownファイル（拡張子が .md のファイル）が見つかりません",
		},
	}

	for _, tt := range tests {
		for _, recursive := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/recursive=%v", tt.name, recursive), func(t *testing.T) {
				// Arrange
				path := tt.path(t, t.TempDir())

				// Act
				items, err := AggregateMarkdownDir(path, recursive)

				// Assert
				if err == nil {
					t.Fatalf("expected error, got nil (items: %+v)", items)
				}
				if items != nil {
					t.Errorf("items = %+v, want nil", items)
				}
				if !strings.Contains(err.Error(), tt.wantMsg) {
					t.Errorf("error message %q does not contain %q", err.Error(), tt.wantMsg)
				}
				if !strings.Contains(err.Error(), path) {
					t.Errorf("error message %q does not contain path %q", err.Error(), path)
				}
				if got := errors.Is(err, fs.ErrNotExist); got != tt.wantNotExist {
					t.Errorf("errors.Is(err, fs.ErrNotExist) = %v, want %v", got, tt.wantNotExist)
				}
			})
		}
	}
}

func TestSaveYAMLData_RoundTrip(t *testing.T) {
	original := []QuizItem{
		{
			Question: "問題文1\n複数行",
			Answer:   "答え1",
			Spell:    "Answer1",
			Tags:     []string{"タグ1", "タグ2"},
			Comments: []string{"コメント1", "コメント2\n2行目"},
			Criteria: &Criteria{
				OK:     []string{"別解1"},
				NG:     []string{"誤答1"},
				Repeat: []string{"もう一度1"},
			},
		},
		{
			Question: "問題文2",
			Answer:   "答え2",
		},
	}

	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "out.yaml")

	if err := SaveYAMLData(original, yamlPath); err != nil {
		t.Fatalf("SaveYAMLData failed: %v", err)
	}

	loaded, err := LoadYAMLData(yamlPath)
	if err != nil {
		t.Fatalf("LoadYAMLData failed: %v", err)
	}

	if !reflect.DeepEqual(original, loaded) {
		t.Errorf("round-trip mismatch:\noriginal = %+v\nloaded   = %+v", original, loaded)
	}
}

// Criteriaを構造体にしても，書き出されるYAMLのフォーマット（criteriaのキー名と省略の規則）が変わらないことを確認する．
func TestSaveYAMLData_CriteriaFormat(t *testing.T) {
	// Arrange
	items := []QuizItem{
		{Question: "問題1", Answer: "答え1", Criteria: &Criteria{OK: []string{"別解1"}, Repeat: []string{"もう一度1"}}},
		{Question: "問題2", Answer: "答え2"},
	}
	yamlPath := filepath.Join(t.TempDir(), "out.yaml")
	expected := `- question: 問題1
  answer: 答え1
  spell: ""
  criteria:
    ok:
        - 別解1
    repeat:
        - もう一度1
- question: 問題2
  answer: 答え2
  spell: ""
`

	// Act
	err := SaveYAMLData(items, yamlPath)

	// Assert
	if err != nil {
		t.Fatalf("SaveYAMLData failed: %v", err)
	}
	got := readFile(t, yamlPath)
	if got != expected {
		t.Errorf("SaveYAMLData output =\n%s\nwant\n%s", got, expected)
	}
}

// Markdownファイルを読み込めない場合と，frontmatterをYAMLとして解析できない場合のエラーメッセージを確認する．
func TestParseMarkdownFile_ReadAndFrontmatterErrors(t *testing.T) {
	tests := []struct {
		name        string
		content     *string // nil の場合はファイルを作らない
		wantMessage func(path string) string
	}{
		{
			name:        "missing file",
			content:     nil,
			wantMessage: func(string) string { return "Markdownファイルの読み込みに失敗しました: " },
		},
		{
			name:        "invalid frontmatter",
			content:     ptr("---\ntitle: [閉じていない配列\n---\n## Question\n\n問題\n\n## Answer\n\n答え\n"),
			wantMessage: func(path string) string { return path + ": frontmatterの解析に失敗しました: " },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := filepath.Join(t.TempDir(), "quiz.md")
			if tt.content != nil {
				if err := os.WriteFile(path, []byte(*tt.content), 0o644); err != nil {
					t.Fatalf("failed to write markdown: %v", err)
				}
			}
			want := tt.wantMessage(path)

			// Act
			_, err := ParseMarkdownFile(path)

			// Assert
			if err == nil || !strings.HasPrefix(err.Error(), want) {
				t.Errorf("ParseMarkdownFile() error = %v, want prefix %q", err, want)
			}
		})
	}
}
