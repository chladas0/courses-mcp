package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterCoursePage(s *mcp.Server, pages *client.PageClient) {
	s.AddTool(&mcp.Tool{
		Name: "get_course_page",
		Description: "Fetches a page from a course site and returns its content as Markdown, preserving links. " +
			"Use this to find teachers, course overview, schedule, links to lectures and materials. " +
			"The default path \"/\" returns the course homepage. " +
			"Pass a relative path (e.g. \"/tutorials/\") to fetch a subpage. " +
			"Links in the output can be passed to get_file_content (read as text) or download_file (save to disk).",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"course_code": {
					"type": "string",
					"description": "KOS course code, e.g. \"NI-VCC\""
				},
				"path": {
					"type": "string",
					"description": "Relative path on the course site, e.g. \"/tutorials/\". Defaults to \"/\"."
				}
			},
			"required": ["course_code"]
		}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := parseArgs(req)
		if err != nil {
			return errResult(err), nil
		}
		courseCode, _ := args["course_code"].(string)
		if courseCode == "" {
			return errResult(fmt.Errorf("course_code is required")), nil
		}
		path, _ := args["path"].(string)
		if path == "" {
			path = "/"
		}

		md, err := pages.FetchPage(ctx, courseCode, path)
		if err != nil {
			return errResult(err), nil
		}
		return textResult(md), nil
	})
}
