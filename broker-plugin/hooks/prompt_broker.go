// Skill Broker — UserPromptSubmit hook
//
// Reads a prompt from stdin, searches the local skill index,
// and injects relevant skill content as additionalContext.
// Prompts never leave the machine — only skill IDs are sent to the network.
//
// Build for all platforms:
//   scripts/build_hook.sh
//
// Env overrides:
//   SKILL_BROKER_TIMEOUT   float seconds   (default 2.0)
//   SKILL_BROKER_TOP_K     int             (default 3)
//   SKILL_BROKER_THRESHOLD float           (default 2.0)
//   SKILL_BROKER_MIN_TERMS int             (default 2)
//   SKILL_BROKER_LOG_USAGE 1               (opt-in analytics, skill ID only)

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ── types ─────────────────────────────────────────────────────────────────────

type Skill struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Summary    string   `json:"summary"`
	Tags       []string `json:"tags"`
	Keywords   []string `json:"keywords"`
	Verified   bool     `json:"verified"`
	TrustScore float64  `json:"trust_score"`
	UsageCount int      `json:"usage_count"`
	ContentURL string   `json:"content_url"`
	Version    string   `json:"version"`
	Install    string   `json:"install"`

	// runtime only — not in JSON
	score         float64
	matchingTerms []string
}

type Config struct {
	RiskLevel              string  `json:"risk_level"`
	LastIndexUpdate        *string `json:"last_index_update"`
	IndexUpdateIntervalHrs int     `json:"index_update_interval_hours"`
}

type HookPayload struct {
	Prompt string `json:"prompt"`
}

type HookOutput struct {
	HookSpecificOutput *HookSpecificOutput `json:"hookSpecificOutput,omitempty"`
}

type HookSpecificOutput struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

// ── risk gates ────────────────────────────────────────────────────────────────

type gate struct {
	requireVerified bool
	minTrustScore   float64
}

var riskGates = map[string]gate{
	"strict":   {true, 4.0},
	"balanced": {false, 3.5},
	"open":     {false, 0.0},
}

// ── env config ────────────────────────────────────────────────────────────────

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

// ── paths ─────────────────────────────────────────────────────────────────────

// pluginRoot returns the plugin root directory.
// Binary lives at <plugin_root>/hooks/prompt_broker[.exe]
func pluginRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(filepath.Dir(exe))
}

// ── logging ───────────────────────────────────────────────────────────────────

var logWriter io.Writer = io.Discard

func initLog(root string) {
	path := filepath.Join(root, "hooks", "broker_debug.log")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		logWriter = f
	}
}

func logf(format string, args ...any) {
	fmt.Fprintf(logWriter, "[%s] %s\n",
		time.Now().UTC().Format(time.RFC3339),
		fmt.Sprintf(format, args...),
	)
}

// ── loading ───────────────────────────────────────────────────────────────────

func loadConfig(root string) Config {
	cfg := Config{RiskLevel: "balanced", IndexUpdateIntervalHrs: 24}
	data, err := os.ReadFile(filepath.Join(root, "config.json"))
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

func loadIndex(root string) []Skill {
	data, err := os.ReadFile(filepath.Join(root, "local_index.json"))
	if err != nil {
		return nil
	}
	var skills []Skill
	json.Unmarshal(data, &skills)
	return skills
}

// ── search ────────────────────────────────────────────────────────────────────

func searchLocal(prompt string, skills []Skill, topK int, scoreThresh float64, minTerms int) []Skill {
	rawTerms := strings.Fields(strings.ToLower(prompt))
	var queryTerms []string
	for _, t := range rawTerms {
		if len(t) > 2 {
			queryTerms = append(queryTerms, t)
		}
	}

	var scored []Skill
	for _, s := range skills {
		hay := strings.ToLower(strings.Join([]string{
			s.Name, s.Summary,
			strings.Join(s.Tags, " "),
			strings.Join(s.Keywords, " "),
		}, " "))

		var matching []string
		seen := map[string]bool{}
		for _, t := range queryTerms {
			if !seen[t] && strings.Contains(hay, t) {
				matching = append(matching, t)
				seen[t] = true
			}
		}
		if len(matching) < minTerms {
			continue
		}

		verBonus := 0.0
		if s.Verified {
			verBonus = 0.75
		}
		sc := float64(len(matching))*2.0 + math.Log1p(float64(s.UsageCount)) + verBonus
		if sc < scoreThresh {
			continue
		}
		s.score = sc
		s.matchingTerms = matching
		scored = append(scored, s)
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})
	if len(scored) > topK {
		scored = scored[:topK]
	}
	return scored
}

// ── risk gate ─────────────────────────────────────────────────────────────────

func passesRiskGate(s Skill, riskLevel string) bool {
	g, ok := riskGates[riskLevel]
	if !ok {
		g = riskGates["balanced"]
	}
	if g.requireVerified && !s.Verified {
		return false
	}
	return s.TrustScore >= g.minTrustScore
}

// ── content fetch ─────────────────────────────────────────────────────────────

func fetchContent(s Skill, root string, timeout time.Duration) string {
	cacheDir := filepath.Join(root, "cache")
	cacheFile := filepath.Join(cacheDir, s.ID+".md")

	if data, err := os.ReadFile(cacheFile); err == nil {
		return string(data)
	}

	if s.ContentURL != "" {
		client := &http.Client{Timeout: timeout}
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

	// fallback: minimal content from metadata
	return fmt.Sprintf("# %s\n\n%s", s.Name, s.Summary)
}

// ── context builder ───────────────────────────────────────────────────────────

func formatCount(n int) string {
	if n >= 1000 {
		return fmt.Sprintf("%d,%03d", n/1000, n%1000)
	}
	return fmt.Sprintf("%d", n)
}

func trustBadge(s Skill) string {
	var parts []string
	if s.Verified {
		parts = append(parts, "✓ Verified")
	}
	parts = append(parts, fmt.Sprintf("★ %.1f (%s users)", s.TrustScore, formatCount(s.UsageCount)))
	return strings.Join(parts, "  ")
}

func buildContext(s Skill, content, riskLevel string) string {
	actionHint := "If this skill is clearly relevant to the user's request, apply its guidance and mention it briefly at the end of your response. If it is not relevant, ignore this entirely."
	if riskLevel == "open" && !s.Verified {
		actionHint = "If this skill is clearly relevant, mention it briefly at the end of your response. Note that this skill is unverified."
	}
	return fmt.Sprintf(
		"The skill broker found a relevant skill for this task.\n\n"+
			"**%s** — %s\n\n"+
			"--- SKILL CONTENT ---\n%s\n--- END SKILL CONTENT ---\n\n"+
			"%s Attribution format: \"skill-broker: `%s` helped here.\"",
		s.Name, trustBadge(s),
		strings.TrimSpace(content),
		actionHint, s.ID,
	)
}

// ── opt-in analytics ──────────────────────────────────────────────────────────

func logUsage(skillID, registryURL string, timeout time.Duration) {
	if os.Getenv("SKILL_BROKER_LOG_USAGE") != "1" {
		return
	}
	payload, _ := json.Marshal(map[string]string{"event": "skill_inject", "skill_id": skillID})
	client := &http.Client{Timeout: timeout}
	client.Post(registryURL+"/events", "application/json",
		strings.NewReader(string(payload)))
}

// ── staleness check ───────────────────────────────────────────────────────────

func checkStaleness(cfg Config) {
	if cfg.LastIndexUpdate == nil {
		logf("INDEX: never updated — run /broker update to fetch latest")
		return
	}
	t, err := time.Parse(time.RFC3339, *cfg.LastIndexUpdate)
	if err != nil {
		return
	}
	interval := float64(cfg.IndexUpdateIntervalHrs)
	if interval == 0 {
		interval = 24
	}
	if time.Since(t).Hours() > interval {
		logf("INDEX: stale (%.0fh old) — run /broker update", time.Since(t).Hours())
	}
}

// ── main ──────────────────────────────────────────────────────────────────────

func main() {
	root := pluginRoot()
	initLog(root)
	logf("HOOK FIRED (go/%s/%s)", runtime.GOOS, runtime.GOARCH)

	timeout := time.Duration(envFloat("SKILL_BROKER_TIMEOUT", 2.0) * float64(time.Second))
	topK := envInt("SKILL_BROKER_TOP_K", 3)
	scoreThresh := envFloat("SKILL_BROKER_THRESHOLD", 2.0)
	minTerms := envInt("SKILL_BROKER_MIN_TERMS", 2)
	registryURL := os.Getenv("REGISTRY_URL")
	if registryURL == "" {
		registryURL = "http://127.0.0.1:8787"
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(strings.TrimSpace(string(data))) == 0 {
		fmt.Println("{}")
		return
	}

	var payload HookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		payload.Prompt = string(data)
	}

	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		fmt.Println("{}")
		return
	}
	logf("PROMPT: %.120s", prompt)

	cfg := loadConfig(root)
	index := loadIndex(root)
	checkStaleness(cfg)

	if len(index) == 0 {
		logf("INDEX: empty or missing")
		fmt.Println("{}")
		return
	}

	results := searchLocal(prompt, index, topK, scoreThresh, minTerms)
	logf("SEARCH: %d matches above threshold", len(results))

	var filtered []Skill
	for _, r := range results {
		if passesRiskGate(r, cfg.RiskLevel) {
			filtered = append(filtered, r)
		}
	}
	if len(filtered) == 0 {
		logf("RISK GATE: no skills passed (risk_level=%s)", cfg.RiskLevel)
		fmt.Println("{}")
		return
	}

	top := filtered[0]
	content := fetchContent(top, root, timeout)
	context := buildContext(top, content, cfg.RiskLevel)
	logf("INJECTING: %s (score=%.3f, risk=%s)", top.ID, top.score, cfg.RiskLevel)

	go logUsage(top.ID, registryURL, timeout)

	out, _ := json.Marshal(HookOutput{
		HookSpecificOutput: &HookSpecificOutput{
			HookEventName:     "UserPromptSubmit",
			AdditionalContext: context,
		},
	})
	fmt.Println(string(out))
}
