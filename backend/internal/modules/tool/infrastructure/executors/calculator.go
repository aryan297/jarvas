package executors

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
	toolsvc "github.com/jarvas/backend/internal/modules/tool/application/service"
)

// CalculatorDef returns a safe arithmetic calculator tool.
// Accepts: {"a": number, "b": number, "op": "add|subtract|multiply|divide|modulo"}
func CalculatorDef() *toolsvc.ToolDef {
	return &toolsvc.ToolDef{
		Info: &schema.ToolInfo{
			Name: "calculator",
			Desc: "Perform basic arithmetic: add, subtract, multiply, divide, modulo. Use for numeric calculations.",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"a": {
					Type:     schema.Number,
					Desc:     "First number",
					Required: true,
				},
				"b": {
					Type:     schema.Number,
					Desc:     "Second number",
					Required: true,
				},
				"op": {
					Type:     schema.String,
					Desc:     "Operation: add, subtract, multiply, divide, modulo",
					Required: true,
					Enum:     []string{"add", "subtract", "multiply", "divide", "modulo"},
				},
			}),
		},
		Execute: calculate,
	}
}

func calculate(argsJSON string) (string, error) {
	var args struct {
		A  json.Number `json:"a"`
		B  json.Number `json:"b"`
		Op string      `json:"op"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("calculator: invalid args: %w", err)
	}

	a, err := strconv.ParseFloat(args.A.String(), 64)
	if err != nil {
		return "", fmt.Errorf("calculator: invalid 'a': %w", err)
	}
	b, err := strconv.ParseFloat(args.B.String(), 64)
	if err != nil {
		return "", fmt.Errorf("calculator: invalid 'b': %w", err)
	}

	op := strings.ToLower(strings.TrimSpace(args.Op))
	var result float64
	switch op {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return "Error: division by zero", nil
		}
		result = a / b
	case "modulo":
		if b == 0 {
			return "Error: modulo by zero", nil
		}
		result = float64(int64(a) % int64(b))
	default:
		return "", fmt.Errorf("calculator: unknown operation %q", op)
	}

	// Format cleanly: drop trailing .000
	formatted := strconv.FormatFloat(result, 'f', -1, 64)
	return fmt.Sprintf("%s %s %s = %s", args.A, op, args.B, formatted), nil
}
