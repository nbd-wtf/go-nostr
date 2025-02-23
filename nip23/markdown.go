package nip23

import (
	stdhtml "html"
	"io"
	"regexp"
	"strings"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
	"github.com/microcosm-cc/bluemonday"
)

var nostrEveryMatcher = regexp.MustCompile(`nostr:((npub|note|nevent|nprofile|naddr)1[a-z0-9]+)\b`)

var renderer = html.NewRenderer(html.RendererOptions{
	Flags: html.HrefTargetBlank | html.SkipHTML,
	RenderNodeHook: func(w io.Writer, node ast.Node, entering bool) (ast.WalkStatus, bool) {
		switch v := node.(type) {
		case *ast.HTMLSpan:
			w.Write([]byte(stdhtml.EscapeString(string(v.Literal))))
			return ast.GoToNext, true
		case *ast.HTMLBlock:
			w.Write([]byte(stdhtml.EscapeString(string(v.Literal))))
			return ast.GoToNext, true
		}

		return ast.GoToNext, false
	},
})

func MarkdownToHTML(md string) string {
	md = strings.ReplaceAll(md, "\u00A0", " ")

	// create markdown parser with extensions
	// this parser is stateful so it must be reinitialized every time
	doc := parser.NewWithExtensions(
		parser.AutoHeadingIDs |
			parser.NoIntraEmphasis |
			parser.FencedCode |
			parser.Autolink |
			parser.Footnotes |
			parser.SpaceHeadings |
			parser.Tables,
	).Parse([]byte(md))

	// create HTML renderer with extensions
	output := string(markdown.Render(doc, renderer))

	// sanitize content
	output = sanitizeXSS(output)

	return output
}

func sanitizeXSS(html string) string {
	p := bluemonday.UGCPolicy()
	p.RequireNoFollowOnLinks(false)
	p.AllowElements("video", "source")
	p.AllowAttrs("controls", "width").OnElements("video")
	p.AllowAttrs("src", "width").OnElements("source")
	return p.Sanitize(html)
}
