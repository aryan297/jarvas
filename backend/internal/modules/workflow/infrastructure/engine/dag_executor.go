package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"

	"github.com/jarvas/backend/internal/modules/workflow/application/port"
	"github.com/jarvas/backend/internal/modules/workflow/domain/entity"
	toolsvc "github.com/jarvas/backend/internal/modules/tool/application/service"
)

// DAGExecutor runs a WorkflowDefinition as a directed acyclic graph.
// Nodes execute in topological order; outputs are passed as template variables.
type DAGExecutor struct {
	runRepo   port.RunRepository
	registry  *toolsvc.Registry
	oai       *openai.Client
	model     string
}

func NewDAGExecutor(
	runRepo port.RunRepository,
	registry *toolsvc.Registry,
	openAIKey, model string,
) *DAGExecutor {
	return &DAGExecutor{
		runRepo:  runRepo,
		registry: registry,
		oai:      openai.NewClient(openAIKey),
		model:    model,
	}
}

// Execute runs the workflow. Called in a goroutine — updates run record directly.
func (e *DAGExecutor) Execute(ctx context.Context, run *entity.WorkflowRun, def entity.WorkflowDefinition) error {
	now := time.Now().UTC()
	run.Status = entity.RunRunning
	run.StartedAt = &now
	_ = e.runRepo.Update(ctx, run)

	// Build execution context from trigger payload.
	execCtx := make(map[string]interface{})
	for k, v := range run.TriggerPayload {
		execCtx[k] = v
	}

	// Topological sort.
	order, err := topologicalSort(def.Nodes, def.Edges)
	if err != nil {
		return e.failRun(ctx, run, fmt.Sprintf("topological sort failed: %v", err))
	}

	// Execute each node.
	for _, nodeID := range order {
		node := findNode(def.Nodes, nodeID)
		if node == nil {
			continue
		}

		output, nodeErr := e.executeNode(ctx, node, execCtx)
		if nodeErr != nil {
			return e.failRun(ctx, run, fmt.Sprintf("node %s (%s) failed: %v", node.ID, node.Type, nodeErr))
		}
		execCtx[node.ID+"_output"] = output
	}

	// Finish run.
	completed := time.Now().UTC()
	run.Status = entity.RunCompleted
	run.CompletedAt = &completed
	run.Result = execCtx
	_ = e.runRepo.Update(ctx, run)
	return nil
}

// ── Node execution ────────────────────────────────────────────────────────────

func (e *DAGExecutor) executeNode(ctx context.Context, node *entity.WorkflowNode, execCtx map[string]interface{}) (interface{}, error) {
	switch strings.ToLower(node.Type) {
	case "agent":
		return e.runAgentNode(ctx, node, execCtx)
	case "tool":
		return e.runToolNode(ctx, node, execCtx)
	case "condition":
		return e.runConditionNode(node, execCtx)
	case "delay":
		return e.runDelayNode(node)
	default:
		return nil, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

func (e *DAGExecutor) runAgentNode(ctx context.Context, node *entity.WorkflowNode, execCtx map[string]interface{}) (string, error) {
	prompt := getString(node.Config, "prompt")
	if prompt == "" {
		prompt = "Perform the next step in the workflow."
	}
	prompt = substituteVars(prompt, execCtx)

	model := getString(node.Config, "model")
	if model == "" {
		model = e.model
	}

	resp, err := e.oai.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:     model,
		MaxTokens: 1024,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleUser, Content: prompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("agent node LLM call: %w", err)
	}
	return resp.Choices[0].Message.Content, nil
}

func (e *DAGExecutor) runToolNode(ctx context.Context, node *entity.WorkflowNode, execCtx map[string]interface{}) (string, error) {
	toolName := getString(node.Config, "tool")
	if toolName == "" {
		return "", fmt.Errorf("tool node missing 'tool' field")
	}

	// Build args JSON from config, substituting template variables.
	argsMap := make(map[string]interface{})
	for k, v := range node.Config {
		if k == "tool" {
			continue
		}
		if s, ok := v.(string); ok {
			argsMap[k] = substituteVars(s, execCtx)
		} else {
			argsMap[k] = v
		}
	}

	argsJSON := mapToJSON(argsMap)
	result, err := e.registry.Execute(toolName, argsJSON)
	if err != nil {
		return "", fmt.Errorf("tool %s: %w", toolName, err)
	}
	return result, nil
}

func (e *DAGExecutor) runConditionNode(node *entity.WorkflowNode, execCtx map[string]interface{}) (string, error) {
	expr := substituteVars(getString(node.Config, "if"), execCtx)
	// Simple evaluation: non-empty string that isn't "false"/"0" is truthy.
	truthy := expr != "" && expr != "false" && expr != "0" && expr != "<nil>"
	if truthy {
		return getString(node.Config, "then"), nil
	}
	return getString(node.Config, "else"), nil
}

func (e *DAGExecutor) runDelayNode(node *entity.WorkflowNode) (string, error) {
	secs := 1
	if v, ok := node.Config["seconds"]; ok {
		if f, ok := v.(float64); ok {
			secs = int(f)
		}
	}
	if secs > 300 {
		secs = 300 // cap at 5 minutes
	}
	time.Sleep(time.Duration(secs) * time.Second)
	return fmt.Sprintf("delayed %ds", secs), nil
}

func (e *DAGExecutor) failRun(ctx context.Context, run *entity.WorkflowRun, msg string) error {
	now := time.Now().UTC()
	run.Status = entity.RunFailed
	run.ErrorMsg = msg
	run.CompletedAt = &now
	_ = e.runRepo.Update(ctx, run)
	return fmt.Errorf(msg)
}

// ── Graph helpers ──────────────────────────────────────────────────────────────

// topologicalSort returns node IDs in execution order using Kahn's algorithm.
func topologicalSort(nodes []entity.WorkflowNode, edges []entity.WorkflowEdge) ([]string, error) {
	// Build in-degree map and adjacency list.
	inDegree := make(map[string]int)
	adj := make(map[string][]string)

	for _, n := range nodes {
		inDegree[n.ID] = 0
	}
	for _, e := range edges {
		if e.From == "START" {
			continue
		}
		adj[e.From] = append(adj[e.From], e.To)
		inDegree[e.To]++
	}

	// Find start nodes (in-degree 0, or reachable from START).
	var queue []string
	for _, e := range edges {
		if e.From == "START" {
			queue = append(queue, e.To)
			inDegree[e.To] = 0 // reset — START edge counts as external
		}
	}
	if len(queue) == 0 {
		// No START edges: enqueue all zero-in-degree nodes.
		for id, deg := range inDegree {
			if deg == 0 {
				queue = append(queue, id)
			}
		}
	}

	var order []string
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		order = append(order, cur)
		for _, next := range adj[cur] {
			if next == "END" {
				continue
			}
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}
	return order, nil
}

func findNode(nodes []entity.WorkflowNode, id string) *entity.WorkflowNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}

// ── Template substitution ─────────────────────────────────────────────────────

// substituteVars replaces {key} placeholders with values from execCtx.
func substituteVars(s string, ctx map[string]interface{}) string {
	for k, v := range ctx {
		s = strings.ReplaceAll(s, "{"+k+"}", fmt.Sprintf("%v", v))
	}
	return s
}

// ── JSON helpers ──────────────────────────────────────────────────────────────

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func mapToJSON(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(m))
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%q:%q", k, fmt.Sprintf("%v", v)))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
