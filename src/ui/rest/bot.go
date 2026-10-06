package rest

import (
	"strconv"
	"strings"

	domainBot "github.com/dresar/gowanew/domains/bot"
	domainChatStorage "github.com/dresar/gowanew/domains/chatstorage"
	"github.com/dresar/gowanew/infrastructure/whatsapp"
	"github.com/dresar/gowanew/pkg/utils"
	"github.com/dresar/gowanew/usecase"
	"github.com/gofiber/fiber/v3"
	"go.mau.fi/whatsmeow"
)

type BotHandler struct {
	botUsecase      usecase.IBotUsecase
	dm              *whatsapp.DeviceManager
	chatStorageRepo domainChatStorage.IChatStorageRepository
}

func InitRestBot(app fiber.Router, botUsecase usecase.IBotUsecase, dm *whatsapp.DeviceManager, chatStorageRepo domainChatStorage.IChatStorageRepository) *BotHandler {
	h := &BotHandler{
		botUsecase:      botUsecase,
		dm:              dm,
		chatStorageRepo: chatStorageRepo,
	}

	app.Get("/bot/rules", h.ListRules)
	app.Post("/bot/rules", h.CreateRule)
	app.Post("/bot/rules/import", h.ImportRules)
	app.Post("/bot/rules/auto-tag-pacar", h.AutoTagPacar)
	app.Get("/bot/rules/:id", h.GetRule)
	app.Put("/bot/rules/:id", h.UpdateRule)
	app.Delete("/bot/rules/:id", h.DeleteRule)
	app.Patch("/bot/rules/:id/toggle", h.ToggleRule)

	app.Get("/bot/group-rules", h.ListGroupRules)
	app.Post("/bot/group-rules", h.UpsertGroupRule)
	app.Put("/bot/group-rules", h.UpsertGroupRule)
	app.Get("/bot/group-rules/:group_jid", h.GetGroupRule)
	app.Delete("/bot/group-rules/:group_jid", h.DeleteGroupRule)

	app.Get("/bot/ai/config", h.GetAIConfig)
	app.Put("/bot/ai/config", h.UpdateAIConfig)
	app.Post("/bot/ai/chat", h.ChatWithAI)

	app.Get("/bot/ai/tools", h.ListTools)
	app.Post("/bot/ai/tools", h.ExecuteTool)
	app.Post("/bot/ai/execute", h.ExecuteTool)

	app.Get("/bot/logs", h.ListLogs)
	app.Delete("/bot/logs", h.ClearLogs)

	return h
}

func (h *BotHandler) ListRules(c fiber.Ctx) error {
	var filter domainBot.RuleFilter
	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true" || activeStr == "1"
		filter.IsActive = &active
	}
	if scopeStr := c.Query("scope"); scopeStr != "" {
		s := domainBot.Scope(scopeStr)
		filter.Scope = &s
	}
	if recipient := c.Query("recipient_jid"); recipient != "" {
		filter.RecipientJID = &recipient
	}
	filter.Search = c.Query("search")
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}
	rules, err := h.botUsecase.ListRules(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rules fetched",
		Results: rules,
	})
}

func (h *BotHandler) CreateRule(c fiber.Ctx) error {
	var req domainBot.CreateRuleRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json body",
		})
	}
	if strings.TrimSpace(req.TriggerValue) == "" || strings.TrimSpace(req.ResponseContent) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "trigger_value and response_content are required",
		})
	}
	if req.Scope == "" {
		req.Scope = domainBot.ScopeAll
	}
	if req.ResponseType == "" {
		req.ResponseType = domainBot.ResponseTypeText
	}
	rule, err := h.botUsecase.CreateRule(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rule created",
		Results: rule,
	})
}

func (h *BotHandler) ImportRules(c fiber.Ctx) error {
	var reqs []domainBot.CreateRuleRequest
	if err := c.Bind().Body(&reqs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json array",
		})
	}
	if len(reqs) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "empty rules array",
		})
	}

	imported := 0
	for _, req := range reqs {
		if strings.TrimSpace(req.TriggerValue) == "" || strings.TrimSpace(req.ResponseContent) == "" {
			continue
		}
		if req.Scope == "" {
			req.Scope = domainBot.ScopeAll
		}
		if req.ResponseType == "" {
			req.ResponseType = domainBot.ResponseTypeText
		}
		if req.TriggerType == "" {
			req.TriggerType = domainBot.TriggerContains
		}
		_, err := h.botUsecase.CreateRule(c.Context(), req)
		if err == nil {
			imported++
		}
	}

	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rules imported",
		Results: map[string]any{
			"total":    len(reqs),
			"imported": imported,
		},
	})
}

func (h *BotHandler) GetRule(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid rule id",
		})
	}
	rule, err := h.botUsecase.GetRuleByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
			Code:    "NOT_FOUND",
			Message: "rule not found",
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rule retrieved",
		Results: rule,
	})
}

func (h *BotHandler) UpdateRule(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid rule id",
		})
	}
	var req domainBot.UpdateRuleRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json",
		})
	}
	rule, err := h.botUsecase.UpdateRule(c.Context(), id, req)
	if err != nil {
		if err == domainBot.ErrRuleNotFound {
			return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
				Code:    "NOT_FOUND",
				Message: "rule not found",
			})
		}
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rule updated",
		Results: rule,
	})
}

func (h *BotHandler) DeleteRule(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid rule id",
		})
	}
	if err := h.botUsecase.DeleteRule(c.Context(), id); err != nil {
		if err == domainBot.ErrRuleNotFound {
			return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
				Code:    "NOT_FOUND",
				Message: "rule not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rule deleted",
		Results: map[string]any{"id": id, "deleted": true},
	})
}

func (h *BotHandler) ToggleRule(c fiber.Ctx) error {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid rule id",
		})
	}
	rule, err := h.botUsecase.ToggleRuleActive(c.Context(), id)
	if err != nil {
		if err == domainBot.ErrRuleNotFound {
			return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
				Code:    "NOT_FOUND",
				Message: "rule not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "rule toggled",
		Results: rule,
	})
}

func (h *BotHandler) AutoTagPacar(c fiber.Ctx) error {
	count, err := h.botUsecase.AutoTagPacarRules(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "berhasil menandai balasan khusus pacar",
		Results: map[string]any{
			"updated": count,
		},
	})
}

func (h *BotHandler) ListGroupRules(c fiber.Ctx) error {
	rules, err := h.botUsecase.ListGroupRules(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "group rules fetched",
		Results: rules,
	})
}

func (h *BotHandler) UpsertGroupRule(c fiber.Ctx) error {
	var req domainBot.UpsertGroupRuleRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json",
		})
	}
	if strings.TrimSpace(req.GroupJID) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "group_jid is required",
		})
	}
	rule, err := h.botUsecase.UpsertGroupRule(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "group rule saved",
		Results: rule,
	})
}

func (h *BotHandler) GetGroupRule(c fiber.Ctx) error {
	groupJID := c.Params("group_jid")
	if groupJID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "group_jid is required",
		})
	}
	rule, err := h.botUsecase.GetGroupRule(c.Context(), groupJID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
			Code:    "NOT_FOUND",
			Message: "group rule not found",
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "group rule retrieved",
		Results: rule,
	})
}

func (h *BotHandler) DeleteGroupRule(c fiber.Ctx) error {
	groupJID := c.Params("group_jid")
	if groupJID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "group_jid is required",
		})
	}
	if err := h.botUsecase.DeleteGroupRule(c.Context(), groupJID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(utils.ResponseData{
			Code:    "NOT_FOUND",
			Message: "group rule not found",
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "group rule deleted",
		Results: map[string]any{"group_jid": groupJID, "deleted": true},
	})
}

func (h *BotHandler) GetAIConfig(c fiber.Ctx) error {
	cfg, err := h.botUsecase.GetAIConfig(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "ai config retrieved",
		Results: cfg,
	})
}

func (h *BotHandler) UpdateAIConfig(c fiber.Ctx) error {
	var req domainBot.UpdateAIConfigRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json",
		})
	}
	cfg, err := h.botUsecase.UpdateAIConfig(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "ai config updated",
		Results: cfg,
	})
}

func (h *BotHandler) ChatWithAI(c fiber.Ctx) error {
	var req domainBot.ChatRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json",
		})
	}
	if strings.TrimSpace(req.Message) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "message is required",
		})
	}
	res, err := h.botUsecase.ChatWithAI(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).JSON(utils.ResponseData{
			Code:    "AI_ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "ai response generated",
		Results: res,
	})
}

func (h *BotHandler) ListTools(c fiber.Ctx) error {
	tools, err := h.botUsecase.GetTools(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "tools list",
		Results: tools,
	})
}

func (h *BotHandler) ExecuteTool(c fiber.Ctx) error {
	var req domainBot.ToolRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "invalid json",
		})
	}
	if strings.TrimSpace(req.Tool) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "BAD_REQUEST",
			Message: "tool name is required",
		})
	}
	var cli *whatsmeow.Client
	if h.dm != nil {
		devID, _ := req.Parameters["device_id"].(string)
		if devID == "" {
			devID = strings.TrimSpace(c.Get("X-Device-Id"))
		}
		if devID == "" {
			devID = strings.TrimSpace(c.Query("device_id"))
		}
		if devID != "" {
			if dev, ok := h.dm.GetDevice(devID); ok && dev != nil {
				cli = dev.GetClient()
			}
		}
		if cli == nil {
			if def := h.dm.DefaultDevice(); def != nil {
				cli = def.GetClient()
			}
		}
	}
	res, err := h.botUsecase.ExecuteTool(c.Context(), cli, h.chatStorageRepo, req)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ResponseData{
			Code:    "TOOL_ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "tool executed",
		Results: res,
	})
}

func (h *BotHandler) ListLogs(c fiber.Ctx) error {
	var filter domainBot.EventLogFilter
	filter.Limit = 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			filter.Limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			filter.Offset = o
		}
	}
	if et := c.Query("event_type"); et != "" {
		eventType := domainBot.EventType(et)
		filter.EventType = &eventType
	}
	if st := c.Query("status"); st != "" {
		status := domainBot.LogStatus(st)
		filter.Status = &status
	}
	filter.GroupJID = c.Query("group_jid")
	filter.SenderJID = c.Query("sender_jid")
	filter.Search = c.Query("search")

	logs, total, err := h.botUsecase.ListEventLogs(c.Context(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "logs retrieved",
		Results: map[string]any{
			"total":  total,
			"limit":  filter.Limit,
			"offset": filter.Offset,
			"logs":   logs,
		},
	})
}

func (h *BotHandler) ClearLogs(c fiber.Ctx) error {
	if err := h.botUsecase.ClearEventLogs(c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(utils.ResponseData{
			Code:    "ERROR",
			Message: err.Error(),
		})
	}
	return c.JSON(utils.ResponseData{
		Code:    "SUCCESS",
		Message: "logs cleared",
		Results: map[string]any{"deleted_count": 0},
	})
}
