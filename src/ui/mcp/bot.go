package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	domainBot "github.com/dresar/gowanew/domains/bot"
	"github.com/dresar/gowanew/usecase"
	mcpg "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var botSchema = `{
  "type": "object",
  "required": ["action"],
  "properties": {
    "action": {
      "type": "string",
      "enum": ["list_rules", "create_rule", "delete_rule", "toggle_rule", "query_logs"],
      "description": "Action: list_rules, create_rule, delete_rule, toggle_rule, query_logs"
    },
    "device_id": {"type": "string", "description": "Act as this device instead of the connection default"},
    "trigger_value": {"type": "string", "description": "Keyword or regex trigger for the bot"},
    "response_content": {"type": "string", "description": "Text response sent when rule triggers"},
    "rule_id": {"type": "integer", "description": "Rule ID for delete or toggle"}
  }
}`

type BotHandler struct {
	botService usecase.IBotUsecase
	resolver   deviceResolver
}

func InitMcpBot(botService usecase.IBotUsecase, resolver deviceResolver) *BotHandler {
	return &BotHandler{botService: botService, resolver: resolver}
}

func (h *BotHandler) AddBotTools(mcpServer *server.MCPServer) {
	if h.botService == nil {
		return
	}

	tool := mcpg.NewTool("whatsapp_bot",
		mcpg.WithDescription("Control WhatsApp Bot automations: manage auto-reply rules and view bot activity event logs."),
		mcpg.WithTitleAnnotation("Bot & Auto-Reply Automation"),
		mcpg.WithReadOnlyHintAnnotation(false),
		mcpg.WithDestructiveHintAnnotation(true),
		mcpg.WithIdempotentHintAnnotation(false),
		mcpg.WithRawInputSchema(json.RawMessage(botSchema)),
	)
	tool.InputSchema = mcpg.ToolInputSchema{}
	mcpServer.AddTool(tool, h.handleBot)
}

func (h *BotHandler) handleBot(ctx context.Context, request mcpg.CallToolRequest) (*mcpg.CallToolResult, error) {
	if h.botService == nil {
		return mcpg.NewToolResultError("bot service is not available"), nil
	}

	action, err := request.RequireString("action")
	if err != nil {
		return mcpg.NewToolResultError(err.Error()), nil
	}

	switch action {
	case "list_rules":
		rules, err := h.botService.ListRules(ctx, domainBot.RuleFilter{})
		if err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		return mcpg.NewToolResultStructured(rules, fmt.Sprintf("found %d rules", len(rules))), nil

	case "create_rule":
		triggerVal, err := request.RequireString("trigger_value")
		if err != nil {
			return mcpg.NewToolResultError("trigger_value is required"), nil
		}
		responseContent, err := request.RequireString("response_content")
		if err != nil {
			return mcpg.NewToolResultError("response_content is required"), nil
		}

		isActive := true
		req := domainBot.CreateRuleRequest{
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    triggerVal,
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: responseContent,
			IsActive:        &isActive,
		}
		rule, err := h.botService.CreateRule(ctx, req)
		if err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		return mcpg.NewToolResultStructured(rule, fmt.Sprintf("created rule ID=%d", rule.ID)), nil

	case "delete_rule":
		id := int64(request.GetInt("rule_id", 0))
		if id == 0 {
			return mcpg.NewToolResultError("valid rule_id is required"), nil
		}
		if err := h.botService.DeleteRule(ctx, id); err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		return mcpg.NewToolResultText(fmt.Sprintf("deleted rule ID=%d", id)), nil

	case "toggle_rule":
		id := int64(request.GetInt("rule_id", 0))
		if id == 0 {
			return mcpg.NewToolResultError("valid rule_id is required"), nil
		}
		rule, err := h.botService.ToggleRuleActive(ctx, id)
		if err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		return mcpg.NewToolResultStructured(rule, fmt.Sprintf("toggled rule ID=%d active=%t", rule.ID, rule.IsActive)), nil

	case "query_logs":
		limit := request.GetInt("limit", 20)
		logs, total, err := h.botService.ListEventLogs(ctx, domainBot.EventLogFilter{Limit: limit})
		if err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		res := map[string]any{"total": total, "logs": logs}
		return mcpg.NewToolResultStructured(res, fmt.Sprintf("retrieved %d logs (total %d)", len(logs), total)), nil

	default:
		return mcpg.NewToolResultError(fmt.Sprintf("unknown action: %s", action)), nil
	}
}
