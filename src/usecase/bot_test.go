package usecase

import (
	"context"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	domainBot "github.com/dresar/gowanew/domains/bot"
	botInfrastructure "github.com/dresar/gowanew/infrastructure/bot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func setupTestBotDB(t *testing.T) domainBot.IBotRepository {
	db, err := botInfrastructure.OpenBotDB(":memory:", 1)
	require.NoError(t, err)
	repo := botInfrastructure.NewSQLiteRepository(db)
	err = repo.InitializeSchema(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = repo.Close()
	})
	return repo
}

func TestBotService_RuleCRUDAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := setupTestBotDB(t)
	svc := NewBotService(repo)

	_, err := svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     "unknown",
		TriggerValue:    "val",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp",
	})
	assert.Error(t, err)
	assert.Equal(t, domainBot.ErrInvalidTriggerType, err)

	_, err = svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp",
	})
	assert.Error(t, err)
	assert.Equal(t, domainBot.ErrMissingTriggerValue, err)

	_, err = svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerRegex,
		TriggerValue:    "[invalid",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp",
	})
	assert.Error(t, err)

	_, err = svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello",
		Scope:           "invalid_scope",
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "resp",
	})
	assert.Error(t, err)
	assert.Equal(t, domainBot.ErrInvalidScope, err)

	_, err = svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello",
		Scope:           domainBot.ScopeAll,
		ResponseType:    "invalid_type",
		ResponseContent: "resp",
	})
	assert.Error(t, err)
	assert.Equal(t, domainBot.ErrInvalidResponseType, err)

	_, err = svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "",
	})
	assert.Error(t, err)
	assert.Equal(t, domainBot.ErrMissingResponseContent, err)

	created, err := svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "World!",
	})
	require.NoError(t, err)
	assert.True(t, created.ID > 0)
	assert.Equal(t, "hello", created.TriggerValue)
	assert.True(t, created.IsActive)

	fetched, err := svc.GetRuleByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, "World!", fetched.ResponseContent)

	updatedContent := "Universe!"
	updated, err := svc.UpdateRule(ctx, created.ID, domainBot.UpdateRuleRequest{
		ResponseContent: &updatedContent,
	})
	require.NoError(t, err)
	assert.Equal(t, "Universe!", updated.ResponseContent)

	toggled, err := svc.ToggleRuleActive(ctx, created.ID)
	require.NoError(t, err)
	assert.False(t, toggled.IsActive)

	toggledBack, err := svc.ToggleRuleActive(ctx, created.ID)
	require.NoError(t, err)
	assert.True(t, toggledBack.IsActive)

	rules, err := svc.ListRules(ctx, domainBot.RuleFilter{Search: "hello"})
	require.NoError(t, err)
	assert.Len(t, rules, 1)

	err = svc.DeleteRule(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetRuleByID(ctx, created.ID)
	assert.Error(t, err)
}

func TestBotService_GroupRuleCRUDAndValidation(t *testing.T) {
	ctx := context.Background()
	repo := setupTestBotDB(t)
	svc := NewBotService(repo)

	_, err := svc.UpsertGroupRule(ctx, domainBot.UpsertGroupRuleRequest{
		GroupJID: "",
	})
	assert.Error(t, err)
	assert.Equal(t, domainBot.ErrMissingGroupJID, err)

	antiLink := true
	welcome := true
	welcomeTpl := "Welcome {name} to {group}"
	farewell := true
	farewellTpl := "Goodbye {name} from {group}"

	saved, err := svc.UpsertGroupRule(ctx, domainBot.UpsertGroupRuleRequest{
		GroupJID:         "120363001@g.us",
		AntiLinkEnabled:  &antiLink,
		WelcomeEnabled:   &welcome,
		WelcomeTemplate:  &welcomeTpl,
		FarewellEnabled:  &farewell,
		FarewellTemplate: &farewellTpl,
	})
	require.NoError(t, err)
	assert.Equal(t, "120363001@g.us", saved.GroupJID)
	assert.True(t, saved.AntiLinkEnabled)
	assert.True(t, saved.WelcomeEnabled)
	assert.Equal(t, welcomeTpl, saved.WelcomeTemplate)
	assert.True(t, saved.FarewellEnabled)
	assert.Equal(t, farewellTpl, saved.FarewellTemplate)

	fetched, err := svc.GetGroupRule(ctx, "120363001@g.us")
	require.NoError(t, err)
	assert.Equal(t, welcomeTpl, fetched.WelcomeTemplate)

	all, err := svc.ListGroupRules(ctx)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	err = svc.DeleteGroupRule(ctx, "120363001@g.us")
	require.NoError(t, err)

	_, err = svc.GetGroupRule(ctx, "120363001@g.us")
	assert.Error(t, err)
}

func TestBotService_RuleMatchingLogic(t *testing.T) {
	rules := []*domainBot.Rule{
		{
			ID:              1,
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "Ping",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Pong",
			IsActive:        true,
		},
		{
			ID:              2,
			TriggerType:     domainBot.TriggerContains,
			TriggerValue:    "price",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Pricing list: $10/mo",
			IsActive:        true,
		},
		{
			ID:              3,
			TriggerType:     domainBot.TriggerStartsWith,
			TriggerValue:    "!menu",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Available commands: 1, 2, 3",
			IsActive:        true,
		},
		{
			ID:              4,
			TriggerType:     domainBot.TriggerRegex,
			TriggerValue:    `(?i)^order-[0-9]{4}$`,
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Order found",
			IsActive:        true,
		},
		{
			ID:              5,
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "private_secret",
			Scope:           domainBot.ScopePrivate,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Top Secret",
			IsActive:        true,
		},
		{
			ID:              6,
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "group_announcement",
			Scope:           domainBot.ScopeGroup,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Attention Group",
			IsActive:        true,
		},
		{
			ID:              7,
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "disabled_test",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Should not match",
			IsActive:        false,
		},
		{
			ID:              8,
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    "🎉 promo2026",
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Discount 50%",
			IsActive:        true,
		},
		{
			ID:              9,
			TriggerType:     domainBot.TriggerRegex,
			TriggerValue:    `[unclosed`,
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "Broken Regex",
			IsActive:        true,
		},
	}

	svc := NewBotService(nil)

	m := svc.MatchRule(rules, "ping", false)
	require.NotNil(t, m)
	assert.Equal(t, int64(1), m.ID)

	m = svc.MatchRule(rules, "  PING  ", true)
	require.NotNil(t, m)
	assert.Equal(t, int64(1), m.ID)

	m = svc.MatchRule(rules, "ping me", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "what is the price of this?", false)
	require.NotNil(t, m)
	assert.Equal(t, int64(2), m.ID)

	m = svc.MatchRule(rules, "!menu help", false)
	require.NotNil(t, m)
	assert.Equal(t, int64(3), m.ID)

	m = svc.MatchRule(rules, "help !menu", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "ORDER-5678", false)
	require.NotNil(t, m)
	assert.Equal(t, int64(4), m.ID)

	m = svc.MatchRule(rules, "ORDER-abc", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "private_secret", true)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "private_secret", false)
	require.NotNil(t, m)
	assert.Equal(t, int64(5), m.ID)

	m = svc.MatchRule(rules, "group_announcement", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "group_announcement", true)
	require.NotNil(t, m)
	assert.Equal(t, int64(6), m.ID)

	m = svc.MatchRule(rules, "disabled_test", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "🎉 promo2026", false)
	require.NotNil(t, m)
	assert.Equal(t, int64(8), m.ID)

	m = svc.MatchRule(rules, "unclosed", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "", false)
	assert.Nil(t, m)

	m = svc.MatchRule(rules, "   ", false)
	assert.Nil(t, m)
}

func TestBotService_GroupModerationAntiLink(t *testing.T) {
	ctx := context.Background()
	repo := setupTestBotDB(t)
	svc := NewBotService(repo)

	antiLinkOn := true
	_, err := svc.UpsertGroupRule(ctx, domainBot.UpsertGroupRuleRequest{
		GroupJID:        "modgroup@g.us",
		AntiLinkEnabled: &antiLinkOn,
	})
	require.NoError(t, err)

	antiLinkOff := false
	_, err = svc.UpsertGroupRule(ctx, domainBot.UpsertGroupRuleRequest{
		GroupJID:        "opengroup@g.us",
		AntiLinkEnabled: &antiLinkOff,
	})
	require.NoError(t, err)

	res, violated := svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "Join my room: https://chat.whatsapp.com/Room123")
	assert.True(t, violated)
	require.NotNil(t, res)
	assert.True(t, res.ShouldRevoke)
	assert.Equal(t, "anti_link_violation", res.Reason)

	res, violated = svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "Chat wa.me/628123456789")
	assert.True(t, violated)
	require.NotNil(t, res)
	assert.True(t, res.ShouldRevoke)

	res, violated = svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "Visit https://google.com for info")
	assert.True(t, violated)
	require.NotNil(t, res)
	assert.True(t, res.ShouldRevoke)

	res, violated = svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "Visit http://insecure.test")
	assert.True(t, violated)
	require.NotNil(t, res)
	assert.True(t, res.ShouldRevoke)

	res, violated = svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "Multiple links https://a.com and wa.me/111")
	assert.True(t, violated)
	require.NotNil(t, res)

	_, violated = svc.ModerateMessage(ctx, "opengroup@g.us", "user@s.whatsapp.net", "https://chat.whatsapp.com/Room123")
	assert.False(t, violated)

	_, violated = svc.ModerateMessage(ctx, "private@s.whatsapp.net", "user@s.whatsapp.net", "https://chat.whatsapp.com/Room123")
	assert.False(t, violated)

	_, violated = svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "")
	assert.False(t, violated)

	_, violated = svc.ModerateMessage(ctx, "unconfigured@g.us", "user@s.whatsapp.net", "https://example.com")
	assert.False(t, violated)

	_, violated = svc.ModerateMessage(ctx, "modgroup@g.us", "user@s.whatsapp.net", "Just normal chat without links")
	assert.False(t, violated)
}

func TestBotService_WelcomeFarewellTemplates(t *testing.T) {
	svc := NewBotService(nil)

	tplWelcome := "Welcome {name} to {group}! Enjoy your stay."
	formatted := svc.FormatWelcome(tplWelcome, "Alice", "Dev Group")
	assert.Equal(t, "Welcome Alice to Dev Group! Enjoy your stay.", formatted)

	tplFarewell := "Goodbye {name}, thanks for contributing to {group}!"
	formatted = svc.FormatFarewell(tplFarewell, "Bob", "Dev Group")
	assert.Equal(t, "Goodbye Bob, thanks for contributing to Dev Group!", formatted)

	tplNoPlaceholders := "Hello everyone!"
	assert.Equal(t, "Hello everyone!", svc.FormatWelcome(tplNoPlaceholders, "Charlie", "Group"))

	tplMultiple := "{name} welcome, {name}! ({group} / {group})"
	assert.Equal(t, "Dave welcome, Dave! (Room / Room)", svc.FormatWelcome(tplMultiple, "Dave", "Room"))

	tplSpecial := "Hi {name} in {group} 🎉"
	assert.Equal(t, "Hi <Admin> in <VIP & Test> 🎉", svc.FormatWelcome(tplSpecial, "<Admin>", "<VIP & Test>"))
}

func TestBotService_TextExtraction(t *testing.T) {
	svc := NewBotService(nil)

	assert.Equal(t, "", svc.ExtractIncomingText(nil))
	assert.Equal(t, "", svc.ExtractIncomingText(&events.Message{}))

	evtConv := &events.Message{
		Message: &waE2E.Message{
			Conversation: proto.String("Hello conversation"),
		},
	}
	assert.Equal(t, "Hello conversation", svc.ExtractIncomingText(evtConv))

	evtExt := &events.Message{
		Message: &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: proto.String("Hello extended text"),
			},
		},
	}
	assert.Equal(t, "Hello extended text", svc.ExtractIncomingText(evtExt))

	evtImg := &events.Message{
		Message: &waE2E.Message{
			ImageMessage: &waE2E.ImageMessage{
				Caption: proto.String("Image caption"),
			},
		},
	}
	assert.Equal(t, "Image caption", svc.ExtractIncomingText(evtImg))

	evtVid := &events.Message{
		Message: &waE2E.Message{
			VideoMessage: &waE2E.VideoMessage{
				Caption: proto.String("Video caption"),
			},
		},
	}
	assert.Equal(t, "Video caption", svc.ExtractIncomingText(evtVid))

	evtDoc := &events.Message{
		Message: &waE2E.Message{
			DocumentMessage: &waE2E.DocumentMessage{
				Caption: proto.String("Doc caption"),
			},
		},
	}
	assert.Equal(t, "Doc caption", svc.ExtractIncomingText(evtDoc))

	evtEdited := &events.Message{
		Message: &waE2E.Message{
			ProtocolMessage: &waE2E.ProtocolMessage{
				Type: waE2E.ProtocolMessage_MESSAGE_EDIT.Enum(),
				EditedMessage: &waE2E.Message{
					Conversation: proto.String("Edited text"),
				},
			},
		},
	}
	assert.Equal(t, "Edited text", svc.ExtractIncomingText(evtEdited))

	evtEphemeral := &events.Message{
		Message: &waE2E.Message{
			EphemeralMessage: &waE2E.FutureProofMessage{
				Message: &waE2E.Message{
					Conversation: proto.String("Ephemeral text"),
				},
			},
		},
	}
	assert.Equal(t, "Ephemeral text", svc.ExtractIncomingText(evtEphemeral))
}

func TestBotService_LoopPrevention(t *testing.T) {
	svc := NewBotService(nil)

	assert.True(t, svc.ShouldIgnoreMessage(nil, nil))

	evtFromMe := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				IsFromMe: true,
				Chat:     types.NewJID("user1", types.DefaultUserServer),
			},
		},
	}
	assert.True(t, svc.ShouldIgnoreMessage(evtFromMe, nil))

	evtBroadcastChat := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: types.NewJID("status", types.BroadcastServer),
			},
		},
	}
	assert.True(t, svc.ShouldIgnoreMessage(evtBroadcastChat, nil))

	evtStatusJID := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: types.NewJID("status@broadcast", types.BroadcastServer),
			},
		},
	}
	assert.True(t, svc.ShouldIgnoreMessage(evtStatusJID, nil))

	evtNormal := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				IsFromMe: false,
				Chat:     types.NewJID("1234567890", types.DefaultUserServer),
				Sender:   types.NewJID("1234567890", types.DefaultUserServer),
			},
		},
	}
	assert.False(t, svc.ShouldIgnoreMessage(evtNormal, nil))
}

func TestBotService_EventLogging(t *testing.T) {
	ctx := context.Background()
	repo := setupTestBotDB(t)
	svc := NewBotService(repo)

	_, err := svc.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
		EventType: "invalid",
		Status:    domainBot.LogStatusSuccess,
		SenderJID: "user@s.whatsapp.net",
	})
	assert.Error(t, err)

	_, err = svc.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
		EventType: domainBot.EventTypeAutoReply,
		Status:    "invalid_status",
		SenderJID: "user@s.whatsapp.net",
	})
	assert.Error(t, err)

	_, err = svc.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
		EventType: domainBot.EventTypeAutoReply,
		Status:    domainBot.LogStatusSuccess,
		SenderJID: "",
	})
	assert.Error(t, err)

	ruleID := int64(42)
	created, err := svc.CreateEventLog(ctx, domainBot.CreateEventLogDTO{
		EventType:       domainBot.EventTypeAutoReply,
		RuleID:          &ruleID,
		SenderJID:       "123@s.whatsapp.net",
		GroupJID:        "group@g.us",
		IncomingMessage: "ping",
		ResponseMessage: "pong",
		LatencyMS:       12,
		Status:          domainBot.LogStatusSuccess,
	})
	require.NoError(t, err)
	assert.True(t, created.ID > 0)
	assert.Equal(t, int64(12), created.LatencyMS)

	eventTypeFilter := domainBot.EventTypeAutoReply
	logs, count, err := svc.ListEventLogs(ctx, domainBot.EventLogFilter{
		EventType: &eventTypeFilter,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Len(t, logs, 1)
	assert.Equal(t, "ping", logs[0].IncomingMessage)

	err = svc.ClearEventLogs(ctx)
	require.NoError(t, err)

	logs, count, err = svc.ListEventLogs(ctx, domainBot.EventLogFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	assert.Len(t, logs, 0)
}

func TestBotService_HandleMessagePipeline(t *testing.T) {
	ctx := context.Background()
	repo := setupTestBotDB(t)
	svc := NewBotService(repo)

	created, err := svc.CreateRule(ctx, domainBot.CreateRuleRequest{
		TriggerType:     domainBot.TriggerExact,
		TriggerValue:    "hello bot",
		Scope:           domainBot.ScopeAll,
		ResponseType:    domainBot.ResponseTypeText,
		ResponseContent: "hello human",
	})
	require.NoError(t, err)

	evt := &events.Message{
		Info: types.MessageInfo{
			ID: "MSG-100",
			MessageSource: types.MessageSource{
				IsFromMe: false,
				Chat:     types.NewJID("customer", types.DefaultUserServer),
				Sender:   types.NewJID("customer", types.DefaultUserServer),
			},
		},
		Message: &waE2E.Message{
			Conversation: proto.String("hello bot"),
		},
	}

	handled, err := svc.HandleMessage(ctx, evt, nil, nil)
	require.NoError(t, err)
	assert.True(t, handled)

	logs, count, err := svc.ListEventLogs(ctx, domainBot.EventLogFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
	assert.Equal(t, domainBot.EventTypeAutoReply, logs[0].EventType)
	assert.Equal(t, &created.ID, logs[0].RuleID)
	assert.Equal(t, "hello human", logs[0].ResponseMessage)

	evtNoMatch := &events.Message{
		Info: types.MessageInfo{
			ID: "MSG-101",
			MessageSource: types.MessageSource{
				IsFromMe: false,
				Chat:     types.NewJID("customer", types.DefaultUserServer),
				Sender:   types.NewJID("customer", types.DefaultUserServer),
			},
		},
		Message: &waE2E.Message{
			Conversation: proto.String("random message"),
		},
	}
	handled, err = svc.HandleMessage(ctx, evtNoMatch, nil, nil)
	require.NoError(t, err)
	assert.False(t, handled)

	antiLinkTrue := true
	_, err = svc.UpsertGroupRule(ctx, domainBot.UpsertGroupRuleRequest{
		GroupJID:        "room@g.us",
		AntiLinkEnabled: &antiLinkTrue,
	})
	require.NoError(t, err)

	evtSpam := &events.Message{
		Info: types.MessageInfo{
			ID: "MSG-SPAM",
			MessageSource: types.MessageSource{
				IsFromMe: false,
				Chat:     types.NewJID("room", types.GroupServer),
				Sender:   types.NewJID("spammer", types.DefaultUserServer),
			},
		},
		Message: &waE2E.Message{
			Conversation: proto.String("Join wa.me/1234567890"),
		},
	}

	handled, err = svc.HandleMessage(ctx, evtSpam, nil, nil)
	require.NoError(t, err)
	assert.True(t, handled)

	logs, count, err = svc.ListEventLogs(ctx, domainBot.EventLogFilter{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
	assert.Equal(t, domainBot.EventTypeGroupModeration, logs[0].EventType)
}

func TestBotService_HandleGroupInfoPipeline(t *testing.T) {
	ctx := context.Background()
	repo := setupTestBotDB(t)
	svc := NewBotService(repo)

	welcomeTrue := true
	welcomeTpl := "Welcome @{name} to {group}"
	farewellTrue := true
	farewellTpl := "Bye @{name} from {group}"

	_, err := svc.UpsertGroupRule(ctx, domainBot.UpsertGroupRuleRequest{
		GroupJID:         "community@g.us",
		WelcomeEnabled:   &welcomeTrue,
		WelcomeTemplate:  &welcomeTpl,
		FarewellEnabled:  &farewellTrue,
		FarewellTemplate: &farewellTpl,
	})
	require.NoError(t, err)

	svc.HandleGroupInfo(ctx, nil, nil)

	rule, err := svc.GetGroupRule(ctx, "community@g.us")
	require.NoError(t, err)
	assert.True(t, rule.WelcomeEnabled)
	assert.True(t, rule.FarewellEnabled)

	res := svc.FormatWelcome(rule.WelcomeTemplate, "NewUser", "community")
	assert.Equal(t, "Welcome @NewUser to community", res)

	resBye := svc.FormatFarewell(rule.FarewellTemplate, "OldUser", "community")
	assert.Equal(t, "Bye @OldUser from community", resBye)
}

func TestBot_NokomenCompliance(t *testing.T) {
	fset := token.NewFileSet()
	files := []string{"bot.go", "bot_test.go"}
	var violations []string

	for _, file := range files {
		node, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		require.NoError(t, err, "failed to parse %s", file)

		for _, cg := range node.Comments {
			for _, comment := range cg.List {
				text := strings.TrimSpace(comment.Text)
				if strings.HasPrefix(text, "//go:build") {
					continue
				}
				violations = append(violations, fset.Position(comment.Pos()).String()+": "+text)
			}
		}
	}

	assert.Empty(t, violations, "nokomen violations found:\n%s", strings.Join(violations, "\n"))
}
