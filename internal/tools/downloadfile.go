package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterDownloadFile(s *mcp.Server, pages *client.PageClient) {
	s.AddTool(&mcp.Tool{
		Name: "download_file",
		Description: "Downloads a raw file from courses.fit.cvut.cz and saves it to the local filesystem. " +
			"Supports any file type (PDF, ZIP, images, etc.). " +
			"Use this when you need the actual file on disk rather than its text content. " +
			"Only works with courses.fit.cvut.cz URLs; other domains are rejected. " +
			"If destination is omitted the file is saved to ~/Downloads/ using the filename from the URL.",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "Full URL of the file to download, must be under courses.fit.cvut.cz"
				},
				"destination": {
					"type": "string",
					"description": "Local path to save the file. May be a directory (filename inferred from URL) or a full file path. Defaults to ~/Downloads/<filename>."
				}
			},
			"required": ["url"]
		}`),
	}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := parseArgs(req)
		if err != nil {
			return errResult(err), nil
		}
		rawURL, _ := args["url"].(string)
		if rawURL == "" {
			return errResult(fmt.Errorf("url is required")), nil
		}
		destination, _ := args["destination"].(string)

		data, _, err := pages.FetchRaw(ctx, rawURL)
		if err != nil {
			return errResult(err), nil
		}

		savePath, err := resolveDest(rawURL, destination)
		if err != nil {
			return errResult(err), nil
		}

		if err := os.MkdirAll(filepath.Dir(savePath), 0o755); err != nil {
			return errResult(fmt.Errorf("create directory: %w", err)), nil
		}
		if err := os.WriteFile(savePath, data, 0o644); err != nil {
			return errResult(fmt.Errorf("write file: %w", err)), nil
		}

		return textResult(fmt.Sprintf("Saved %d bytes to %s", len(data), savePath)), nil
	})
}

func resolveDest(rawURL, destination string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	filename := filepath.Base(parsed.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "download"
	}

	if destination == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("find home dir: %w", err)
		}
		return filepath.Join(home, "Downloads", filename), nil
	}

	info, err := os.Stat(destination)
	if err == nil && info.IsDir() {
		return filepath.Join(destination, filename), nil
	}
	return destination, nil
}
