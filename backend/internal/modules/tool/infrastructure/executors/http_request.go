package executors

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	toolsvc "github.com/jarvas/backend/internal/modules/tool/application/service"
)

var httpReqClient = &http.Client{Timeout: 15 * time.Second}

// HTTPRequestDef returns a generic HTTP request tool.
// Useful for calling external APIs from workflow nodes.
func HTTPRequestDef() *toolsvc.ToolDef {
	return &toolsvc.ToolDef{
		Info: &schema.ToolInfo{
			Name: "http_request",
			Desc: "Make an HTTP request to any URL. Use for calling external APIs or webhooks.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"url": {
					Type:     schema.String,
					Desc:     "The full URL to request",
					Required: true,
				},
				"method": {
					Type:     schema.String,
					Desc:     "HTTP method: GET, POST, PUT, DELETE",
					Required: false,
					Enum:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
				},
				"body": {
					Type:     schema.String,
					Desc:     "Request body (for POST/PUT)",
					Required: false,
				},
				"headers": {
					Type:     schema.String,
					Desc:     `JSON object of headers e.g. {"Authorization":"Bearer token"}`,
					Required: false,
				},
			}),
		},
		Execute: doHTTPRequest,
	}
}

func doHTTPRequest(argsJSON string) (string, error) {
	var args struct {
		URL     string `json:"url"`
		Method  string `json:"method"`
		Body    string `json:"body"`
		Headers string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("http_request: invalid args: %w", err)
	}
	if args.URL == "" {
		return "", fmt.Errorf("http_request: url is required")
	}

	method := strings.ToUpper(args.Method)
	if method == "" {
		method = "GET"
	}

	var bodyReader io.Reader
	if args.Body != "" {
		bodyReader = strings.NewReader(args.Body)
	}

	req, err := http.NewRequest(method, args.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("http_request: build request: %w", err)
	}

	// Parse and apply headers.
	if args.Headers != "" {
		var hdrs map[string]string
		if err := json.Unmarshal([]byte(args.Headers), &hdrs); err == nil {
			for k, v := range hdrs {
				req.Header.Set(k, v)
			}
		}
	}
	if args.Body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpReqClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192)) // cap at 8 KB
	return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body)), nil
}
