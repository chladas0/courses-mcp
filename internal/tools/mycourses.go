package tools

import (
	"context"
	"encoding/json"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMyCourses(s *mcp.Server, api *client.APIClient) {
	s.AddTool(&mcp.Tool{
		Name:        "get_my_courses",
		Description: "Returns the list of courses the user is currently enrolled in (studying) and teaching. Each entry is a KOS course code (e.g. \"NI-VCC\"). Use this as the starting point before fetching course pages or materials.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		courses, err := api.UserCourses(ctx)
		if err != nil {
			return errResult(err), nil
		}
		return jsonResult(courses)
	})
}
