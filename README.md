# courses-mcp

MCP server for [FIT CTU Courses](https://courses.fit.cvut.cz) - lets Claude browse course pages, read lecture materials, and look up course info.

## Examples

> *What is the first lecture in NI-PDP about?*

Claude fetches the lecture PDF and tells you: it covers types of parallel computers (SMP, clusters, GPUs), parallel computation models (PRAM, APRAM), and metrics like speedup, efficiency, and Amdahl's/Gustafson's laws.

> *How many points do I need for credit in NI-SIB?*

Claude reads the classification page and tells you: at least 20 points from homework assignments (out of 50), plus passing the final exam with at least 10 points from the written part.

> *Download all lecture PDFs from NI-PDP to ~/Downloads/NI-PDP/*

Claude lists the lectures page, finds all PDF links, and downloads them one by one.

## Requirements

- Go 1.26+
- **Linux:** a keyring daemon must be running (`gnome-keyring`, `kwallet`, or any `secret-service`-compatible daemon)
- **macOS / Windows:** works out of the box (system keychain)

## Install

```sh
go install github.com/chladas0/courses-mcp/cmd/courses-mcp@latest
```

Make sure `~/go/bin` is in your `PATH`. If `courses-mcp` is not found after install, add this to your shell profile (`~/.zshrc`, `~/.bashrc`, etc.) and reload it:

```sh
export PATH="$HOME/go/bin:$PATH"
```

## Setup

Run this once to store your refresh token in the OS keychain:

```sh
courses-mcp --setup
```

Open [courses.fit.cvut.cz](https://courses.fit.cvut.cz) in your browser and log in. Then open **DevTools → Application → Cookies → courses.fit.cvut.cz**, copy the value of **`oauth_refresh_token`**, and paste it when prompted.

Setup automatically registers the server in `~/.claude.json`. Restart Claude Code to apply.

The refresh token lasts ~1 month. The server uses it to obtain short-lived access tokens (~2h) automatically - only re-run setup when the refresh token expires.

## Tools

| Tool | Description |
|------|-------------|
| `get_my_info` | Authenticated user's name and username |
| `get_my_courses` | Enrolled and teaching courses |
| `get_course_info` | Credits and completion type for a course |
| `get_course_page` | Course page as Markdown (homepage or subpage) |
| `get_file_content` | Read a lecture or assignment file as text (HTML or PDF) |
| `download_file` | Download a raw file to disk (PDF, ZIP, etc.) |
