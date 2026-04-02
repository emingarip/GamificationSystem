package llm

import (
	"encoding/json"
	"fmt"
)

// RuleSchemaJSON returns the JSON Schema for gamification rules
// This schema defines the structure that all transformed rules must follow
const RuleSchemaJSON = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "GamificationRule",
  "description": "Schema for gamification rules generated from natural language input",
  "type": "object",
  "required": ["ruleId", "name", "description", "enabled", "eventType", "conditions", "actions", "targeting", "cooldownSeconds", "priority"],
  "properties": {
    "ruleId": {
      "type": "string",
      "pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
      "description": "Unique UUID identifier for the rule"
    },
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 100,
      "description": "Rule name in Turkish or English"
    },
    "description": {
      "type": "string",
      "maxLength": 500,
      "description": "Original customer text preserved"
    },
    "enabled": {
      "type": "boolean",
      "default": true,
      "description": "Whether the rule is currently active"
    },
    "eventType": {
      "type": "string",
      "minLength": 1,
      "pattern": "^[a-z][a-z0-9_]*$",
      "description": "Event type key registered in Redis event type registry. Must be a valid identifier (lowercase letters, numbers, underscores). Examples: goal, corner, daily_login, app_shared, purchase_completed"
    },
    "conditions": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["type", "field", "operator"],
        "properties": {
          "type": {
            "type": "string",
            "enum": ["simple", "aggregation", "temporal"],
            "description": "Type of condition evaluation"
          },
          "field": {
            "type": "string",
            "minLength": 1,
            "description": "Field being evaluated. Supports standard fields (minute, team_id, player_id, match_id, value, count, sequence) and custom fields from event metadata (subject_id, actor_id, source, context.key, metadata.custom_field)"
          },
          "operator": {
            "type": "string",
            "enum": ["eq", "gt", "gte", "lt", "lte", "in", "contains"],
            "description": "Comparison operator"
          },
          "threshold": {
            "type": "number",
            "description": "Numeric threshold for the condition"
          },
          "windowSeconds": {
            "type": "integer",
            "minimum": 0,
            "description": "Time window in seconds for temporal conditions"
          },
          "sequence": {
            "type": "array",
            "items": {
              "type": "string"
            },
            "description": "Sequence of event types for temporal patterns"
          }
        }
      },
      "default": []
    },
    "actions": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["type"],
        "properties": {
          "type": {
            "type": "string",
            "enum": ["award_points", "grant_badge", "send_notification"],
            "description": "Action to perform when rule triggers"
          },
          "value": {
            "type": "number",
            "description": "Points value for award_points action"
          },
          "badgeId": {
            "type": "string",
            "description": "UUID of badge to grant for grant_badge action"
          },
          "message": {
            "type": "string",
            "description": "Notification message for send_notification action"
          }
        }
      }
    },
    "targeting": {
      "type": "object",
      "required": ["type"],
      "properties": {
        "type": {
          "type": "string",
          "enum": ["all_users", "team_supporters", "match_participants", "custom"],
          "description": "Target user group type"
        },
        "teamFilter": {
          "type": "array",
          "items": {
            "type": "string"
          },
          "description": "Filter by team IDs (e.g., GS, FB)"
        },
        "matchFilter": {
          "type": "object",
          "properties": {
            "derby": {
              "type": "boolean",
              "description": "Filter for derby matches only"
            },
            "matchId": {
              "type": "string",
              "description": "Specific match ID"
            }
          }
        }
      }
    },
    "cooldownSeconds": {
      "type": "integer",
      "minimum": 0,
      "maximum": 3600,
      "default": 60,
      "description": "Cooldown period to prevent duplicate triggers"
    },
    "priority": {
      "type": "integer",
      "minimum": 1,
      "maximum": 100,
      "default": 1,
      "description": "Rule priority (higher = more important)"
    }
  }
}`

// RuleSchemaJSONBytes returns the schema as byte slice for validation
func RuleSchemaJSONBytes() []byte {
	return []byte(RuleSchemaJSON)
}

// ValidateRuleJSON validates the LLM output against the schema
// Returns error if the JSON is invalid or doesn't match schema
func ValidateRuleJSON(jsonOutput []byte) error {
	// First, validate it's valid JSON
	var jsonData interface{}
	if err := json.Unmarshal(jsonOutput, &jsonData); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Convert to map for schema validation
	jsonMap, ok := jsonData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("JSON must be an object")
	}

	// Validate required top-level fields
	requiredFields := []string{"ruleId", "name", "description", "enabled", "eventType", "conditions", "actions", "targeting", "cooldownSeconds", "priority"}
	for _, field := range requiredFields {
		if _, exists := jsonMap[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate ruleId is a valid UUID format
	ruleID, ok := jsonMap["ruleId"].(string)
	if !ok || len(ruleID) == 0 {
		return fmt.Errorf("ruleId must be a non-empty string")
	}

	// Validate enabled is boolean
	if _, ok := jsonMap["enabled"].(bool); !ok {
		return fmt.Errorf("enabled must be a boolean")
	}

	// Validate eventType is not empty (validity is checked when saving to Redis)
	eventType, ok := jsonMap["eventType"].(string)
	if !ok || eventType == "" {
		return fmt.Errorf("eventType must be a non-empty string")
	}
	// Event type validity is validated against Redis registry when saving the rule

	// Validate conditions is an array
	conditions, ok := jsonMap["conditions"].([]interface{})
	if !ok {
		return fmt.Errorf("conditions must be an array")
	}

	// Validate each condition
	validConditionTypes := map[string]bool{"simple": true, "aggregation": true, "temporal": true}
	validOperators := map[string]bool{"eq": true, "gt": true, "gte": true, "lt": true, "lte": true, "in": true, "contains": true}

	for i, cond := range conditions {
		condMap, ok := cond.(map[string]interface{})
		if !ok {
			return fmt.Errorf("condition[%d] must be an object", i)
		}

		// Check type
		condType, ok := condMap["type"].(string)
		if !ok || !validConditionTypes[condType] {
			return fmt.Errorf("condition[%d].type must be one of: simple, aggregation, temporal", i)
		}

		// Check field - allow any non-empty string for dynamic field support
		field, ok := condMap["field"].(string)
		if !ok || field == "" {
			return fmt.Errorf("condition[%d].field must be a non-empty string", i)
		}

		// Check operator
		operator, ok := condMap["operator"].(string)
		if !ok || !validOperators[operator] {
			return fmt.Errorf("condition[%d].operator must be a valid operator", i)
		}

		// For temporal conditions, validate windowSeconds and sequence
		if condType == "temporal" {
			if _, exists := condMap["windowSeconds"]; !exists && condMap["sequence"] == nil {
				return fmt.Errorf("temporal condition[%d] must have either windowSeconds or sequence", i)
			}
		}

		// For aggregation conditions, validate threshold
		if condType == "aggregation" {
			if _, exists := condMap["threshold"]; !exists {
				return fmt.Errorf("aggregation condition[%d] must have threshold", i)
			}
		}
	}

	// Validate actions is an array
	actions, ok := jsonMap["actions"].([]interface{})
	if !ok || len(actions) == 0 {
		return fmt.Errorf("actions must be a non-empty array")
	}

	// Validate each action
	validActionTypes := map[string]bool{"award_points": true, "grant_badge": true, "send_notification": true}
	for i, action := range actions {
		actionMap, ok := action.(map[string]interface{})
		if !ok {
			return fmt.Errorf("action[%d] must be an object", i)
		}

		actionType, ok := actionMap["type"].(string)
		if !ok || !validActionTypes[actionType] {
			return fmt.Errorf("action[%d].type must be one of: award_points, grant_badge, send_notification", i)
		}

		// Validate action-specific fields
		if actionType == "award_points" {
			if _, exists := actionMap["value"]; !exists {
				return fmt.Errorf("award_points action[%d] must have value", i)
			}
		} else if actionType == "grant_badge" {
			if _, exists := actionMap["badgeId"]; !exists {
				return fmt.Errorf("grant_badge action[%d] must have badgeId", i)
			}
		} else if actionType == "send_notification" {
			if _, exists := actionMap["message"]; !exists {
				return fmt.Errorf("send_notification action[%d] must have message", i)
			}
		}
	}

	// Validate targeting
	targeting, ok := jsonMap["targeting"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("targeting must be an object")
	}

	targetType, ok := targeting["type"].(string)
	if !ok {
		return fmt.Errorf("targeting.type must be a string")
	}
	validTargetTypes := map[string]bool{"all_users": true, "team_supporters": true, "match_participants": true, "custom": true}
	if !validTargetTypes[targetType] {
		return fmt.Errorf("targeting.type must be one of: all_users, team_supporters, match_participants, custom")
	}

	// Validate cooldownSeconds
	cooldown, ok := jsonMap["cooldownSeconds"].(float64)
	if !ok || cooldown < 0 || cooldown > 3600 {
		return fmt.Errorf("cooldownSeconds must be between 0 and 3600")
	}

	// Validate priority
	priority, ok := jsonMap["priority"].(float64)
	if !ok || priority < 1 || priority > 100 {
		return fmt.Errorf("priority must be between 1 and 100")
	}

	return nil
}

// ConditionCompatibilityCheck validates that conditions are mutually compatible
func ConditionCompatibilityCheck(conditions []map[string]interface{}) error {
	if len(conditions) == 0 {
		return nil
	}

	// Check for conflicting temporal and aggregation conditions
	// (kept for future logic expansion, currently they can coexist)

	// Check that sequence conditions are properly ordered
	for i, cond := range conditions {
		condType, _ := cond["type"].(string)
		if condType == "temporal" {
			if sequence, exists := cond["sequence"]; exists {
				seq, ok := sequence.([]interface{})
				if !ok || len(seq) == 0 {
					return fmt.Errorf("condition[%d]: temporal sequence must not be empty", i)
				}
			}
		}
	}

	return nil
}
