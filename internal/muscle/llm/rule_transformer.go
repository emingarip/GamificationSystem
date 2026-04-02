package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"gamification/models"
)

// RuleTransformerSystemPrompt is the master system prompt that instructs the LLM
// how to parse natural language gamification rules from Turkish/English input
// into structured JSON that the Rule Engine can execute.
const RuleTransformerSystemPrompt = `# Natural Language to Rule JSON Transformer - Türkçe öncelikli

Sen bir uzman yapay zeka asistanısın. Müşterilerin doğal dilde yazdığı gamifikasyon kurallarını,
sistemin çalıştırabileceği yapılandırılmış JSON'a dönüştürüyorsun.

## Platform Önceliği

Bu platform Türkiye pazarına yöneliktir. Türkçe girdiler öncelikli olarak işlenir.
İngilizce girdiler de desteklenmektedir.

## Temel Yetenekler

### 1. Niyet Anlama
Doğal dilden şunları çıkar:
- **Olay tetikleyicileri**: Gol, Korner, Foul, SarıKart, KırmızıKart, Penaltı, Ofsayt
- **Koşullar**: ardışık_sayı, zaman_penceresi, takım_spesifik, maç_türü
- **Ödüller**: puan_ver, rozet_ver, bildirim_gönder
- **Kapsam**: spesifik takımlar, maçlar, oyuncular, kullanıcı grupları

### 2. Kavram Eşleme

#### Türkçe → Sistem Eşlemeleri:
| Türkçe İfade | Sistem Kavramı |
|--------------|----------------|
| "derbi", "rival maç" | matchFilter.derby: true |
| "üst üste", "ardışık" | condition.type: temporal, sequence |
| "puan ver", "puan kazandır" | action.type: award_points |
| "rozet ver", "madalya ver" | action.type: grant_badge |
| "bildirim gönder" | action.type: send_notification |
| "takım taraftarları" | targeting.type: team_supporters |
| "izleyenler", "seyirciler" | targeting.type: match_participants |
| "her maçta" | targeting.type: all_users |
| "Galatasaray", "GS" | teamFilter: ["GS"] |
| "Fenerbahçe", "FB" | teamFilter: ["FB"] |
| "Beşiktaş", "BJK" | teamFilter: ["BJK"] |
| "Trabzonspor", "TS" | teamFilter: ["TS"] |

#### English → System Mappings:
| English Phrase | System Concept |
|----------------|----------------|
| "derby", "rivalry match" | matchFilter.derby: true |
| "consecutive", "in a row" | condition.type: temporal, sequence |
| "award points" | action.type: award_points |
| "give badge", "grant badge" | action.type: grant_badge |
| "send notification" | action.type: send_notification |
| "team supporters" | targeting.type: team_supporters |
| "viewers", "watchers" | targeting.type: match_participants |
| "every match" | targeting.type: all_users |
| "Galatasaray", "GS" | teamFilter: ["GS"] |
| "Fenerbahçe", "FB" | teamFilter: ["FB"] |
| "Beşiktaş", "BJK" | teamFilter: ["BJK"] |
| "Trabzonspor", "TS" | teamFilter: ["TS"] |

### 3. Desteklenen Olay Türleri

#### Spor Eventleri (özellikle Türkiye Süper Lig):
- "gol", "goal", "skor" → goal
- "korner", "corner" → corner
- "faul", "foul" → foul
- "sarı kart", "yellow card" → yellow_card
- "kırmızı kart", "red card" → red_card
- "penaltı", "penalty" → penalty
- "ofsayt", "offside" → offside

#### Generic/App Eventleri:
- "giriş", "login", "her gün" → daily_login
- "paylaş", "share", "arkadaş davet" → app_shared
- "satın alma", "purchase", "alışveriş" → purchase_completed
- "beğeni", "like", "oylama" → user_interaction
- "kayıt", "register", "üye ol" → user_registered

### 4. Derby Özel Durumu
Derbi maçları özel olarak ele alınmalıdır:
- GS vs FB (Galatasaray - Fenerbahçe)
- FB vs GS (Fenerbahçe - Galatasaray)
- BJK vs FB (Beşiktaş - Fenerbahçe)
- FB vs BJK (Fenerbahçe - Beşiktaş)
- GS vs BJK (Galatasaray - Beşiktaş)
- BJK vs GS (Beşiktaş - Galatasaray)

Derbi kelimesi geçiyorsa veya iki büyük takım belirtiliyorsa:
- targeting.type: "match_participants"
- matchFilter.derby: true
- Takım filtreleri otomatik olarak ekle

### 5. Koşul Türleri

#### simple (Basit)
- Doğrudan alan karşılaştırmaları
- Örnek: minute > 45, team_id == "GS"

#### aggregation (Toplama)
- Sayı tabanlı koşullar
- Örnek: count >= 3 (3 ardışık korner)

#### temporal (Zamansal)
- Zaman ve sıra tabanlı koşullar
- Örnek: sequence: ["corner", "corner", "corner"], windowSeconds: 300

## Çıktı Formatı

Geçerli JSON çıktısı MUTLAKA şu şemaya uymalıdır:

{
  "ruleId": "uuid-formatında-benzersiz-id",
  "name": "Türkçe veya İngilizce Kural Adı",
  "description": "Orijinal müşteri metni korunmalı",
  "enabled": true,
  "eventType": "Redis event type registry'deki herhangi bir key (örn: goal, daily_login, app_shared, purchase_completed)",
  "conditions": [
    {
      "type": "simple|aggregation|temporal",
      "field": "minute, team_id, player_id, match_id, value, count, subject_id, actor_id, source, context.*, metadata.*",
      "operator": "eq|gt|gte|lt|lte|in|contains",
      "threshold": 3,
      "windowSeconds": 300,
      "sequence": ["corner", "corner", "corner"]
    }
  ],
  "actions": [
    {
      "type": "award_points|grant_badge|send_notification",
      "value": 100,
      "badgeId": "opsiyonel-uuid",
      "message": "Tebrikler! | Congratulations!"
    }
  ],
  "targeting": {
    "type": "all_users|team_supporters|match_participants|custom",
    "teamFilter": ["GS", "FB"],
    "matchFilter": {"derby": true}
  },
  "cooldownSeconds": 60,
  "priority": 1
}

## Önemli Kurallar

1. **Her zaman UUID oluştur**: ruleId formatı "550e8400-e29b-41d4-a716-446655440000" gibi olmalı
2. **enabled varsayılan true**: Açıkça belirtilmedikçe true
3. **Açıklama korunmalı**: description alanında orijinal müşteri metni korunmalı
4. **Koşul uyumluluğu**: Koşullar birbiriyle uyumlu olmalı
5. **Derbi işleme**: GS-FB, FB-BJK, GS-BJK gibi derbi maçları otomatik algıla
6. **Sadece JSON çıktısı**: Açıklama, markdown veya JSON dışında metin YOK

## Örnek Dönüşümler

### Giriş (Türkçe):
"Derbide 3 korner üst üste olursa puan ver"

### Çıktı:
{
  "ruleId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "name": "Derbi Ardışık Korner",
  "description": "Derbide 3 korner üst üste olursa puan ver",
  "enabled": true,
  "eventType": "corner",
  "conditions": [
    {
      "type": "temporal",
      "field": "sequence",
      "operator": "eq",
      "sequence": ["corner", "corner", "corner"],
      "windowSeconds": 600
    }
  ],
  "actions": [
    {
      "type": "award_points",
      "value": 100,
      "message": "Tebrikler! Derbi korner başarısı!"
    }
  ],
  "targeting": {
    "type": "match_participants",
    "teamFilter": ["GS", "FB"],
    "matchFilter": {"derby": true}
  },
  "cooldownSeconds": 60,
  "priority": 10
}

### Giriş (English):
"Award 50 points when player scores a goal"

### Çıktı:
{
  "ruleId": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
  "name": "Goal Scorer Reward",
  "description": "Award 50 points when player scores a goal",
  "enabled": true,
  "eventType": "goal",
  "conditions": [
    {
      "type": "simple",
      "field": "value",
      "operator": "eq",
      "threshold": 1
    }
  ],
  "actions": [
    {
      "type": "award_points",
      "value": 50,
      "message": "Congratulations! Great goal!"
    }
  ],
  "targeting": {
    "type": "all_users"
  },
  "cooldownSeconds": 0,
  "priority": 5
}

### Giriş (Türkçe - Generic Event):
"Kullanıcı her gün uygulamaya giriş yaptığında 10 puan ver"

### Çıktı:
{
  "ruleId": "c3d4e5f6-a7b8-9012-cdef-345678901234",
  "name": "Günlük Giriş Ödülü",
  "description": "Kullanıcı her gün uygulamaya giriş yaptığında 10 puan ver",
  "enabled": true,
  "eventType": "daily_login",
  "conditions": [
    {
      "type": "simple",
      "field": "value",
      "operator": "eq",
      "threshold": 1
    }
  ],
  "actions": [
    {
      "type": "award_points",
      "value": 10,
      "message": "Günlük giriş ödülü kazandınız!"
    }
  ],
  "targeting": {
    "type": "all_users"
  },
  "cooldownSeconds": 86400,
  "priority": 1
}

### Giriş (English - Generic Event):
"When a user shares the app, grant a badge"

### Çıktı:
{
  "ruleId": "d4e5f6a7-b8c9-0123-defa-456789012345",
  "name": "App Sharer Badge",
  "description": "When a user shares the app, grant a badge",
  "enabled": true,
  "eventType": "app_shared",
  "conditions": [
    {
      "type": "simple",
      "field": "value",
      "operator": "eq",
      "threshold": 1
    }
  ],
  "actions": [
    {
      "type": "grant_badge",
      "badgeId": "app_sharer_badge",
      "message": "Thanks for sharing!"
    }
  ],
  "targeting": {
    "type": "all_users"
  },
  "cooldownSeconds": 0,
  "priority": 5
}

## Yanıt Gereksinimleri

1. **Her zaman geçerli JSON**: Markdown, açıklama veya JSON dışında metin YOK
2. **Türkçe girdi için Türkçe**: Kural adı ve açıklaması girdi dilinde
3. **Açıklamalar kısa**: Maksimum 500 karakter
4. **Tüm zorunlu alanlar**: Boş dizi/obje bile olsa dahil edilmeli
5. **Trailing comma YOK**: Sadece geçerli JSON`

// BuildRuleTransformerUserPrompt creates the user prompt for transforming a natural language rule
func BuildRuleTransformerUserPrompt(naturalLanguageRule string) string {
	return "Aşağıdaki doğal dil kuralını JSON'a dönüştür:\n\n" + naturalLanguageRule
}

// intermediateRuleJSON is an intermediate struct for parsing LLM output
// before converting to models.Rule
type intermediateRuleJSON struct {
	RuleID          string                  `json:"ruleId"`
	Name            string                  `json:"name"`
	Description     string                  `json:"description"`
	Enabled         bool                    `json:"enabled"`
	EventType       string                  `json:"eventType"`
	Conditions      []intermediateCondition `json:"conditions"`
	Actions         []intermediateAction    `json:"actions"`
	Targeting       intermediateTargeting   `json:"targeting"`
	CooldownSeconds int                     `json:"cooldownSeconds"`
	Priority        int                     `json:"priority"`
}

type intermediateCondition struct {
	Type          string      `json:"type"`
	Field         string      `json:"field"`
	Operator      string      `json:"operator"`
	Threshold     json.Number `json:"threshold"`
	WindowSeconds json.Number `json:"windowSeconds"`
	Sequence      []string    `json:"sequence"`
}

type intermediateAction struct {
	Type    string      `json:"type"`
	Value   json.Number `json:"value"`
	BadgeID string      `json:"badgeId"`
	Message string      `json:"message"`
}

type intermediateTargeting struct {
	Type        string   `json:"type"`
	TeamFilter  []string `json:"teamFilter"`
	MatchFilter struct {
		Derby   bool   `json:"derby"`
		MatchID string `json:"matchId"`
	} `json:"matchFilter"`
}

// NaturalLanguageToRuleParser transforms a natural language rule string into a models.Rule struct
// This function calls the LLM with the RuleTransformerSystemPrompt to parse Turkish/English input
// into a structured JSON that the Rule Engine can execute.
func NaturalLanguageToRuleParser(naturalRule string, llmClient *Client) (models.Rule, error) {
	if strings.TrimSpace(naturalRule) == "" {
		return models.Rule{}, fmt.Errorf("natural language rule cannot be empty")
	}

	// Build the user prompt
	userPrompt := BuildRuleTransformerUserPrompt(naturalRule)

	// Create the LLM request
	request := LLMRequest{
		Model: llmClient.modelName,
		Messages: []ChatMessage{
			{Role: "system", Content: RuleTransformerSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: llmClient.config.Temperature,
		TopP:        llmClient.config.TopP,
		MaxTokens:   llmClient.config.MaxTokens,
		Stream:      false,
		ResponseFormat: map[string]interface{}{
			"type": "json_object",
		},
	}

	// Send request to LLM
	responseContent, err := sendRuleTransformRequest(context.Background(), llmClient, request)
	if err != nil {
		return models.Rule{}, fmt.Errorf("failed to get LLM response: %w", err)
	}

	// Validate the JSON output against schema
	if err := ValidateRuleJSON([]byte(responseContent)); err != nil {
		return models.Rule{}, fmt.Errorf("LLM output validation failed: %w", err)
	}

	// Parse into models.Rule
	rule, err := parseToModelsRule(responseContent)
	if err != nil {
		return models.Rule{}, fmt.Errorf("failed to parse to models.Rule: %w", err)
	}

	return rule, nil
}

// sendRuleTransformRequest sends the request to LLM and returns the response content
func sendRuleTransformRequest(ctx context.Context, client *Client, request LLMRequest) (string, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", client.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	// Extract JSON from response (handle markdown blocks)
	content := response.Choices[0].Message.Content
	jsonStr := extractRuleJSON(content)

	if jsonStr == "" {
		return "", fmt.Errorf("no JSON found in response")
	}

	return jsonStr, nil
}

// extractRuleJSON extracts JSON from LLM response that might contain markdown or extra text
func extractRuleJSON(content string) string {
	// Try to find JSON block in markdown
	if strings.Contains(content, "```json") {
		start := strings.Index(content, "```json")
		content = content[start+7:]
		if end := strings.Index(content, "```"); end > 0 {
			content = content[:end]
		}
	} else if strings.Contains(content, "```") {
		start := strings.Index(content, "```")
		content = content[start+3:]
		if end := strings.Index(content, "```"); end > 0 {
			content = content[:end]
		}
	}

	// Trim whitespace
	content = strings.TrimSpace(content)

	// Check if it's wrapped in braces
	if strings.HasPrefix(content, "{") && strings.HasSuffix(content, "}") {
		return content
	}

	// Try to find the first { and last }
	start := strings.Index(content, "{")
	if start == -1 {
		return content
	}
	end := strings.LastIndex(content, "}")
	if end == -1 {
		return content
	}

	return content[start : end+1]
}

// parseToModelsRule parses the LLM JSON output into a models.Rule struct
func parseToModelsRule(jsonContent string) (models.Rule, error) {
	// Parse JSON using standard library
	var intermediate intermediateRuleJSON
	if err := json.Unmarshal([]byte(jsonContent), &intermediate); err != nil {
		return models.Rule{}, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Create rule with default values
	rule := models.Rule{
		IsActive:        true,
		Priority:        1,
		Conditions:      []models.RuleCondition{},
		TargetUsers:     models.TargetUsers{},
		Actions:         []models.RuleAction{},
		CooldownSeconds: 60,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Parse ruleId - generate UUID if not present
	if intermediate.RuleID != "" {
		rule.RuleID = intermediate.RuleID
	} else {
		rule.RuleID = uuid.New().String()
	}

	// Parse name
	rule.Name = intermediate.Name

	// Parse description - preserve original customer text
	rule.Description = intermediate.Description

	// Parse enabled
	rule.IsActive = intermediate.Enabled

	// Parse eventType
	rule.EventType = models.EventType(intermediate.EventType)

	// Parse priority
	if intermediate.Priority > 0 {
		rule.Priority = intermediate.Priority
	}

	// Parse cooldownSeconds
	if intermediate.CooldownSeconds > 0 {
		rule.CooldownSeconds = intermediate.CooldownSeconds
	}

	// Parse conditions
	for _, cond := range intermediate.Conditions {
		condition := models.RuleCondition{
			Field:          cond.Field,
			Operator:       cond.Operator,
			EvaluationType: cond.Type,
		}

		// Parse threshold if present
		if cond.Threshold != "" {
			if val, err := cond.Threshold.Int64(); err == nil {
				condition.Value = val
			}
		}

		// Parse windowSeconds for temporal conditions
		if cond.WindowSeconds != "" {
			if val, err := cond.WindowSeconds.Int64(); err == nil {
				condition.Value = val
			}
		}

		// Parse sequence for temporal conditions
		if len(cond.Sequence) > 0 {
			condition.Value = cond.Sequence
		}

		rule.Conditions = append(rule.Conditions, condition)
	}

	// Parse actions
	for _, action := range intermediate.Actions {
		actionType := action.Type
		params := make(map[string]any)

		// Parse action-specific parameters
		if action.Value != "" {
			if val, err := action.Value.Int64(); err == nil {
				params["value"] = int(val)
			}
		}
		if action.BadgeID != "" {
			params["badgeId"] = action.BadgeID
		}
		if action.Message != "" {
			params["message"] = action.Message
		}

		rule.Actions = append(rule.Actions, models.RuleAction{
			ActionType: actionType,
			Params:     params,
		})
	}

	// Parse targeting
	rule.TargetUsers = models.TargetUsers{
		QueryPattern: intermediate.Targeting.Type,
		Params:       make(map[string]string),
	}

	// Parse teamFilter
	if len(intermediate.Targeting.TeamFilter) > 0 {
		rule.TargetUsers.Params["teamFilter"] = strings.Join(intermediate.Targeting.TeamFilter, ",")
	}

	// Parse matchFilter
	if intermediate.Targeting.MatchFilter.Derby {
		rule.TargetUsers.Params["derby"] = "true"
	}
	if intermediate.Targeting.MatchFilter.MatchID != "" {
		rule.TargetUsers.Params["matchId"] = intermediate.Targeting.MatchFilter.MatchID
	}

	return rule, nil
}

// TransformNaturalLanguageRule is a convenience function that takes a natural language rule
// and an LLM client, returning a structured models.Rule
func TransformNaturalLanguageRule(naturalRule string, llmClient *Client) (models.Rule, error) {
	return NaturalLanguageToRuleParser(naturalRule, llmClient)
}
