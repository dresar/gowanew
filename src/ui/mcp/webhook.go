package mcp

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dresar/gowanew/domains/chatstorage"
	domainDevice "github.com/dresar/gowanew/domains/device"
	mcpg "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var webhookSchema = `{
  "type": "object",
  "required": ["action"],
  "properties": {
    "action": {
      "type": "string",
      "enum": ["get", "set", "test"],
      "description": "Action: get (current config), set (update config), test (dispatch test event)"
    },
    "device_id": {"type": "string", "description": "Act as this device instead of the connection default"},
    "webhook_url": {"type": "string", "description": "Target HTTP/HTTPS webhook URL"},
    "webhook_secret": {"type": "string", "description": "HMAC-SHA256 secret key for signing payloads"},
    "webhook_events": {"type": "string", "description": "Comma-separated event names (e.g. message,message.ack,group.participants)"},
    "insecure_skip_verify": {"type": "boolean", "description": "Skip SSL/TLS certificate verification"},
    "event": {"type": "string", "description": "Event type for test (default: message)"}
  }
}`

type WebhookHandler struct {
	deviceService domainDevice.IDeviceUsecase
	resolver      deviceResolver
}

func InitMcpWebhook(deviceService domainDevice.IDeviceUsecase, resolver deviceResolver) *WebhookHandler {
	return &WebhookHandler{deviceService: deviceService, resolver: resolver}
}

func (h *WebhookHandler) AddWebhookTools(mcpServer *server.MCPServer) {
	if h.deviceService == nil {
		return
	}

	tool := mcpg.NewTool("whatsapp_webhook",
		mcpg.WithDescription("Inspect, configure, and test WhatsApp event webhooks for real-time integrations."),
		mcpg.WithTitleAnnotation("Webhook Management"),
		mcpg.WithReadOnlyHintAnnotation(false),
		mcpg.WithDestructiveHintAnnotation(true),
		mcpg.WithIdempotentHintAnnotation(false),
		mcpg.WithRawInputSchema(json.RawMessage(webhookSchema)),
	)
	tool.InputSchema = mcpg.ToolInputSchema{}
	mcpServer.AddTool(tool, h.handleWebhook)
}

func (h *WebhookHandler) handleWebhook(ctx context.Context, request mcpg.CallToolRequest) (*mcpg.CallToolResult, error) {
	ctx, inst, err := resolveDeviceContext(ctx, request, h.resolver)
	if err != nil {
		return mcpg.NewToolResultError(err.Error()), nil
	}
	deviceID := inst.ID()

	action, err := request.RequireString("action")
	if err != nil {
		return mcpg.NewToolResultError(err.Error()), nil
	}

	switch action {
	case "get":
		cfg, err := h.deviceService.GetDeviceWebhookConfig(ctx, deviceID)
		if err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		url := ""
		if cfg != nil && cfg.WebhookURL != nil {
			url = *cfg.WebhookURL
		}
		res := map[string]any{
			"device_id":                    deviceID,
			"webhook_url":                  url,
			"webhook_secret":               cfg.WebhookSecret,
			"webhook_events":               cfg.WebhookEvents,
			"webhook_insecure_skip_verify": cfg.WebhookInsecureSkipVerify,
		}
		return mcpg.NewToolResultStructured(res, fmt.Sprintf("webhook config for device %s", deviceID)), nil

	case "set":
		targetURL, err := request.RequireString("webhook_url")
		if err != nil {
			return mcpg.NewToolResultError("webhook_url is required"), nil
		}
		secret := request.GetString("webhook_secret", "")
		events := request.GetString("webhook_events", "message,message.ack")
		skipVerify := request.GetBool("insecure_skip_verify", false)

		cfg := &chatstorage.DeviceWebhookConfig{
			WebhookURL:                &targetURL,
			WebhookSecret:             secret,
			WebhookEvents:             events,
			WebhookInsecureSkipVerify: skipVerify,
		}
		if err := h.deviceService.SetDeviceWebhookConfig(ctx, deviceID, cfg); err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}
		return mcpg.NewToolResultText(fmt.Sprintf("updated webhook for %s -> %s", deviceID, targetURL)), nil

	case "test":
		targetURL := request.GetString("webhook_url", "")
		secret := request.GetString("webhook_secret", "")
		skipVerify := request.GetBool("insecure_skip_verify", false)
		event := request.GetString("event", "message")

		if targetURL == "" {
			cfg, _ := h.deviceService.GetDeviceWebhookConfig(ctx, deviceID)
			if cfg != nil && cfg.WebhookURL != nil {
				targetURL = *cfg.WebhookURL
				if secret == "" {
					secret = cfg.WebhookSecret
				}
				if !skipVerify {
					skipVerify = cfg.WebhookInsecureSkipVerify
				}
			}
		}

		if targetURL == "" {
			return mcpg.NewToolResultError("webhook_url is required"), nil
		}

		samplePayload := map[string]any{
			"event":     event,
			"timestamp": time.Now().Unix(),
			"device_id": deviceID,
			"test":      true,
			"payload": map[string]any{
				"message_id": "MCP_TEST_EVENT_12345",
				"sender_jid": "6281234567890@s.whatsapp.net",
				"chat_jid":   "6281234567890@s.whatsapp.net",
				"text":       "Pengujian webhook dari MCP Server.",
				"type":       "text",
			},
		}

		payloadBytes, _ := json.Marshal(samplePayload)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, strings.NewReader(string(payloadBytes)))
		if err != nil {
			return mcpg.NewToolResultError(err.Error()), nil
		}

		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("User-Agent", "GoWA-MCP-Tester/1.0")
		if secret != "" {
			mac := hmac.New(sha256.New, []byte(secret))
			mac.Write(payloadBytes)
			sig := hex.EncodeToString(mac.Sum(nil))
			httpReq.Header.Set("X-Hub-Signature-256", "sha256="+sig)
		}

		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: skipVerify},
			},
			Timeout: 10 * time.Second,
		}

		start := time.Now()
		resp, reqErr := client.Do(httpReq)
		latency := time.Since(start).Milliseconds()

		if reqErr != nil {
			res := map[string]any{
				"success":    false,
				"url":        targetURL,
				"latency_ms": latency,
				"error":      reqErr.Error(),
			}
			return mcpg.NewToolResultStructured(res, fmt.Sprintf("delivery failed: %v", reqErr)), nil
		}
		defer resp.Body.Close()

		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		res := map[string]any{
			"success":       resp.StatusCode >= 200 && resp.StatusCode < 300,
			"status_code":   resp.StatusCode,
			"latency_ms":    latency,
			"response_body": string(bodyBytes),
		}
		return mcpg.NewToolResultStructured(res, fmt.Sprintf("HTTP %d in %dms", resp.StatusCode, latency)), nil

	default:
		return mcpg.NewToolResultError(fmt.Sprintf("unknown action: %s", action)), nil
	}
}
