package tools

import (
	"context"
	"encoding/json"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterMyInfo(s *mcp.Server, api *client.APIClient) {
	s.AddTool(&mcp.Tool{
		Name:        "get_my_info",
		Description: "Returns the authenticated user's display name, username, and personal number from UMAPI. Use this to find out who is logged in.",
		InputSchema: json.RawMessage(`{"type":"object","properties":{}}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		info, err := api.UserInfo(ctx)
		if err != nil {
			return errResult(err), nil
		}
		return jsonResult(info)
	})
}
