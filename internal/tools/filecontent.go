package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterFileContent(s *mcp.Server, pages *client.PageClient) {
	s.AddTool(&mcp.Tool{
		Name: "get_file_content",
		Description: "Downloads a file from courses.fit.cvut.cz and returns its text content. " +
			"Supports HTML (converted to Markdown) and PDF (text extracted). " +
			"Use this to read lecture slides, assignments, and other course materials whose links appear in get_course_page output. " +
			"Only works with courses.fit.cvut.cz URLs; other domains are rejected.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "Full URL of the file to download, must be under courses.fit.cvut.cz"
				}
			},
			"required": ["url"]
		}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := parseArgs(req)
		if err != nil {
			return errResult(err), nil
		}
		url, _ := args["url"].(string)
		if url == "" {
			return errResult(fmt.Errorf("url is required")), nil
		}

		text, err := pages.FetchFile(ctx, url)
		if err != nil {
			return errResult(err), nil
		}
		return textResult(text), nil
	})
}
