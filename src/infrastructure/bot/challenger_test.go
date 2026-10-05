package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	domainBot "github.com/dresar/gowanew/domains/bot"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChallenger_ConcurrentStress_50Goroutines_SingleRepo(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "challenger_stress_single.db")
	db, err := OpenBotDB(dbPath, 10)
	require.NoError(t, err)
	defer db.Close()

	repo := NewSQLiteRepository(db)
	ctx := context.Background()
	require.NoError(t, repo.InitializeSchema(ctx))

	const numGoroutines = 50
	const iterationsPerGoroutine = 20

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*iterationsPerGoroutine*10)

	startTime := time.Now()

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for i := 0; i < iterationsPerGoroutine; i++ {
				trigVal := fmt.Sprintf("trigger_w%d_it%d", workerID, i)
				rule := &domainBot.Rule{
					TriggerType:     domainBot.TriggerExact,
					TriggerValue:    trigVal,
					Scope:           domainBot.ScopeAll,
					ResponseType:    domainBot.ResponseTypeText,
					ResponseContent: fmt.Sprintf("resp_content_%d_%d", workerID, i),
					IsActive:        true,
				}
				created, err := repo.CreateRule(ctx, rule)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d create rule failed: %w", workerID, i, err)
					continue
				}

				fetched, err := repo.GetRuleByID(ctx, created.ID)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d get rule failed: %w", workerID, i, err)
				} else if fetched.TriggerValue != trigVal {
					errCh <- fmt.Errorf("worker %d it %d trigger value mismatch: got %s, want %s", workerID, i, fetched.TriggerValue, trigVal)
				}

				newContent := fmt.Sprintf("updated_resp_%d_%d", workerID, i)
				_, err = repo.UpdateRule(ctx, created.ID, domainBot.UpdateRuleRequest{
					ResponseContent: &newContent,
				})
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d update rule failed: %w", workerID, i, err)
				}

				_, err = repo.ToggleRuleActive(ctx, created.ID)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d toggle 1 failed: %w", workerID, i, err)
				}
				_, err = repo.ToggleRuleActive(ctx, created.ID)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d toggle 2 failed: %w", workerID, i, err)
				}

				_, err = repo.ListRules(ctx, domainBot.RuleFilter{
					Limit: 10,
				})
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d list rules failed: %w", workerID, i, err)
				}

				groupJID := fmt.Sprintf("1203630000000%02d@g.us", workerID)
				grp := &domainBot.GroupRule{
					GroupJID:         groupJID,
					AntiLinkEnabled:  i%2 == 0,
					WelcomeEnabled:   true,
					WelcomeTemplate:  fmt.Sprintf("Welcome worker %d it %d", workerID, i),
					FarewellEnabled:  i%3 == 0,
					FarewellTemplate: fmt.Sprintf("Goodbye worker %d", workerID),
				}
				_, err = repo.UpsertGroupRule(ctx, grp)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d upsert group rule failed: %w", workerID, i, err)
				}

				_, err = repo.GetGroupRuleByGroupJID(ctx, groupJID)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d get group rule failed: %w", workerID, i, err)
				}

				_, err = repo.ListAllGroupRules(ctx)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d list all group rules failed: %w", workerID, i, err)
				}

				newModel := fmt.Sprintf("gpt-4o-w%d", workerID)
				newTemp := 0.5 + float64(i%10)*0.05
				_, err = repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{
					Model:       &newModel,
					Temperature: &newTemp,
				})
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d update ai config failed: %w", workerID, i, err)
				}

				_, err = repo.GetAIConfig(ctx)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d get ai config failed: %w", workerID, i, err)
				}

				log := &domainBot.EventLog{
					EventType:       domainBot.EventTypeAutoReply,
					RuleID:          &created.ID,
					SenderJID:       fmt.Sprintf("user_%d@s.whatsapp.net", workerID),
					GroupJID:        groupJID,
					IncomingMessage: fmt.Sprintf("in_%d_%d", workerID, i),
					ResponseMessage: fmt.Sprintf("out_%d_%d", workerID, i),
					LatencyMS:       int64(workerID*10 + i),
					Status:          domainBot.LogStatusSuccess,
				}
				_, err = repo.CreateEventLog(ctx, log)
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d create log failed: %w", workerID, i, err)
				}

				_, _, err = repo.ListEventLogs(ctx, domainBot.EventLogFilter{
					Limit: 5,
				})
				if err != nil {
					errCh <- fmt.Errorf("worker %d it %d list logs failed: %w", workerID, i, err)
				}

				if i%2 == 0 {
					err = repo.DeleteRule(ctx, created.ID)
					if err != nil {
						errCh <- fmt.Errorf("worker %d it %d delete rule failed: %w", workerID, i, err)
					}
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	elapsed := time.Since(startTime)
	t.Logf("Stress test completed in %v across 50 concurrent goroutines", elapsed)

	var errList []error
	for err := range errCh {
		errList = append(errList, err)
	}

	assert.Empty(t, errList, "found %d concurrent execution errors:\n%v", len(errList), errList)

	finalRules, err := repo.ListRules(ctx, domainBot.RuleFilter{Limit: 2000})
	require.NoError(t, err)
	expectedRemainingRules := (numGoroutines * iterationsPerGoroutine) / 2
	assert.Equal(t, expectedRemainingRules, len(finalRules))

	finalGroups, err := repo.ListAllGroupRules(ctx)
	require.NoError(t, err)
	assert.Equal(t, numGoroutines, len(finalGroups))

	_, totalLogs, err := repo.ListEventLogs(ctx, domainBot.EventLogFilter{Limit: 1})
	require.NoError(t, err)
	assert.Equal(t, int64(numGoroutines*iterationsPerGoroutine), totalLogs)
}

func TestChallenger_ConcurrentStress_50Goroutines_MultiRepoSharedFile(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "challenger_stress_multirepo.db")
	ctx := context.Background()

	initDB, err := OpenBotDB(dbPath, 5)
	require.NoError(t, err)
	initRepo := NewSQLiteRepository(initDB)
	require.NoError(t, initRepo.InitializeSchema(ctx))
	require.NoError(t, initDB.Close())

	const numRepos = 5
	const goroutinesPerRepo = 10
	totalGoroutines := numRepos * goroutinesPerRepo

	repos := make([]domainBot.IBotRepository, numRepos)
	dbs := make([]*sql.DB, numRepos)
	for r := 0; r < numRepos; r++ {
		dbInstance, err := OpenBotDB(dbPath, 5)
		require.NoError(t, err)
		dbs[r] = dbInstance
		repos[r] = NewSQLiteRepository(dbInstance)
	}

	defer func() {
		for _, db := range dbs {
			_ = db.Close()
		}
	}()

	var wg sync.WaitGroup
	errCh := make(chan error, totalGoroutines*20)

	for r := 0; r < numRepos; r++ {
		for g := 0; g < goroutinesPerRepo; g++ {
			wg.Add(1)
			go func(repoIndex, workerID int) {
				defer wg.Done()
				targetRepo := repos[repoIndex]

				for i := 0; i < 10; i++ {
					rule := &domainBot.Rule{
						TriggerType:     domainBot.TriggerContains,
						TriggerValue:    fmt.Sprintf("multi_r%d_w%d_it%d", repoIndex, workerID, i),
						Scope:           domainBot.ScopeGroup,
						ResponseType:    domainBot.ResponseTypeText,
						ResponseContent: "multi_test_response",
						IsActive:        true,
					}
					created, err := targetRepo.CreateRule(ctx, rule)
					if err != nil {
						errCh <- fmt.Errorf("multi repo %d worker %d create rule failed: %w", repoIndex, workerID, err)
						continue
					}

					_, err = targetRepo.GetRuleByID(ctx, created.ID)
					if err != nil {
						errCh <- fmt.Errorf("multi repo %d worker %d get rule failed: %w", repoIndex, workerID, err)
					}

					grp := &domainBot.GroupRule{
						GroupJID:        fmt.Sprintf("multi_grp_%d_%d@g.us", repoIndex, workerID),
						AntiLinkEnabled: true,
						WelcomeEnabled:  true,
					}
					_, err = targetRepo.UpsertGroupRule(ctx, grp)
					if err != nil {
						errCh <- fmt.Errorf("multi repo %d worker %d upsert group failed: %w", repoIndex, workerID, err)
					}

					log := &domainBot.EventLog{
						EventType:       domainBot.EventTypeAIChat,
						SenderJID:       fmt.Sprintf("multi_sender_%d_%d@s.whatsapp.net", repoIndex, workerID),
						IncomingMessage: "multi_in",
						ResponseMessage: "multi_out",
						LatencyMS:       12,
						Status:          domainBot.LogStatusSuccess,
					}
					_, err = targetRepo.CreateEventLog(ctx, log)
					if err != nil {
						errCh <- fmt.Errorf("multi repo %d worker %d create log failed: %w", repoIndex, workerID, err)
					}

					_, err = targetRepo.ListRules(ctx, domainBot.RuleFilter{Limit: 5})
					if err != nil {
						errCh <- fmt.Errorf("multi repo %d worker %d list rules failed: %w", repoIndex, workerID, err)
					}
				}
			}(r, g)
		}
	}

	wg.Wait()
	close(errCh)

	var errList []error
	for err := range errCh {
		errList = append(errList, err)
	}

	assert.Empty(t, errList, "multi-repo file lock failure under 50 goroutines: %v", errList)
}

func TestChallenger_ConcurrentStress_50Goroutines_InMemory(t *testing.T) {
	repo, _ := newTestInMemoryRepository(t)
	ctx := context.Background()

	const numGoroutines = 50
	const opsPerWorker = 15

	var wg sync.WaitGroup
	errCh := make(chan error, numGoroutines*opsPerWorker*5)

	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				rule := &domainBot.Rule{
					TriggerType:     domainBot.TriggerStartsWith,
					TriggerValue:    fmt.Sprintf("!inmem_w%d_i%d", workerID, i),
					Scope:           domainBot.ScopeAll,
					ResponseType:    domainBot.ResponseTypeText,
					ResponseContent: "inmem_pong",
					IsActive:        true,
				}
				created, err := repo.CreateRule(ctx, rule)
				if err != nil {
					errCh <- fmt.Errorf("inmem worker %d create rule failed: %w", workerID, err)
					continue
				}

				_, err = repo.GetRuleByID(ctx, created.ID)
				if err != nil {
					errCh <- fmt.Errorf("inmem worker %d get rule failed: %w", workerID, err)
				}

				_, err = repo.ListRules(ctx, domainBot.RuleFilter{Limit: 5})
				if err != nil {
					errCh <- fmt.Errorf("inmem worker %d list rules failed: %w", workerID, err)
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	var errList []error
	for err := range errCh {
		errList = append(errList, err)
	}
	assert.Empty(t, errList, "in-memory stress errors: %v", errList)
}

func TestChallenger_EdgeCase_UnicodeAndEmoji(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	vectors := []struct {
		name         string
		triggerVal   string
		responseVal  string
		searchSubstr string
	}{
		{
			name:         "Emojis Complex and Modifiers",
			triggerVal:   "👋 Hello 🔥 🚀 🤖 💯 🇯🇵 🇺🇸 👨‍👩‍👧‍👦 🏳️‍🌈",
			responseVal:  "Selamat datang! ❤️ 🎉 👍 💡",
			searchSubstr: "🚀 🤖",
		},
		{
			name:         "Arabic Script RTL",
			triggerVal:   "مرحبا بالعالم - رسالة تلقائية",
			responseVal:  "أهلاً وسهلاً بك في نظام الرد الآلي",
			searchSubstr: "بالعالم",
		},
		{
			name:         "Chinese Simplified and Traditional",
			triggerVal:   "你好世界，自動回覆測試規則！",
			responseVal:  "收到您的消息，这是简体中文与繁體字测试。",
			searchSubstr: "自動回覆",
		},
		{
			name:         "Japanese Hiragana Katakana Kanji",
			triggerVal:   "こんにちは世界！自動応答ボットテスト🌸",
			responseVal:  "メッセージを受信しました。ありがとうございます！✨",
			searchSubstr: "自動応答",
		},
		{
			name:         "Russian Cyrillic",
			triggerVal:   "Привет, мир! Правило автоответа.",
			responseVal:  "Сообщение успешно обработано ботом.",
			searchSubstr: "автоответа",
		},
		{
			name:         "Hindi Devanagari",
			triggerVal:   "नमस्ते दुनिया! स्वचालित उत्तर नियम।",
			responseVal:  "नमस्ते! आपका संदेश प्राप्त हुआ।",
			searchSubstr: "स्वचालित",
		},
		{
			name:         "Latin Accented and Diacritics",
			triggerVal:   "Café naïf façade résumé crème brûlée",
			responseVal:  "Touches d'accents correctement préservées",
			searchSubstr: "façade",
		},
		{
			name:         "Zero Width Characters and Invisible Symbols",
			triggerVal:   "Word\u200bWith\u200cZero\u200dWidth\u00a0Space",
			responseVal:  "Response\u200bWith\u200cHidden\u200dMarkers",
			searchSubstr: "Zero\u200dWidth",
		},
	}

	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			rule := &domainBot.Rule{
				TriggerType:     domainBot.TriggerExact,
				TriggerValue:    v.triggerVal,
				Scope:           domainBot.ScopeAll,
				ResponseType:    domainBot.ResponseTypeText,
				ResponseContent: v.responseVal,
				IsActive:        true,
			}
			created, err := repo.CreateRule(ctx, rule)
			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, v.triggerVal, created.TriggerValue)
			assert.Equal(t, v.responseVal, created.ResponseContent)

			fetched, err := repo.GetRuleByID(ctx, created.ID)
			require.NoError(t, err)
			assert.Equal(t, v.triggerVal, fetched.TriggerValue)
			assert.Equal(t, v.responseVal, fetched.ResponseContent)

			searchResults, err := repo.ListRules(ctx, domainBot.RuleFilter{
				Search: v.searchSubstr,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, searchResults, "search failed to find unicode string: %s", v.searchSubstr)
			found := false
			for _, r := range searchResults {
				if r.ID == created.ID {
					found = true
					break
				}
			}
			assert.True(t, found, "matching rule not present in search results")

			groupJID := fmt.Sprintf("grp_%x@g.us", []byte(v.triggerVal)[:8])
			grp := &domainBot.GroupRule{
				GroupJID:         groupJID,
				AntiLinkEnabled:  true,
				WelcomeEnabled:   true,
				WelcomeTemplate:  v.responseVal,
				FarewellEnabled:  true,
				FarewellTemplate: v.triggerVal,
			}
			savedGrp, err := repo.UpsertGroupRule(ctx, grp)
			require.NoError(t, err)
			assert.Equal(t, v.responseVal, savedGrp.WelcomeTemplate)
			assert.Equal(t, v.triggerVal, savedGrp.FarewellTemplate)

			fetchedGrp, err := repo.GetGroupRuleByGroupJID(ctx, groupJID)
			require.NoError(t, err)
			assert.Equal(t, v.responseVal, fetchedGrp.WelcomeTemplate)
			assert.Equal(t, v.triggerVal, fetchedGrp.FarewellTemplate)

			log := &domainBot.EventLog{
				EventType:       domainBot.EventTypeAutoReply,
				RuleID:          &created.ID,
				SenderJID:       "628123456789@s.whatsapp.net",
				GroupJID:        groupJID,
				IncomingMessage: v.triggerVal,
				ResponseMessage: v.responseVal,
				LatencyMS:       42,
				Status:          domainBot.LogStatusSuccess,
			}
			savedLog, err := repo.CreateEventLog(ctx, log)
			require.NoError(t, err)
			assert.Equal(t, v.triggerVal, savedLog.IncomingMessage)
			assert.Equal(t, v.responseVal, savedLog.ResponseMessage)
		})
	}
}

func TestChallenger_EdgeCase_SpecialRegexPatterns(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	regexPatterns := []struct {
		name    string
		pattern string
	}{
		{
			name:    "Complex Bot Command Regex",
			pattern: `^\[bot\]\s+([\w\.\-]+)\?*(foo|bar)$`,
		},
		{
			name:    "Escaped Digits and Non-Capturing Groups",
			pattern: `\d{3}-\d{4}\s+(?:\w+)`,
		},
		{
			name:    "WhatsApp Invite Link Regex",
			pattern: `^(?:https?:\/\/)?(?:www\.)?chat\.whatsapp\.com\/([0-9A-Za-z]{20,24})`,
		},
		{
			name:    "SQL Injection Pattern in Trigger",
			pattern: `'; DROP TABLE bot_rules; SELECT * FROM bot_rules WHERE '1'='1`,
		},
		{
			name:    "Unbalanced Quotes and Brackets",
			pattern: `'''"""[[[{{{(((\\\\////`,
		},
		{
			name:    "Unicode Regex Character Classes",
			pattern: `^[\p{L}\p{N}\s\-_.,!?'"()\[\]{}|\\\/@#$%^&*+=~` + "`" + `<>:;]+$`,
		},
		{
			name:    "Newlines and Whitespace Escapes",
			pattern: "line1\\r\\nline2\\tline3\nline4\rline5",
		},
	}

	for _, tc := range regexPatterns {
		t.Run(tc.name, func(t *testing.T) {
			rule := &domainBot.Rule{
				TriggerType:     domainBot.TriggerRegex,
				TriggerValue:    tc.pattern,
				Scope:           domainBot.ScopeAll,
				ResponseType:    domainBot.ResponseTypeText,
				ResponseContent: "regex_ok",
				IsActive:        true,
			}

			created, err := repo.CreateRule(ctx, rule)
			require.NoError(t, err)
			require.NotNil(t, created)
			assert.Equal(t, tc.pattern, created.TriggerValue)

			fetched, err := repo.GetRuleByID(ctx, created.ID)
			require.NoError(t, err)
			assert.Equal(t, tc.pattern, fetched.TriggerValue)

			newPattern := tc.pattern + "_updated"
			updated, err := repo.UpdateRule(ctx, created.ID, domainBot.UpdateRuleRequest{
				TriggerValue: &newPattern,
			})
			require.NoError(t, err)
			assert.Equal(t, newPattern, updated.TriggerValue)

			err = repo.DeleteRule(ctx, created.ID)
			require.NoError(t, err)
		})
	}

	rules, err := repo.ListRules(ctx, domainBot.RuleFilter{Limit: 10})
	require.NoError(t, err)
	assert.Empty(t, rules, "table was corrupted or contains orphaned rules")
}

func TestChallenger_EdgeCase_MaxBoundaryPayloads(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	sizes := []int{64 * 1024, 256 * 1024, 1024 * 1024}
	for _, sz := range sizes {
		t.Run(fmt.Sprintf("Payload_%d_Bytes", sz), func(t *testing.T) {
			largeContent := strings.Repeat("A", sz)
			rule := &domainBot.Rule{
				TriggerType:     domainBot.TriggerExact,
				TriggerValue:    fmt.Sprintf("large_trigger_%d", sz),
				Scope:           domainBot.ScopeAll,
				ResponseType:    domainBot.ResponseTypeText,
				ResponseContent: largeContent,
				IsActive:        true,
			}
			created, err := repo.CreateRule(ctx, rule)
			require.NoError(t, err)

			fetched, err := repo.GetRuleByID(ctx, created.ID)
			require.NoError(t, err)
			assert.Equal(t, sz, len(fetched.ResponseContent))
			assert.Equal(t, largeContent, fetched.ResponseContent)
		})
	}

	largePrompt := strings.Repeat("EnterpriseSystemPrompt", 20000)
	cfg, err := repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{
		SystemPrompt: &largePrompt,
	})
	require.NoError(t, err)
	assert.Equal(t, len(largePrompt), len(cfg.SystemPrompt))

	fetchedCfg, err := repo.GetAIConfig(ctx)
	require.NoError(t, err)
	assert.Equal(t, largePrompt, fetchedCfg.SystemPrompt)

	logSizes := 256 * 1024
	largeMsg := strings.Repeat("IncomingChatPayload", logSizes/19)
	log := &domainBot.EventLog{
		EventType:       domainBot.EventTypeAIChat,
		SenderJID:       "628999999999@s.whatsapp.net",
		GroupJID:        "120363000000000000@g.us",
		IncomingMessage: largeMsg,
		ResponseMessage: largeMsg,
		LatencyMS:       math.MaxInt64,
		Status:          domainBot.LogStatusSuccess,
	}
	savedLog, err := repo.CreateEventLog(ctx, log)
	require.NoError(t, err)
	assert.Equal(t, int64(math.MaxInt64), savedLog.LatencyMS)
	assert.Equal(t, len(largeMsg), len(savedLog.IncomingMessage))
	assert.Equal(t, len(largeMsg), len(savedLog.ResponseMessage))

	largeWelcome := strings.Repeat("WelcomeTemplateChunk", 5000)
	grp := &domainBot.GroupRule{
		GroupJID:        "120363999999999999@g.us",
		WelcomeEnabled:  true,
		WelcomeTemplate: largeWelcome,
	}
	savedGrp, err := repo.UpsertGroupRule(ctx, grp)
	require.NoError(t, err)
	assert.Equal(t, len(largeWelcome), len(savedGrp.WelcomeTemplate))

	fetchedGrp, err := repo.GetGroupRuleByGroupJID(ctx, "120363999999999999@g.us")
	require.NoError(t, err)
	assert.Equal(t, largeWelcome, fetchedGrp.WelcomeTemplate)
}

func TestChallenger_EdgeCase_NumericBoundariesAndPagination(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	zeroTemp := 0.0
	cfg, err := repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{
		Temperature: &zeroTemp,
	})
	require.NoError(t, err)
	assert.Equal(t, 0.0, cfg.Temperature)

	maxTemp := 2.0
	cfg2, err := repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{
		Temperature: &maxTemp,
	})
	require.NoError(t, err)
	assert.Equal(t, 2.0, cfg2.Temperature)

	extremeTemp := 99.9999
	cfg3, err := repo.UpdateAIConfig(ctx, domainBot.UpdateAIConfigRequest{
		Temperature: &extremeTemp,
	})
	require.NoError(t, err)
	assert.Equal(t, 99.9999, cfg3.Temperature)

	for i := 0; i < 5; i++ {
		_, err := repo.CreateRule(ctx, &domainBot.Rule{
			TriggerType:     domainBot.TriggerExact,
			TriggerValue:    fmt.Sprintf("boundary_rule_%d", i),
			Scope:           domainBot.ScopeAll,
			ResponseType:    domainBot.ResponseTypeText,
			ResponseContent: "resp",
			IsActive:        true,
		})
		require.NoError(t, err)
	}

	largeLimitRules, err := repo.ListRules(ctx, domainBot.RuleFilter{
		Limit:  1000000,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, len(largeLimitRules))

	outOfRangeRules, err := repo.ListRules(ctx, domainBot.RuleFilter{
		Limit:  10,
		Offset: 1000000,
	})
	require.NoError(t, err)
	assert.Empty(t, outOfRangeRules)

	zeroLimitRules, err := repo.ListRules(ctx, domainBot.RuleFilter{
		Limit:  0,
		Offset: 0,
	})
	require.NoError(t, err)
	assert.Equal(t, 5, len(zeroLimitRules))

	nonExistentLog := &domainBot.EventLog{
		EventType:       domainBot.EventTypeAutoReply,
		RuleID:          nil,
		SenderJID:       "sender@s.whatsapp.net",
		IncomingMessage: "msg",
		ResponseMessage: "resp",
		LatencyMS:       0,
		Status:          domainBot.LogStatusSuccess,
	}
	_, err = repo.CreateEventLog(ctx, nonExistentLog)
	require.NoError(t, err)

	dummyID := int64(999999999)
	withForeignID := &domainBot.EventLog{
		EventType:       domainBot.EventTypeAutoReply,
		RuleID:          &dummyID,
		SenderJID:       "sender@s.whatsapp.net",
		IncomingMessage: "msg",
		ResponseMessage: "resp",
		LatencyMS:       -1,
		Status:          domainBot.LogStatusFailed,
	}
	_, err = repo.CreateEventLog(ctx, withForeignID)
	require.NoError(t, err)
}

func TestChallenger_Adversarial_ConcurrentReadWriteDeleteRaces(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	var createdIDs sync.Map
	var idCounter int64

	const duration = 500 * time.Millisecond
	stopTime := time.Now().Add(duration)

	var wg sync.WaitGroup
	errCh := make(chan error, 500)

	for g := 0; g < 15; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				idx := atomic.AddInt64(&idCounter, 1)
				rule := &domainBot.Rule{
					TriggerType:     domainBot.TriggerExact,
					TriggerValue:    fmt.Sprintf("race_rule_%d_%d", workerID, idx),
					Scope:           domainBot.ScopeAll,
					ResponseType:    domainBot.ResponseTypeText,
					ResponseContent: "race_content",
					IsActive:        true,
				}
				created, err := repo.CreateRule(ctx, rule)
				if err != nil {
					errCh <- fmt.Errorf("create race err: %w", err)
					return
				}
				createdIDs.Store(created.ID, created.ID)
				time.Sleep(2 * time.Millisecond)
			}
		}(g)
	}

	for g := 0; g < 15; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				itemCount := 0
				createdIDs.Range(func(key, value any) bool {
					itemCount++
					targetID := key.(int64)
					newContent := fmt.Sprintf("race_update_%d", targetID)
					_, err := repo.UpdateRule(ctx, targetID, domainBot.UpdateRuleRequest{
						ResponseContent: &newContent,
					})
					if err != nil && !errors.Is(err, domainBot.ErrRuleNotFound) {
						errCh <- fmt.Errorf("update race unexpected err: %w", err)
					}

					_, err = repo.ToggleRuleActive(ctx, targetID)
					if err != nil && !errors.Is(err, domainBot.ErrRuleNotFound) {
						errCh <- fmt.Errorf("toggle race unexpected err: %w", err)
					}

					_, err = repo.GetRuleByID(ctx, targetID)
					if err != nil && !errors.Is(err, domainBot.ErrRuleNotFound) {
						errCh <- fmt.Errorf("get race unexpected err: %w", err)
					}

					return itemCount < 3
				})
				time.Sleep(5 * time.Millisecond)
			}
		}(g)
	}

	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				createdIDs.Range(func(key, value any) bool {
					targetID := key.(int64)
					err := repo.DeleteRule(ctx, targetID)
					if err != nil && !errors.Is(err, domainBot.ErrRuleNotFound) {
						errCh <- fmt.Errorf("delete race unexpected err: %w", err)
					}
					createdIDs.Delete(targetID)
					return false
				})
				time.Sleep(10 * time.Millisecond)
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	var errList []error
	for err := range errCh {
		errList = append(errList, err)
	}
	assert.Empty(t, errList, "unexpected race condition errors: %v", errList)
}

func TestChallenger_PurgeEventLogs_UnderConcurrentWrites(t *testing.T) {
	repo, _ := newTestRepository(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	errCh := make(chan error, 100)
	stopTime := time.Now().Add(500 * time.Millisecond)

	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				log := &domainBot.EventLog{
					EventType:       domainBot.EventTypeAutoReply,
					SenderJID:       fmt.Sprintf("sender_%d@s.whatsapp.net", workerID),
					IncomingMessage: "msg",
					ResponseMessage: "resp",
					Status:          domainBot.LogStatusSuccess,
				}
				_, err := repo.CreateEventLog(ctx, log)
				if err != nil {
					errCh <- fmt.Errorf("purge race create log failed: %w", err)
					return
				}
				time.Sleep(2 * time.Millisecond)
			}
		}(g)
	}

	for g := 0; g < 3; g++ {
		wg.Add(1)
		go func(cleanerID int) {
			defer wg.Done()
			for time.Now().Before(stopTime) {
				_, err := repo.PurgeEventLogs(ctx, nil)
				if err != nil {
					errCh <- fmt.Errorf("purge logs failed: %w", err)
					return
				}
				time.Sleep(20 * time.Millisecond)
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	var errList []error
	for err := range errCh {
		errList = append(errList, err)
	}
	assert.Empty(t, errList, "purge concurrent errors: %v", errList)
}
