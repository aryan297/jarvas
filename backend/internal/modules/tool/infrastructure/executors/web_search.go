package executors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	toolsvc "github.com/jarvas/backend/internal/modules/tool/application/service"
)

var httpClient = &http.Client{Timeout: 8 * time.Second}

// WebSearchDef returns the tool definition for DuckDuckGo instant answers.
// No API key required.
func WebSearchDef() *toolsvc.ToolDef {
	return &toolsvc.ToolDef{
		Info: &schema.ToolInfo{
			Name: "web_search",
			Desc: "Search the web for current information. Use when you need facts you don't know or recent events.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"query": {
					Type:     schema.String,
					Desc:     "The search query",
					Required: true,
				},
			}),
		},
		Execute: webSearch,
	}
}

func webSearch(argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("web_search: invalid args: %w", err)
	}
	if args.Query == "" {
		return "", fmt.Errorf("web_search: query is required")
	}

	apiURL := fmt.Sprintf(
		"https://api.duckduckgo.com/?q=%s&format=json&no_redirect=1&no_html=1",
		url.QueryEscape(args.Query),
	)
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("web_search: http error: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var ddg struct {
		AbstractText    string `json:"AbstractText"`
		AbstractSource  string `json:"AbstractSource"`
		AbstractURL     string `json:"AbstractURL"`
		Answer          string `json:"Answer"`
		RelatedTopics   []struct {
			Text string `json:"Text"`
		} `json:"RelatedTopics"`
	}
	if err := json.Unmarshal(body, &ddg); err != nil {
		return fmt.Sprintf("Search completed for: %s. No structured results available.", args.Query), nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Search results for: %q\n\n", args.Query))

	if ddg.Answer != "" {
		sb.WriteString(fmt.Sprintf("Answer: %s\n\n", ddg.Answer))
	}
	if ddg.AbstractText != "" {
		sb.WriteString(fmt.Sprintf("Summary: %s\nSource: %s\n\n", ddg.AbstractText, ddg.AbstractSource))
	}
	if sb.Len() < 30 && len(ddg.RelatedTopics) > 0 {
		sb.WriteString("Related:\n")
		limit := 3
		if len(ddg.RelatedTopics) < limit {
			limit = len(ddg.RelatedTopics)
		}
		for _, t := range ddg.RelatedTopics[:limit] {
			if t.Text != "" {
				sb.WriteString("- ")
				sb.WriteString(t.Text)
				sb.WriteString("\n")
			}
		}
	}
	if sb.Len() < 30 {
		return fmt.Sprintf("No results found for: %s", args.Query), nil
	}
	return sb.String(), nil
}
