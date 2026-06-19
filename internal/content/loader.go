package content

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"gopkg.in/yaml.v3"
)

type frontmatter struct {
	Title       scalarString `yaml:"title"`
	Slug        scalarString `yaml:"slug"`
	Category    scalarString `yaml:"category"`
	Difficulty  scalarString `yaml:"difficulty"`
	ReadingTime scalarString `yaml:"reading_time"`
	Tags        tagList      `yaml:"tags"`
	Description scalarString `yaml:"description"`
}

type scalarString string

func (s *scalarString) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == 0 {
		return nil
	}
	if value.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected scalar string, got YAML kind %d", value.Kind)
	}
	*s = scalarString(strings.TrimSpace(value.Value))
	return nil
}

type tagList []string

func (t *tagList) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case 0:
		return nil
	case yaml.SequenceNode:
		for _, node := range value.Content {
			tag := strings.TrimSpace(node.Value)
			if tag != "" {
				*t = append(*t, tag)
			}
		}
	case yaml.ScalarNode:
		for _, tag := range strings.Split(value.Value, ",") {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				*t = append(*t, tag)
			}
		}
	default:
		return fmt.Errorf("expected tags sequence or comma-separated scalar, got YAML kind %d", value.Kind)
	}
	return nil
}

func Load(root string) (*Store, error) {
	markdown := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithGuessLanguage(true),
				highlighting.WithStyle("github"),
				highlighting.WithFormatOptions(chromahtml.WithClasses(true)),
			),
		),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)

	sanitizer := bluemonday.UGCPolicy()
	sanitizer.AllowAttrs("class").OnElements("code", "pre", "span", "div")
	sanitizer.AllowAttrs("id").OnElements("h2", "h3", "h4", "h5", "h6")

	var guides []Guide
	seenSlugs := map[string]string{}

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".markdown" {
			return nil
		}

		guide, err := loadGuide(path, markdown, sanitizer)
		if err != nil {
			return err
		}
		if previous, ok := seenSlugs[guide.Slug]; ok {
			return fmt.Errorf("duplicate guide slug %q in %s and %s", guide.Slug, previous, path)
		}
		seenSlugs[guide.Slug] = path
		guides = append(guides, guide)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(guides) == 0 {
		return nil, fmt.Errorf("no markdown guides found under %s", root)
	}

	return newStore(guides), nil
}

func loadGuide(path string, markdown goldmark.Markdown, sanitizer *bluemonday.Policy) (Guide, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return Guide{}, fmt.Errorf("read %s: %w", path, err)
	}

	meta, body, err := splitFrontmatter(source)
	if err != nil {
		return Guide{}, fmt.Errorf("%s: %w", path, err)
	}

	var fm frontmatter
	if len(meta) > 0 {
		if err := yaml.Unmarshal(meta, &fm); err != nil {
			return Guide{}, fmt.Errorf("parse frontmatter in %s: %w", path, err)
		}
	}

	title := string(fm.Title)
	if title == "" {
		return Guide{}, fmt.Errorf("%s: frontmatter title is required", path)
	}

	slug := Slugify(string(fm.Slug))
	if slug == "" {
		slug = Slugify(title)
	}

	category := string(fm.Category)
	if category == "" {
		category = "Uncategorized"
	}

	var rendered bytes.Buffer
	if err := markdown.Convert(body, &rendered); err != nil {
		return Guide{}, fmt.Errorf("render markdown in %s: %w", path, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		return Guide{}, fmt.Errorf("stat %s: %w", path, err)
	}

	bodyText := markdownText(body, markdown)
	readingTime := string(fm.ReadingTime)
	if readingTime == "" {
		readingTime = estimateReadingTime(bodyText)
	}

	guide := Guide{
		Title:        title,
		Slug:         slug,
		Category:     category,
		CategorySlug: Slugify(category),
		Difficulty:   string(fm.Difficulty),
		ReadingTime:  readingTime,
		Tags:         []string(fm.Tags),
		Description:  string(fm.Description),
		HTML:         string(sanitizer.SanitizeBytes(rendered.Bytes())),
		BodyText:     bodyText,
		SourcePath:   path,
		UpdatedAt:    info.ModTime().UTC().Format("2006-01-02"),
	}
	return guide, nil
}

func splitFrontmatter(source []byte) ([]byte, []byte, error) {
	normalized := strings.ReplaceAll(string(source), "\r\n", "\n")
	if !strings.HasPrefix(normalized, "---\n") {
		return nil, []byte(normalized), nil
	}

	rest := normalized[len("---\n"):]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return nil, nil, fmt.Errorf("frontmatter opener found without closing delimiter")
	}

	meta := rest[:end]
	body := rest[end+len("\n---\n"):]
	return []byte(meta), []byte(body), nil
}

func markdownText(source []byte, markdown goldmark.Markdown) string {
	doc := markdown.Parser().Parse(text.NewReader(source))

	var b strings.Builder
	_ = ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n := node.(type) {
		case *ast.Text:
			b.Write(n.Segment.Value(source))
			b.WriteByte(' ')
		case *ast.String:
			b.Write(n.Value)
			b.WriteByte(' ')
		case *ast.CodeSpan:
			for child := n.FirstChild(); child != nil; child = child.NextSibling() {
				if textNode, ok := child.(*ast.Text); ok {
					b.Write(textNode.Segment.Value(source))
					b.WriteByte(' ')
				}
			}
			return ast.WalkSkipChildren, nil
		case *ast.FencedCodeBlock:
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				b.Write(segment.Value(source))
				b.WriteByte(' ')
			}
			return ast.WalkSkipChildren, nil
		case *ast.CodeBlock:
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				b.Write(segment.Value(source))
				b.WriteByte(' ')
			}
			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	})

	return strings.Join(strings.Fields(b.String()), " ")
}

func estimateReadingTime(body string) string {
	words := len(strings.Fields(body))
	minutes := words / 220
	if words%220 != 0 {
		minutes++
	}
	if minutes < 1 {
		minutes = 1
	}
	return strconv.Itoa(minutes) + " min read"
}
