package client

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"golang.org/x/net/html"
)

var stripTags = map[string]bool{
	"nav":    true,
	"header": true,
	"footer": true,
}

var stripClasses = map[string]bool{
	"menubar": true,
	"sidebar": true,
}

var imagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]*)\)`)

// HTMLToMarkdown converts HTML to Markdown, stripping nav chrome and replacing
// image embeds with links.
func HTMLToMarkdown(htmlContent string) (string, error) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("HTMLToMarkdown: parse HTML: %w", err)
	}

	stripNode(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", fmt.Errorf("HTMLToMarkdown: render stripped HTML: %w", err)
	}

	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
	md, err := conv.ConvertString(buf.String())
	if err != nil {
		return "", fmt.Errorf("HTMLToMarkdown: convert to markdown: %w", err)
	}

	return imagePattern.ReplaceAllString(md, "[image: $1]($2)"), nil
}

func stripNode(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if shouldStrip(c) {
			n.RemoveChild(c)
		} else {
			stripNode(c)
		}
	}
}

func shouldStrip(n *html.Node) bool {
	if n.Type != html.ElementNode {
		return false
	}
	if stripTags[n.Data] {
		return true
	}
	for _, a := range n.Attr {
		if a.Key == "class" {
			for _, cls := range strings.Fields(a.Val) {
				if stripClasses[cls] {
					return true
				}
			}
		}
	}
	return false
}
