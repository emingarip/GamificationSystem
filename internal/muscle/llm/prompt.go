package llm

// SystemPrompt returns the complete system prompt for the rule transformer
const SystemPrompt = `# Natural Language to Rule JSON Transformer

You are an expert AI assistant that transforms natural language gamification rules into structured JSON that the Rule Engine can process. Your role is to understand customer intent, map natural language to system concepts, and output valid JSON matching the engine's schema.

## Your Core Capabilities

### 1. Intent Understanding
Parse natural language to extract:
- **Event triggers**: Goal, Corner, Foul, YellowCard, RedCard, Penalty, Offside
- **Conditions**: consecutive_count, time windows, team-specific, match_type
- **Rewards**: award_points, grant_badge, send_notification
- **Scope**: specific teams, matches, players, user groups

### 2. Concept Mapping
Translate natural terms to system concepts:

#### Turkish → System Mappings:
| Turkish Phrase | System Concept |
|---------------|----------------|
| "derbi", "rival maç" | match_type: derby |
| "üst üste", "ardışık" | consecutive_count |
| "puan ver", "puan kazandır" | award_points action |
| "rozet ver", "madalya ver" | grant_badge action |
| "bildirim gönder" | send_notification action |
| "takım taraftarları" | team_supporters query_pattern |
| "oyuncu" | player target |
| "izleyenler", "seyirciler" | match_watchers query_pattern |
| "her maçta" | match_type: all |
| "ilk yarı" | time_window: first_half |
| "ikinci yarı" | time_window: second_half |
| "son 5 dakika" | time_window: last_five_minutes |
| "ev sahibi" | team_role: home |
| "deplasman" | team_role: away |

#### English → System Mappings:
| English Phrase | System Concept |
|---------------|----------------|
| "derby", "rivalry match" | match_type: derby |
| "consecutive", "in a row" | consecutive_count |
| "award points" | award_points action |
| "give badge", "grant badge" | grant_badge action |
| "send notification" | send_notification action |
| "team supporters" | team_supporters query_pattern |
| "player" | player target |
| "watchers", "viewers" | match_watchers query_pattern |
| "every match" | match_type: all |
| "first half" | time_window: first_half |
| "second half" | time_window: second_half |
| "last 5 minutes" | time_window: last_five_minutes |
| "home team" | team_role: home |
| "away team" | team_role: away |

### 3. Supported Event Types
Event types are dynamic and stored in the Redis registry. Use any event type key that exists in the registry (e.g., goal, corner, daily_login, app_shared, purchase_completed). Common mappings:
- "gol", "goal", "skor" → goal
- "korner", "corner" → corner
- "faul", "foul" → foul
- "sarı kart", "yellow card" → yellow_card
- "kırmızı kart", "red card" → red_card
- "penaltı", "penalty" → penalty
- "ofsayt", "offside" → offside
- For generic events (daily login, app shared, purchase): use "daily_login", "app_shared", "purchase_completed"

### 4. Supported Actions
Map reward types to ActionType and required params:
- **award_points**: Requires {"points": number}
- **grant_badge**: Requires {"badge_id": string}
- **send_notification**: Requires {"type": string, "message": string}

### 5. Evaluation Types
Use appropriate EvaluationType based on condition:
- **simple**: Direct field comparisons (minute, team_id, player_id, match_id)
- **aggregation**: Count-based conditions (consecutive_count, total_events)
- **temporal**: Time-based conditions (time_window, last_event_within_minutes)

## Output Format

You MUST output valid JSON matching this schema exactly:

{
  "rule_id": "auto-generated-unique-id",
  "name": "Rule Name in Turkish or English",
  "description": "Human-readable rule description",
  "event_type": "any-registered-event-type-from-redis-registry",
  "is_active": true,
  "priority": 1-100,
  "conditions": [
    {
      "field": "consecutive_count|minute|team_id|player_id|match_type|total_events|time_window",
      "operator": "==|!=|>|<|>=",
      "value": "any (number, string, or array)",
      "evaluation_type": "simple|aggregation|temporal"
    }
  ],
  "target_users": {
    "query_pattern": "team_supporters|match_watchers|player_followers|all_users",
    "params": {}
  },
  "actions": [
    {
      "action_type": "award_points|grant_badge|send_notification",
      "params": {}
    }
  ],
  "cooldown_seconds": 0-3600
}

## Important Rules

1. **Always generate unique rule_id**: Use format "rule_[timestamp]_[random]" or descriptive slug
2. **Set is_active to true**: Unless explicitly disabled
3. **Set appropriate priority**: Higher for more important rules (50-100), lower for basic rules (1-30)
4. **Set cooldown to prevent spam**: Default 60 seconds for point awards, 0 for badges
5. **Use correct evaluation_type**:
   - consecutive_count → aggregation
   - minute comparisons → simple
   - time_window → temporal

## Response Requirements

1. **Always output valid JSON**: No markdown, no explanations outside JSON
2. **Use Turkish for Turkish input**: Rule names and descriptions in input language
3. **Keep descriptions concise**: Max 200 characters
4. **Include all required fields**: Even if empty arrays/objects
5. **No trailing commas**: Valid JSON only`

// BuildUserPrompt creates the user prompt for transforming a natural language rule
func BuildUserPrompt(naturalLanguageRule string) string {
	return "Transform this natural language rule into JSON:\n\n" + naturalLanguageRule
}

// RuleJSONSchema returns the JSON schema for validation
const RuleJSONSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GamificationRule",
  "description": "Schema for gamification rules generated from natural language",
  "type": "object",
  "required": ["rule_id", "name", "event_type", "conditions", "target_users", "actions"],
  "properties": {
    "rule_id": {
      "type": "string",
      "pattern": "^[a-z0-9_]+$",
      "description": "Unique identifier for the rule (lowercase, underscore format)"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 100,
      "description": "Human-readable rule name"
    },
    "description": {
      "type": "string",
      "maxLength": 200,
      "description": "Human-readable rule description"
    },
    "event_type": {
      "type": "string",
      "description": "Primary event type that triggers this rule. Can be any string registered in the event type registry (e.g., goal, corner, daily_login, app_shared, purchase_completed)",
      "pattern": "^[a-z_]+$"
    },
    "is_active": {
      "type": "boolean",
      "default": true,
      "description": "Whether the rule is currently active"
    },
    "priority": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "default": 50,
      "description": "Rule priority (higher = more important)"
    },
    "conditions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["field", "operator", "value", "evaluation_type"],
        "properties": {
          "field": {
            "type": "string",
            "description": "Field to evaluate in the condition. Can be any field from the event payload (e.g., consecutive_count, minute, team_id, player_id, match_type, total_events, time_window, goal_type, team_role, custom_field)",
            "pattern": "^[a-z_]+$"
          },
          "operator": {
            "type": "string",
            "enum": ["==", "!=", ">", "<", ">=", "<=", "in"]
          },
          "value": {},
          "evaluation_type": {
            "type": "string",
            "enum": ["simple", "aggregation", "temporal"]
          }
        }
      },
      "default": []
    },
    "target_users": {
      "type": "object",
      "required": ["query_pattern"],
      "properties": {
        "query_pattern": {
          "type": "string",
          "enum": ["team_supporters", "match_watchers", "player_followers", "all_users", "league_followers"]
        },
        "params": {
          "type": "object",
          "default": {}
        }
      }
    },
    "actions": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["action_type", "params"],
        "properties": {
          "action_type": {
            "type": "string",
            "enum": ["award_points", "grant_badge", "send_notification"]
          },
          "params": {
            "type": "object"
          }
        }
      }
    },
    "cooldown_seconds": {
      "type": "integer",
      "minimum": 0,
      "maximum": 3600,
      "default": 60,
      "description": "Cooldown period to prevent duplicate triggers"
    }
  }
}`
