// Skill server — MCP stdio server for skillpatch
//
// Tools:
//   browse_skills(category?)  — list skills by category; omit for full index
//   find_skill(id)            — fetch and return full skill content
//
// Build for all platforms:
//   scripts/build.sh
//
// Binary lives at <plugin_root>/mcp/skill_server_<os>_<arch>[.exe]
// Shell wrapper at <plugin_root>/mcp/skill_server picks the correct binary.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	protocolVersion = "2024-11-05"
	serverName      = "skillpatch"
	serverVersion   = "0.1.0"
	topPerCategory  = 10
	httpTimeout     = 5 * time.Second
)

// ── types ─────────────────────────────────────────────────────────────────────

type Skill struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Summary    string   `json:"summary"`
	Category   string   `json:"category"`
	Tags       []string `json:"tags"`
	Keywords   []string `json:"keywords"`
	Verified   bool     `json:"verified"`
	TrustScore float64  `json:"trust_score"`
	UsageCount int      `json:"usage_count"`
	ContentURL string   `json:"content_url"`
	Version    string   `json:"version"`
	Featured   bool     `json:"featured"`
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolResult struct {
	Content []toolContent `json:"content"`
}

// ── paths ─────────────────────────────────────────────────────────────────────

// pluginRoot returns the plugin root directory.
// Binary lives at <plugin_root>/mcp/skill_server_*
func pluginRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(exe))
}

// ── index ─────────────────────────────────────────────────────────────────────

func loadIndex(root string) []Skill {
	data, err := os.ReadFile(filepath.Join(root, "local_index.json"))
	if err != nil {
		return nil
	}
	var skills []Skill
	json.Unmarshal(data, &skills)
	return skills
}

// ── content fetch ─────────────────────────────────────────────────────────────

func fetchContent(s Skill, root string) string {
	cacheDir := filepath.Join(root, "cache")
	cacheFile := filepath.Join(cacheDir, s.ID+".md")

	if data, err := os.ReadFile(cacheFile); err == nil {
		return string(data)
	}

	if s.ContentURL != "" {
		client := &http.Client{Timeout: httpTimeout}
		resp, err := client.Get(s.ContentURL)
		if err == nil {
			defer resp.Body.Close()
			if body, err := io.ReadAll(resp.Body); err == nil {
				os.MkdirAll(cacheDir, 0755)
				os.WriteFile(cacheFile, body, 0644)
				return string(body)
			}
		}
	}

	return fmt.Sprintf("# %s\n\n%s", s.Name, s.Summary)
}

// ── category helpers ──────────────────────────────────────────────────────────

var categoryDisplayNames = map[string]string{
	"productivity": "Productivity",
	"documents":    "Documents & Office",
	"data":         "Data & Analysis",
	"development":  "Development",
}

var categoryOrder = []string{"productivity", "documents", "data", "development"}

func displayName(category string) string {
	if name, ok := categoryDisplayNames[category]; ok {
		return name
	}
	if category == "" {
		return "Uncategorized"
	}
	return strings.ToUpper(category[:1]) + category[1:]
}

// ── tool: browse_skills ───────────────────────────────────────────────────────

func browseSkills(skills []Skill, category string) string {
	var sb strings.Builder

	if category == "" {
		var featured []Skill
		for _, s := range skills {
			if s.Featured {
				featured = append(featured, s)
			}
		}
		sort.Slice(featured, func(i, j int) bool {
			return featured[i].TrustScore > featured[j].TrustScore
		})
		if len(featured) > 0 {
			sb.WriteString("## Featured\n")
			for _, s := range featured {
				fmt.Fprintf(&sb, "- `%s` ★%.1f — %s\n", s.ID, s.TrustScore, s.Summary)
			}
			sb.WriteString("\n")
		}

		byCategory := map[string][]Skill{}
		for _, s := range skills {
			byCategory[s.Category] = append(byCategory[s.Category], s)
		}

		for _, cat := range categoryOrder {
			catSkills, ok := byCategory[cat]
			if !ok {
				continue
			}
			sort.Slice(catSkills, func(i, j int) bool {
				return catSkills[i].TrustScore > catSkills[j].TrustScore
			})
			total := len(catSkills)
			shown := catSkills
			if len(shown) > topPerCategory {
				shown = shown[:topPerCategory]
			}
			fmt.Fprintf(&sb, "## %s\n", displayName(cat))
			for _, s := range shown {
				fmt.Fprintf(&sb, "- `%s` ★%.1f — %s\n", s.ID, s.TrustScore, s.Summary)
			}
			if total > topPerCategory {
				fmt.Fprintf(&sb, "... %d more — call browse_skills(\"%s\") to see all\n", total-topPerCategory, cat)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("Call `find_skill(id)` to load full guidance for any skill.")
	} else {
		var catSkills []Skill
		for _, s := range skills {
			if s.Category == category {
				catSkills = append(catSkills, s)
			}
		}
		if len(catSkills) == 0 {
			return fmt.Sprintf("No skills found in category %q. Available: %s",
				category, strings.Join(categoryOrder, ", "))
		}
		sort.Slice(catSkills, func(i, j int) bool {
			return catSkills[i].TrustScore > catSkills[j].TrustScore
		})
		fmt.Fprintf(&sb, "## %s (%d skills)\n", displayName(category), len(catSkills))
		for _, s := range catSkills {
			fmt.Fprintf(&sb, "- `%s` ★%.1f — %s\n", s.ID, s.TrustScore, s.Summary)
		}
		sb.WriteString("\nCall `find_skill(id)` to load full guidance.")
	}

	return sb.String()
}

// ── tool: find_skill ──────────────────────────────────────────────────────────

func findSkill(skills []Skill, id, root string) string {
	for _, s := range skills {
		if s.ID == id {
			return fetchContent(s, root)
		}
	}
	return fmt.Sprintf("Skill %q not found. Call browse_skills() to see available skills.", id)
}

// ── MCP protocol ──────────────────────────────────────────────────────────────

func toolsList() interface{} {
	return map[string]interface{}{
		"tools": []interface{}{
			map[string]interface{}{
				"name":        "browse_skills",
				"description": "Browse the skillpatch library. Call with no arguments to see all categories with top skills. Pass a category name to see the full listing for that category.",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Category to drill into: productivity, documents, data, development (optional)",
						},
					},
				},
			},
			map[string]interface{}{
				"name":        "find_skill",
				"description": "Load the full content and guidance for a skill by ID.",
				"inputSchema": map[string]interface{}{
					"type":     "object",
					"required": []string{"id"},
					"properties": map[string]interface{}{
						"id": map[string]interface{}{
							"type":        "string",
							"description": "Skill ID from the index, e.g. pptx, csv-insight-kit, meeting-notes-polisher",
						},
					},
				},
			},
		},
	}
}

func handleToolCall(skills []Skill, root string, params json.RawMessage) (interface{}, *rpcError) {
	var p struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, &rpcError{Code: -32602, Message: "invalid params"}
	}

	var text string
	switch p.Name {
	case "browse_skills":
		category := ""
		if v, ok := p.Arguments["category"]; ok {
			category, _ = v.(string)
		}
		text = browseSkills(skills, category)
	case "find_skill":
		id, _ := p.Arguments["id"].(string)
		if id == "" {
			return nil, &rpcError{Code: -32602, Message: "id is required"}
		}
		text = findSkill(skills, id, root)
	default:
		return nil, &rpcError{Code: -32601, Message: fmt.Sprintf("unknown tool: %s", p.Name)}
	}

	return toolResult{Content: []toolContent{{Type: "text", Text: text}}}, nil
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	root := pluginRoot()
	skills := loadIndex(root)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	enc := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		// notifications have no id — process but don't respond
		if req.ID == nil {
			continue
		}

		var result interface{}
		var rpcErr *rpcError

		switch req.Method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
				"serverInfo":      map[string]interface{}{"name": serverName, "version": serverVersion},
			}
		case "tools/list":
			result = toolsList()
		case "tools/call":
			result, rpcErr = handleToolCall(skills, root, req.Params)
		default:
			rpcErr = &rpcError{Code: -32601, Message: fmt.Sprintf("method not found: %s", req.Method)}
		}

		resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			resp.Result = result
		}
		enc.Encode(resp)
	}
}
