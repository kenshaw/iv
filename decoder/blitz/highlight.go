package blitz

import (
	"context"
	"fmt"
	"html"
	"io/fs"
	"strings"
	"sync"

	"github.com/kenshaw/iv/ivctx"
	"github.com/xo/blitz"
	"github.com/xo/transit"
	"github.com/xo/transit/grammars/bash"
	"github.com/xo/transit/grammars/c"
	"github.com/xo/transit/grammars/cpp"
	golang "github.com/xo/transit/grammars/go"
	htmlgrammar "github.com/xo/transit/grammars/html"
	"github.com/xo/transit/grammars/java"
	"github.com/xo/transit/grammars/javascript"
	"github.com/xo/transit/grammars/json"
	"github.com/xo/transit/grammars/python"
	"github.com/xo/transit/grammars/ruby"
	"github.com/xo/transit/grammars/rust"
	"github.com/xo/transit/grammars/sql"
	"github.com/xo/transit/grammars/typescript/typescript"
	"github.com/xo/transit/styles"
)

// The styles that color the code of a light and of a dark document. They
// follow the syntax colors of GitHub.
const (
	lightStyle = "github"
	darkStyle  = "github-dark"
)

// grammar is a language of transit, and the files of its highlight query.
// The query is compiled on the first use of the language, and goroutines
// share it.
type grammar struct {
	language func() *transit.Language
	queries  fs.FS
	// paths are the files of the highlight query, in the order of the
	// tree-sitter.json of the grammar. A later pattern wins over an earlier
	// one on the same node, as in transit.
	paths []string
	query func() (*transit.Query, error)
}

// grammars are the languages that a fenced code block is highlighted in.
var grammars = map[string]*grammar{
	"bash":       newGrammar(bash.Language, bash.Queries, "queries/highlights.scm"),
	"c":          newGrammar(c.Language, c.Queries, "queries/highlights.scm"),
	"cpp":        newGrammar(cpp.Language, cpp.Queries, "queries/c/highlights.scm", "queries/highlights.scm"),
	"go":         newGrammar(golang.Language, golang.Queries, "queries/highlights.scm"),
	"html":       newGrammar(htmlgrammar.Language, htmlgrammar.Queries, "queries/highlights.scm"),
	"java":       newGrammar(java.Language, java.Queries, "queries/highlights.scm"),
	"javascript": newGrammar(javascript.Language, javascript.Queries, "queries/highlights.scm", "queries/highlights-jsx.scm", "queries/highlights-params.scm"),
	"json":       newGrammar(json.Language, json.Queries, "queries/highlights.scm"),
	"python":     newGrammar(python.Language, python.Queries, "queries/highlights.scm"),
	"ruby":       newGrammar(ruby.Language, ruby.Queries, "queries/highlights.scm"),
	"rust":       newGrammar(rust.Language, rust.Queries, "queries/highlights.scm"),
	"sql":        newGrammar(sql.Language, sql.Queries, "queries/highlights.scm"),
	"typescript": newGrammar(typescript.Language, typescript.Queries, "queries/highlights.scm", "queries/javascript/highlights.scm"),
}

// otherNames are the other names that a fenced code block can give for the
// language of a grammar.
var otherNames = map[string][]string{
	"bash":       {"sh", "shell", "zsh"},
	"c":          {"h"},
	"cpp":        {"c++", "cc", "cxx", "hpp"},
	"go":         {"golang"},
	"html":       {"htm"},
	"javascript": {"js", "jsx", "mjs", "cjs"},
	"python":     {"py"},
	"ruby":       {"rb"},
	"rust":       {"rs"},
	// the dialects of sql that transit has grammars for, except clickhouse,
	// neo4j and ydb, use the generic grammar
	"sql": {
		"cql",
		"duckdb",
		"mariadb", "mysql",
		"oracle", "plsql",
		"pgsql", "plpgsql", "postgres", "postgresql", "psql",
		"sqlite", "sqlite3",
		"mssql", "sqlserver", "t-sql", "tsql",
		"surql", "surrealdb", "surrealql",
		"presto", "trino",
		"usql",
	},
	"typescript": {"ts"},
}

// aliases maps each name of a language, in lower case, to the name of its
// grammar.
var aliases = func() map[string]string {
	m := make(map[string]string)
	for name := range grammars {
		m[name] = name
		for _, other := range otherNames[name] {
			m[other] = name
		}
	}
	return m
}()

// newGrammar returns a grammar whose highlight query is the files at paths
// in queries, joined in order.
func newGrammar(language func() *transit.Language, queries fs.FS, paths ...string) *grammar {
	g := &grammar{language: language, queries: queries, paths: paths}
	g.query = sync.OnceValues(g.compile)
	return g
}

// compile reads the files of the highlight query and compiles them as one
// query, so that the order of the patterns is the order of the files.
func (g *grammar) compile() (*transit.Query, error) {
	var b strings.Builder
	for _, name := range g.paths {
		buf, err := fs.ReadFile(g.queries, name)
		if err != nil {
			return nil, fmt.Errorf("highlight query %s: %w", name, err)
		}
		b.Write(buf)
		b.WriteByte('\n')
	}
	q, err := transit.NewQuery(g.language(), b.String())
	if err != nil {
		return nil, fmt.Errorf("highlight query: %w", err)
	}
	return q, nil
}

// highlighter returns the hook of blitz that colors the fenced code blocks of
// a Markdown document. The colors are those of the GitHub style that matches
// the color scheme of the document. A block in a language that has no
// grammar keeps its plain look.
//
// The hook keeps a parser and a query cursor, which goroutines must not
// share, so each document makes its own. It also keeps the html of each
// block, so that a second render of the document does not parse it again.
func highlighter(ctx context.Context) blitz.Highlighter {
	name := lightStyle
	if ivctx.Get(ctx).BlitzDark {
		name = darkStyle
	}
	style, ok := styles.Get(name)
	if !ok {
		ivctx.Logf(ctx, "blitz highlight: no style %q", name)
		return nil
	}
	parser, cursor := transit.NewParser(), transit.NewQueryCursor()
	done := make(map[[2]string]string)
	return func(lang, code string) (string, bool) {
		key := [2]string{lang, code}
		s, ok := done[key]
		if !ok {
			var err error
			if s, err = highlight(ctx, parser, cursor, style, lang, code); err != nil {
				ivctx.Logf(ctx, "blitz highlight %q: %v", lang, err)
			}
			done[key] = s
		}
		return s, s != ""
	}
}

// highlight returns the code of a block as html, with a span for each run of
// text that a capture of the highlight query covers. It returns "" for a
// language that has no grammar.
func highlight(ctx context.Context, parser *transit.Parser, cursor *transit.QueryCursor, style *styles.Style, lang, code string) (string, error) {
	// the language is the first word of the info string, and an attribute
	// can touch it, as in rust,ignore or js{1,3}
	lang, _, _ = strings.Cut(strings.ToLower(lang), ",")
	lang, _, _ = strings.Cut(lang, "{")
	g, ok := grammars[aliases[lang]]
	if !ok {
		return "", nil
	}
	q, err := g.query()
	if err != nil {
		return "", err
	}
	if err := parser.SetLanguage(g.language()); err != nil {
		return "", err
	}
	src := []byte(code)
	tree, err := parser.Parse(ctx, src, nil)
	if err != nil {
		return "", err
	}
	// the capture name of each byte. A later capture is inside an earlier
	// one, after it, or a later pattern on the same node, so the innermost
	// capture wins, and of two patterns on one node the last wins
	names := make([]string, len(src))
	root, captureNames := tree.RootNode(), q.CaptureNames()
	cursor.SetByteRange(root.StartByte(), root.EndByte())
	for m, i := range cursor.Captures(ctx, q, root, src) {
		c := m.Captures[i]
		name := captureNames[c.Index]
		if strings.HasPrefix(name, "_") {
			// a name that starts with _ belongs to a predicate
			continue
		}
		for j := c.Node.StartByte(); j < c.Node.EndByte() && j < len(names); j++ {
			names[j] = name
		}
	}
	var b strings.Builder
	for i := 0; i < len(src); {
		j := i + 1
		for j < len(src) && names[j] == names[i] {
			j++
		}
		text := html.EscapeString(string(src[i:j]))
		if css := inline(style, names[i]); css != "" {
			fmt.Fprintf(&b, `<span style="%s">%s</span>`, css, text)
		} else {
			b.WriteString(text)
		}
		i = j
	}
	return b.String(), nil
}

// inline returns the inline css of a capture name, or "" for text that no
// capture names, which keeps the color of the page.
func inline(style *styles.Style, name string) string {
	if name == "" {
		return ""
	}
	e := style.Lookup(name)
	var css []string
	if e.Fg.IsSet() {
		r, g, b := e.Fg.RGB()
		css = append(css, fmt.Sprintf("color:#%02x%02x%02x", r, g, b))
	}
	if e.Bold {
		css = append(css, "font-weight:bold")
	}
	if e.Italic {
		css = append(css, "font-style:italic")
	}
	return strings.Join(css, ";")
}
