// capy-mcp is a Model Context Protocol server that exposes Capy as a tool
// to AI agents (Claude Desktop, Claude Code, any MCP client).
//
// It speaks JSON-RPC 2.0 over stdio, line-delimited per the MCP stdio
// transport. Tools advertised:
//
//   - capy_run     : transpile a script through an inline library (string)
//   - capy_run_file: transpile a script through a library file on disk
//   - capy_check   : validate a library (no script) — returns function/type names
//
// Wire it into Claude Desktop by adding to ~/Library/Application Support/
// Claude/claude_desktop_config.json (macOS) or the equivalent on your OS:
//
//   {
//     "mcpServers": {
//       "capy": { "command": "capy-mcp" }
//     }
//   }
//
// See docs/mcp.md for the full setup guide.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/olivierdevelops/capy"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "capy"
	serverVersion   = "0.3.0"
)

// --- JSON-RPC 2.0 envelopes -------------------------------------------------

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// --- MCP message shapes -----------------------------------------------------

type toolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type textContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []textContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// --- tools ------------------------------------------------------------------

var tools = []toolDef{
	{
		Name: "capy_run",
		Description: "Transpile a Capy source script through an inline library and return the generated output. " +
			"Use this when you have both a library definition and source text in memory — no disk needed.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"library": map[string]any{
					"type":        "string",
					"description": "Full contents of the library file (.capy syntax).",
				},
				"script": map[string]any{
					"type":        "string",
					"description": "The source code to transpile (Capy DSL declared by the library).",
				},
			},
			"required": []string{"library", "script"},
		},
	},
	{
		Name:        "capy_run_file",
		Description: "Transpile a script file through a .capy library file on disk and return the generated output.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"library_path": map[string]any{
					"type":        "string",
					"description": "Absolute or working-directory-relative path to the .capy library file.",
				},
				"script_path": map[string]any{
					"type":        "string",
					"description": "Path to the source script (typically .capy).",
				},
			},
			"required": []string{"library_path", "script_path"},
		},
	},
	{
		Name: "capy_check",
		Description: "Validate a Capy library (no script run). Returns the declared function names and type names " +
			"if it parses, or a precise error message if it doesn't. Use before calling capy_run to give the user " +
			"actionable feedback about library bugs.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"library": map[string]any{
					"type":        "string",
					"description": "Full contents of the .capy library file.",
				},
			},
			"required": []string{"library"},
		},
	},
}

// --- tool implementations ---------------------------------------------------

func toolCapyRun(args map[string]any) (string, error) {
	libSrc, _ := args["library"].(string)
	scriptSrc, _ := args["script"].(string)
	if libSrc == "" {
		return "", fmt.Errorf("library is required")
	}
	if scriptSrc == "" {
		return "", fmt.Errorf("script is required")
	}
	lib, err := capy.NewLibrary(libSrc)
	if err != nil {
		return "", fmt.Errorf("library: %v", err)
	}
	out, err := lib.Run(scriptSrc)
	if err != nil {
		return "", fmt.Errorf("transpile: %v", err)
	}
	return out, nil
}

func toolCapyRunFile(args map[string]any) (string, error) {
	libPath, _ := args["library_path"].(string)
	scriptPath, _ := args["script_path"].(string)
	if libPath == "" || scriptPath == "" {
		return "", fmt.Errorf("library_path and script_path are required")
	}
	lib, err := capy.NewLibraryFromFile(libPath)
	if err != nil {
		return "", fmt.Errorf("load %s: %v", libPath, err)
	}
	scriptBytes, err := os.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("read %s: %v", scriptPath, err)
	}
	out, err := lib.Run(string(scriptBytes))
	if err != nil {
		return "", fmt.Errorf("transpile: %v", err)
	}
	return out, nil
}

type checkResult struct {
	Valid     bool     `json:"valid"`
	Functions []string `json:"functions,omitempty"`
	Extension string   `json:"extension,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func toolCapyCheck(args map[string]any) (string, error) {
	libSrc, _ := args["library"].(string)
	if libSrc == "" {
		return "", fmt.Errorf("library is required")
	}
	lib, err := capy.NewLibrary(libSrc)
	if err != nil {
		res, _ := json.MarshalIndent(checkResult{Valid: false, Error: err.Error()}, "", "  ")
		return string(res), nil
	}
	res, _ := json.MarshalIndent(checkResult{
		Valid:     true,
		Functions: lib.FunctionNames(),
		Extension: lib.Extension(),
	}, "", "  ")
	return string(res), nil
}

func sniffCapy(src string) bool {
	for _, line := range strings.Split(src, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}
		// First non-comment line decides. Capy starts with bareword `function`,
		// `extension`, `type`, `output_file`, or `file_template:`. YAML almost
		// always has `key:` mapping syntax up front (e.g. `extension: py`).
		if strings.HasPrefix(s, "function ") || strings.HasPrefix(s, "type ") ||
			strings.HasPrefix(s, "extension ") || strings.HasPrefix(s, "output_file ") ||
			strings.HasPrefix(s, "file_template:") {
			return true
		}
		return false
	}
	return false
}

// --- dispatcher -------------------------------------------------------------

func handle(req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo": map[string]any{
				"name":    serverName,
				"version": serverVersion,
			},
		}
	case "notifications/initialized":
		// notification — no response needed; signal upstream by returning a
		// zero-id response that we'll drop.
		return rpcResponse{}
	case "tools/list":
		resp.Result = map[string]any{"tools": tools}
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: "invalid params: " + err.Error()}
			return resp
		}
		text, err := dispatchTool(params.Name, params.Arguments)
		if err != nil {
			resp.Result = toolResult{
				Content: []textContent{{Type: "text", Text: err.Error()}},
				IsError: true,
			}
		} else {
			resp.Result = toolResult{
				Content: []textContent{{Type: "text", Text: text}},
			}
		}
	case "ping":
		resp.Result = map[string]any{}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

func dispatchTool(name string, args map[string]any) (string, error) {
	switch name {
	case "capy_run":
		return toolCapyRun(args)
	case "capy_run_file":
		return toolCapyRunFile(args)
	case "capy_check":
		return toolCapyCheck(args)
	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

// --- stdio loop -------------------------------------------------------------

func main() {
	// Some clients send long single-line JSON; bump the scanner buffer well
	// past the 64 KiB default to handle big inline libraries.
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 8*1024*1024)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal(line, &req); err != nil {
			fmt.Fprintf(os.Stderr, "capy-mcp: parse error: %v\n", err)
			continue
		}
		resp := handle(req)
		// notifications (no id) and the synthetic empty response from
		// initialized notification produce no output.
		if len(req.ID) == 0 || (resp.JSONRPC == "" && resp.Result == nil && resp.Error == nil) {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "capy-mcp: encode error: %v\n", err)
			continue
		}
		out.Flush()
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "capy-mcp: stdin error: %v\n", err)
		os.Exit(1)
	}
}
