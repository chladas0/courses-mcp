// Package main is the entry point for the courses-mcp MCP server.
// It supports two modes: normal (MCP server over stdio) and setup (first-time
// token configuration via --setup flag).
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chladas0/courses-mcp/internal/auth"
	"github.com/chladas0/courses-mcp/internal/client"
	"github.com/chladas0/courses-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	setup := flag.Bool("setup", false, "Run first-time token setup")
	flag.Parse()

	ctx := context.Background()

	store := auth.NewTokenStore()
	provider := auth.NewProvider(store)

	if *setup {
		runSetup(ctx, store, provider)
		return
	}

	runServer(ctx, provider)
}

// runSetup guides the user through first-time refresh token configuration.
func runSetup(ctx context.Context, store *auth.TokenStore, provider *auth.Provider) {
	fmt.Print("courses-mcp setup\n\n")
	fmt.Println("Open https://courses.fit.cvut.cz in your browser and log in.")
	fmt.Println("Then open DevTools → Application → Cookies → courses.fit.cvut.cz")
	fmt.Println("Copy the value of \"oauth_refresh_token\" and paste it below.")
	fmt.Println()
	fmt.Print("Refresh token: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	refreshToken := strings.TrimSpace(scanner.Text())

	if refreshToken == "" {
		fmt.Fprintln(os.Stderr, "error: refresh token must not be empty")
		os.Exit(1)
	}

	if err := store.Save(auth.Token{RefreshToken: refreshToken}); err != nil {
		fmt.Fprintln(os.Stderr, "error saving token:", err)
		os.Exit(1)
	}

	if _, err := provider.Token(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "error validating token:", err)
		os.Exit(1)
	}

	fmt.Println("Setup complete.")
	if err := registerWithClaudeCode(); err != nil {
		fmt.Fprintln(os.Stderr, "warning: could not update ~/.claude.json:", err)
		fmt.Printf("Add manually:\n  {\"mcpServers\":{\"courses\":{\"command\":%q}}}\n", os.Args[0])
	}
}

// registerWithClaudeCode adds this binary to the mcpServers section of ~/.claude.json.
func registerWithClaudeCode() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find home dir: %w", err)
	}
	configPath := filepath.Join(home, ".claude.json")

	// Read existing config or start with empty object.
	var cfg map[string]json.RawMessage
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return fmt.Errorf("parse %s: %w", configPath, err)
		}
	} else {
		cfg = make(map[string]json.RawMessage)
	}

	// Parse or create mcpServers section.
	var servers map[string]json.RawMessage
	if raw, ok := cfg["mcpServers"]; ok {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return fmt.Errorf("parse mcpServers: %w", err)
		}
	} else {
		servers = make(map[string]json.RawMessage)
	}

	// Add or overwrite the courses entry.
	entry, err := json.Marshal(map[string]string{"command": os.Args[0]})
	if err != nil {
		return err
	}
	servers["courses"] = entry

	raw, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	cfg["mcpServers"] = raw

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	// Write to a temp file in the same directory, then rename for atomic replace.
	tmp, err := os.CreateTemp(filepath.Dir(configPath), ".claude.json.tmp*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if rename succeeded
	if _, err := tmp.Write(append(out, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := os.Rename(tmpName, configPath); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}

	fmt.Printf("Registered in %s - restart Claude Code to apply.\n", configPath)
	return nil
}

// runServer starts the MCP server over stdio.
func runServer(ctx context.Context, provider *auth.Provider) {
	apiClient := client.NewAPIClient(provider)
	pageClient := client.NewPageClient(provider)

	server := mcp.NewServer(&mcp.Implementation{Name: "courses-mcp", Version: "1.0.0"}, nil)

	tools.RegisterMyInfo(server, apiClient)
	tools.RegisterMyCourses(server, apiClient)
	tools.RegisterCourseInfo(server, apiClient)
	tools.RegisterCoursePage(server, pageClient)
	tools.RegisterFileContent(server, pageClient)
	tools.RegisterDownloadFile(server, pageClient)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, "server error:", err)
		os.Exit(1)
	}
}
