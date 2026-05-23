package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterCourseInfo(s *mcp.Server, api *client.APIClient) {
	s.AddTool(&mcp.Tool{
		Name: "get_course_info",
		Description: "Returns basic KOS metadata for a course: credit count and completion type " +
			"(EXAM, CREDIT, CREDIT_EXAM). Does NOT include teachers, schedule, materials, or any page " +
			"content; use get_course_page for those.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"course_code": {
					"type": "string",
					"description": "KOS course code, e.g. \"NI-VCC\""
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

		info, err := api.CourseInfo(ctx, courseCode)
		if err != nil {
			return errResult(err), nil
		}
		return jsonResult(info)
	})
}
