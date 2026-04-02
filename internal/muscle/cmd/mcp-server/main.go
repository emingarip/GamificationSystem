package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"gamification/config"
	"gamification/engine"
	"gamification/mcp/backend"
	"gamification/mcp/resources"
	"gamification/neo4j"
	"gamification/redis"
	"gamification/websocket"

	mcpjsonrpc "github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	version = "1.0.0"
)

// NOTE: This MCP server operates as an internal trusted tool surface.
// Write operations (update_user_points, assign_badge_to_user) bypass user-level
// authorization and should only be used by trusted AI agents with appropriate
// system-level access. Authentication is delegated to the calling agent.

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Parse command line flags
	transport := flag.String("transport", "stdio", "Transport type: stdio (default) or http (streamable-http for remote MCP clients)")
	port := flag.Int("port", 3002, "HTTP port for http transport")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Override transport from environment if set
	if envTransport := os.Getenv("TRANSPORT"); envTransport != "" {
		*transport = envTransport
	}

	if *showVersion {
		fmt.Printf("MCP Gamification Server v%s\n", version)
		fmt.Println("Supported transports: stdio, http (streamable-http)")
		os.Exit(0)
	}

	log.Printf("Starting MCP Gamification Server v%s", version)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: failed to load config from env, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	// Initialize Redis client
	var redisClient *redis.Client
	redisClient, err = redis.NewClient(&cfg.Redis)
	if err != nil {
		log.Printf("Warning: failed to connect to Redis: %v", err)
		log.Println("Running in limited mode (analytics will be unavailable)")
		redisClient = nil
	} else {
		log.Println("Connected to Redis")
		defer redisClient.Close()

		// Seed default event types to Redis registry (idempotent - safe to call on every startup)
		log.Println("Seeding default event types to registry...")
		if err := redisClient.SeedDefaultEventTypes(ctx); err != nil {
			log.Printf("Warning: failed to seed event types: %v", err)
		} else {
			log.Println("Default event types seeded successfully")
		}
	}

	// Initialize Neo4j client
	var neo4jClient *neo4j.Client
	neo4jClient, err = neo4j.NewClient(&cfg.Neo4j)
	if err != nil {
		log.Printf("Warning: failed to connect to Neo4j: %v", err)
		log.Println("Running in limited mode (user/badge operations will be unavailable)")
		neo4jClient = nil
	} else {
		log.Println("Connected to Neo4j")
		defer neo4jClient.Close()
	}

	// Create backend service layer
	backendService := backend.NewService(redisClient, neo4jClient)

	// Initialize rule engine and reward layer if clients are available
	if redisClient != nil && neo4jClient != nil {
		// Create WebSocket server (nil for MCP - no real-time notifications)
		wsServer := &websocket.Server{}

		// Create reward layer
		rewardLayer := engine.NewRewardLayer(redisClient, neo4jClient, wsServer)
		backendService.SetRewardLayer(rewardLayer)

		// Create rule engine
		ruleEngine := engine.NewRuleEngine(cfg, redisClient, neo4jClient)
		ruleEngine.SetRewardLayer(rewardLayer)
		backendService.SetRuleEngine(ruleEngine)

		log.Println("Rule engine initialized")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down MCP server...")
		cancel()
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	// Create and run MCP server
	server := NewMCPServer(backendService)

	switch *transport {
	case "stdio":
		log.Println("Starting MCP server with stdio transport")
		if err := server.RunStdio(ctx); err != nil {
			log.Printf("Server error: %v", err)
			os.Exit(1)
		}
	case "http":
		log.Printf("Starting MCP server with HTTP transport on port %d", *port)
		if err := server.RunHTTP(ctx, *port); err != nil {
			log.Printf("Server error: %v", err)
			os.Exit(1)
		}
	default:
		log.Fatalf("Unknown transport type: %s", *transport)
	}
}

// MCPServer represents a minimal MCP server implementation
type MCPServer struct {
	backend *backend.Service
}

// NewMCPServer creates a new MCP server
func NewMCPServer(backendService *backend.Service) *MCPServer {
	return &MCPServer{
		backend: backendService,
	}
}

// RunStdio runs the MCP server with stdio transport
func (s *MCPServer) RunStdio(ctx context.Context) error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			// Read next JSON-RPC message
			var msg jsonRPCMessage
			if err := decoder.Decode(&msg); err != nil {
				if err.Error() == "EOF" {
					return nil
				}
				log.Printf("Error decoding message: %v", err)
				continue
			}

			// Process message and send response
			response := s.processMessage(ctx, msg)
			if response != nil {
				if err := encoder.Encode(response); err != nil {
					log.Printf("Error encoding response: %v", err)
				}
			}
		}
	}
}

// RunHTTP runs the MCP server with HTTP transport
func (s *MCPServer) RunHTTP(ctx context.Context, port int) error {
	mux := http.NewServeMux()
	mux.Handle("/mcp", s.newStreamableHTTPHandler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "gamification-mcp",
			"version": version,
		})
	})

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	return server.Shutdown(context.Background())
}

// Known JSON-RPC methods and notifications
var knownMethods = map[string]bool{
	"initialize":     true,
	"tools/list":     true,
	"tools/call":     true,
	"resources/list": true,
	"resources/read": true,
	"prompts/list":   true,
	"prompts/get":    true,
}

var knownNotifications = map[string]bool{
	"notifications/initialized": true,
}

// isNotification checks if a message is a JSON-RPC notification (no ID)
func isNotification(msg jsonRPCMessage) bool {
	return msg.ID == nil
}

// isKnownNotification checks if the method is a known notification
func isKnownNotification(method string) bool {
	return knownNotifications[method]
}

// processMessage processes an incoming JSON-RPC message
func (s *MCPServer) processMessage(ctx context.Context, msg jsonRPCMessage) *jsonRPCMessage {
	// Handle notifications (messages without ID) - no response should be sent
	if isNotification(msg) {
		// Handle known notifications
		if isKnownNotification(msg.Method) {
			s.handleNotification(ctx, msg)
		}
		// Unknown notifications are silently ignored per JSON-RPC spec
		return nil
	}

	// Handle initialize request
	if msg.Method == "initialize" {
		return s.handleInitialize(msg)
	}

	// Handle tools/list request
	if msg.Method == "tools/list" {
		return s.handleToolsList(msg)
	}

	// Handle tools/call request
	if msg.Method == "tools/call" {
		return s.handleToolsCall(ctx, msg)
	}

	// Handle resources/list request
	if msg.Method == "resources/list" {
		return s.handleResourcesList(msg)
	}

	// Handle resources/read request
	if msg.Method == "resources/read" {
		return s.handleResourcesRead(ctx, msg)
	}

	// Handle prompts/list request
	if msg.Method == "prompts/list" {
		return s.handlePromptsList(msg)
	}

	// Handle prompts/get request
	if msg.Method == "prompts/get" {
		return s.handlePromptsGet(msg)
	}

	// Unknown method - return error
	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Error: &jsonRPCError{
			Code:    -32601,
			Message: "Method not found",
		},
	}
}

// handleNotification handles JSON-RPC notifications
func (s *MCPServer) handleNotification(ctx context.Context, msg jsonRPCMessage) {
	// Handle initialized notification
	if msg.Method == "notifications/initialized" {
		log.Println("Client initialized notification received")
	}
}

// ==================== JSON-RPC Types ====================

type jsonRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// ==================== Initialize Handler ====================

func (s *MCPServer) handleInitialize(msg jsonRPCMessage) *jsonRPCMessage {
	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]any{
				"name":    "gamification-mcp",
				"version": version,
			},
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
				"prompts":   map[string]any{},
			},
		},
	}
}

func schemaString(description string) map[string]any {
	schema := map[string]any{"type": "string"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func schemaInteger(description string) map[string]any {
	schema := map[string]any{"type": "integer"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func schemaNumber(description string) map[string]any {
	schema := map[string]any{"type": "number"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func schemaBoolean(description string) map[string]any {
	schema := map[string]any{"type": "boolean"}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func schemaArray(items any, description string) map[string]any {
	schema := map[string]any{
		"type":  "array",
		"items": items,
	}
	if description != "" {
		schema["description"] = description
	}
	return schema
}

func schemaObject(properties map[string]any, required ...string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func ruleSchema() map[string]any {
	return schemaObject(map[string]any{
		"id":          schemaString("Unique rule identifier"),
		"name":        schemaString("Rule name"),
		"event_type":  schemaString("Event type that triggers the rule"),
		"points":      schemaInteger("Points awarded by the rule"),
		"enabled":     schemaBoolean("Whether the rule is enabled"),
		"description": schemaString("Human-readable rule description"),
	}, "id", "name", "event_type", "points", "enabled", "description")
}

func listRulesOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"rules":             schemaArray(ruleSchema(), "Active rules that matched the optional filter"),
		"count":             schemaInteger("Number of returned rules"),
		"event_type_filter": map[string]any{"type": []string{"string", "null"}, "description": "Applied event type filter, or null when no filter was provided"},
	}, "rules", "count")
}

func getRuleOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"rule": ruleSchema(),
	}, "rule")
}

func triggeredRuleSchema() map[string]any {
	return schemaObject(map[string]any{
		"rule_id":      schemaString("Triggered rule identifier"),
		"name":         schemaString("Triggered rule name"),
		"matched":      schemaBoolean("Whether the rule matched the event"),
		"eval_time_ms": schemaNumber("Rule evaluation time in milliseconds"),
		"users":        schemaArray(schemaString("User ID"), "Users targeted by the rule"),
		"actions":      schemaInteger("Number of actions prepared for execution"),
		"actions_detail": schemaArray(
			map[string]any{"type": "object"},
			"Detailed action payloads when the rule matched",
		),
	}, "rule_id", "name", "matched", "eval_time_ms", "users", "actions")
}

func testEventOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"success":         schemaBoolean("Whether rule evaluation completed successfully"),
		"event_id":        schemaString("Resolved event identifier"),
		"event_type":      schemaString("Resolved event type"),
		"dry_run":         schemaBoolean("Whether actions were only evaluated"),
		"total_time_ms":   schemaNumber("Total processing time in milliseconds"),
		"matched_rules":   schemaInteger("Number of matched rules"),
		"error":           map[string]any{"type": []string{"string", "null"}, "description": "Processing error when evaluation failed"},
		"triggered_rules": schemaArray(triggeredRuleSchema(), "Triggered rule evaluation results"),
	}, "success", "event_id", "event_type", "dry_run", "total_time_ms", "matched_rules", "triggered_rules")
}

func assignBadgeOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"success": schemaBoolean("Whether the badge assignment succeeded"),
		"message": schemaString("Human-readable badge assignment summary"),
	}, "success", "message")
}

func updateUserPointsOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"success":    schemaBoolean("Whether the points update succeeded"),
		"new_points": schemaInteger("User points after the update"),
	}, "success", "new_points")
}

func userInfoSchema() map[string]any {
	return schemaObject(map[string]any{
		"id":        schemaString("User identifier"),
		"name":      schemaString("User display name"),
		"points":    schemaInteger("Current user points"),
		"badges":    schemaInteger("Number of earned badges"),
		"level":     schemaInteger("Current user level"),
		"joined_at": schemaString("ISO-8601 join timestamp"),
	}, "id", "name", "points", "badges", "level", "joined_at")
}

func listUsersOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"users": schemaArray(userInfoSchema(), "Returned user page"),
		"count": schemaInteger("Number of returned users"),
	}, "users", "count")
}

func badgeSchema() map[string]any {
	return schemaObject(map[string]any{
		"badge_id":    schemaString("Badge identifier"),
		"name":        schemaString("Badge name"),
		"description": schemaString("Badge description"),
		"icon":        schemaString("Badge icon or asset identifier"),
		"points":      schemaInteger("Points associated with the badge"),
		"earned_at":   schemaString("ISO-8601 badge earn timestamp"),
	}, "badge_id", "name", "description", "icon", "points", "earned_at")
}

func activitySchema() map[string]any {
	return schemaObject(map[string]any{
		"action_type": schemaString("Action type recorded in recent activity"),
		"points":      schemaInteger("Points awarded or deducted"),
		"reason":      schemaString("Reason for the activity entry"),
		"timestamp":   schemaString("ISO-8601 activity timestamp"),
	}, "action_type", "points", "reason", "timestamp")
}

func userProfileOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"user_id":         schemaString("User identifier"),
		"name":            schemaString("User display name"),
		"email":           schemaString("User email"),
		"points":          schemaInteger("Current user points"),
		"level":           schemaInteger("Current user level"),
		"created_at":      schemaString("ISO-8601 account creation timestamp"),
		"stats":           map[string]any{"type": "object", "description": "Aggregated user statistics"},
		"badges":          schemaArray(badgeSchema(), "Earned badges"),
		"recent_activity": schemaArray(activitySchema(), "Recent user activity"),
	}, "user_id", "name", "email", "points", "level", "created_at", "stats", "badges", "recent_activity")
}

func analyticsSummaryOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"total_users":         schemaInteger("Total users"),
		"total_badges_earned": schemaInteger("Total earned badges"),
		"total_points":        schemaInteger("Total distributed points"),
		"active_users_30d":    schemaInteger("Active users in the last 30 days"),
		"badge_catalog":       schemaInteger("Total badge catalog entries"),
		"active_rules":        schemaInteger("Total active rules"),
	}, "total_users", "total_badges_earned", "total_points", "active_users_30d", "badge_catalog", "active_rules")
}

func listEventTypesOutputSchema() map[string]any {
	return schemaObject(map[string]any{
		"event_types": schemaArray(map[string]any{
			"type": "object",
			"properties": map[string]any{
				"key":         schemaString("Event type key"),
				"name":        schemaString("Event type name"),
				"description": schemaString("Event type description"),
				"category":    schemaString("Event type category"),
				"enabled":     schemaBoolean("Whether the event type is enabled"),
				"created_at":  schemaString("ISO-8601 creation timestamp"),
				"updated_at":  schemaString("ISO-8601 update timestamp"),
			},
		}, "List of registered event types"),
		"count": schemaInteger("Number of returned event types"),
	}, "event_types", "count")
}

func toolSuccessResponse(id any, structuredContent any) *jsonRPCMessage {
	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]any{
			"content": []map[string]any{
				{
					"type": "text",
					"text": formatJSON(structuredContent),
				},
			},
			"structuredContent": structuredContent,
			"isError":           false,
		},
	}
}

// ==================== Tools Handlers ====================

func (s *MCPServer) handleToolsList(msg jsonRPCMessage) *jsonRPCMessage {
	tools := []map[string]any{
		{
			"name":        "list_rules",
			"description": "Lists active rules from Redis, filtered by event type. Returns only enabled/active rules.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"event_type": map[string]any{
						"type":        "string",
						"description": "Optional filter by event type (e.g., goal, corner, foul)",
					},
				},
			},
			"outputSchema": listRulesOutputSchema(),
		},
		{
			"name":        "get_rule",
			"description": "Get detailed information about a specific rule by its ID.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"rule_id": map[string]any{
						"type":        "string",
						"description": "The unique identifier of the rule",
					},
				},
				"required": []string{"rule_id"},
			},
			"outputSchema": getRuleOutputSchema(),
		},
		{
			"name":        "test_event",
			"description": "Test how an event would be processed by the rule engine. Use dry_run=true to evaluate without executing actions. If event_id is not provided, a UUID will be auto-generated. For generic events (daily_login, app_shared, etc.), only event_type is required. For sports events, match_id and player_id may be required depending on rule conditions.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"event": map[string]any{
						"type":        "object",
						"description": "The event to test (event_type is always required, other fields are optional)",
						"properties": map[string]any{
							"event_id":   map[string]any{"type": "string", "description": "Unique identifier for the event. If not provided, a UUID will be auto-generated."},
							"event_type": map[string]any{"type": "string", "description": "The type of event (e.g., goal, corner, foul, daily_login, app_shared, purchase_completed)", "minLength": 1},
							"match_id":   map[string]any{"type": "string", "description": "The match identifier (optional for generic events)"},
							"team_id":    map[string]any{"type": "string", "description": "The team identifier (optional)"},
							"player_id":  map[string]any{"type": "string", "description": "The player identifier (optional for generic events)"},
							"minute":     map[string]any{"type": "integer", "description": "Match minute when event occurred (optional)"},
							"metadata":   map[string]any{"type": "object", "description": "Additional event metadata (optional)"},
						},
						"required": []string{"event_type"},
					},
					"dry_run": map[string]any{
						"type":        "boolean",
						"description": "If true, only evaluate rules without executing actions (default: true)",
					},
				},
				"required": []string{"event"},
			},
			"outputSchema": testEventOutputSchema(),
		},
		{
			"name":        "assign_badge_to_user",
			"description": "Manually assign a badge to a user. NOTE: This is a write operation that bypasses normal authorization.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{
						"type":        "string",
						"description": "The user ID to assign the badge to",
					},
					"badge_id": map[string]any{
						"type":        "string",
						"description": "The badge ID to assign",
					},
					"reason": map[string]any{
						"type":        "string",
						"description": "Reason for the badge assignment (optional)",
					},
				},
				"required": []string{"user_id", "badge_id"},
			},
			"outputSchema": assignBadgeOutputSchema(),
		},
		{
			"name":        "update_user_points",
			"description": "Add, subtract, or set user points. Supports 'add' (default), 'subtract', and 'set' operations. NOTE: This is a write operation that bypasses normal authorization.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{
						"type":        "string",
						"description": "The user ID to update points for",
					},
					"points": map[string]any{
						"type":        "integer",
						"description": "The number of points to add, subtract, or set",
					},
					"operation": map[string]any{
						"type":        "string",
						"description": "Operation: 'add' (default), 'subtract', or 'set'",
						"enum":        []string{"add", "subtract", "set"},
					},
				},
				"required": []string{"user_id", "points"},
			},
			"outputSchema": updateUserPointsOutputSchema(),
		},
		{
			"name":        "list_users",
			"description": "List all users with pagination support.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"limit": map[string]any{
						"type":        "integer",
						"description": "Number of users to return (default: 50)",
					},
					"offset": map[string]any{
						"type":        "integer",
						"description": "Number of users to skip (default: 0)",
					},
				},
			},
			"outputSchema": listUsersOutputSchema(),
		},
		{
			"name":        "get_user_profile",
			"description": "Get detailed profile of a user including points, badges, and recent activity.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"user_id": map[string]any{
						"type":        "string",
						"description": "The user ID to get profile for",
					},
				},
				"required": []string{"user_id"},
			},
			"outputSchema": userProfileOutputSchema(),
		},
		{
			"name":        "get_analytics_summary",
			"description": "Get analytics summary including total users, badges, points, and active rules.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			"outputSchema": analyticsSummaryOutputSchema(),
		},
		{
			"name":        "list_event_types",
			"description": "List all registered event types from the event type registry.",
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
			"outputSchema": listEventTypesOutputSchema(),
		},
	}

	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]any{
			"tools": tools,
		},
	}
}

func (s *MCPServer) handleToolsCall(ctx context.Context, msg jsonRPCMessage) *jsonRPCMessage {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return errorResponse(msg.ID, -32602, "Invalid params")
	}

	var args map[string]any
	if len(params.Arguments) > 0 {
		json.Unmarshal(params.Arguments, &args)
	}

	var result any
	var err error

	switch params.Name {
	case "list_rules":
		eventType := ""
		if et, ok := args["event_type"].(string); ok {
			eventType = et
		}
		result, err = s.backend.ListRules(ctx, eventType)
		if err == nil {
			rules, _ := result.([]backend.Rule)
			return toolSuccessResponse(msg.ID, map[string]any{
				"rules":             rules,
				"count":             len(rules),
				"event_type_filter": eventType,
			})
		}

	case "get_rule":
		ruleID, _ := args["rule_id"].(string)
		if ruleID == "" {
			return errorResponse(msg.ID, -32602, "rule_id is required")
		}
		result, err = s.backend.GetRule(ctx, ruleID)
		if err == nil {
			return toolSuccessResponse(msg.ID, map[string]any{"rule": result})
		}

	case "test_event":
		eventArg, _ := args["event"].(map[string]any)
		if eventArg == nil {
			return errorResponse(msg.ID, -32602, "event is required")
		}

		// Validate event_type is required
		eventType, _ := eventArg["event_type"].(string)
		if eventType == "" {
			return errorResponse(msg.ID, -32602, "validation error: event_type is required")
		}

		// Validate match_id and player_id for non-generic (sports) events using registry
		if !s.isGenericEventType(ctx, eventType) {
			if matchID, _ := eventArg["match_id"].(string); matchID == "" {
				return errorResponse(msg.ID, -32602, fmt.Sprintf("validation error: match_id is required for event type '%s'", eventType))
			}
			if playerID, _ := eventArg["player_id"].(string); playerID == "" {
				return errorResponse(msg.ID, -32602, fmt.Sprintf("validation error: player_id is required for event type '%s'", eventType))
			}
		}

		if s.backend == nil {
			return errorResponse(msg.ID, -32000, "Backend service not available")
		}

		dryRun := true
		if dr, ok := args["dry_run"].(bool); ok {
			dryRun = dr
		}
		result, err = s.backend.TestEvent(ctx, eventArg, dryRun)
		if err != nil {
			errMsg := err.Error()
			// Return -32602 for validation errors, -32000 for backend/runtime errors
			if isValidationError(errMsg) {
				return errorResponse(msg.ID, -32602, errMsg)
			}
			return errorResponse(msg.ID, -32000, errMsg)
		}
		return toolSuccessResponse(msg.ID, result)

	case "assign_badge_to_user":
		userID, _ := args["user_id"].(string)
		badgeID, _ := args["badge_id"].(string)
		reason, _ := args["reason"].(string)
		if userID == "" || badgeID == "" {
			return errorResponse(msg.ID, -32602, "user_id and badge_id are required")
		}
		if reason == "" {
			reason = "Manual badge assignment via MCP"
		}
		err = s.backend.AssignBadgeToUser(ctx, userID, badgeID, reason)
		if err == nil {
			result = map[string]any{
				"success": true,
				"message": fmt.Sprintf("Badge %s assigned to user %s", badgeID, userID),
			}
			return toolSuccessResponse(msg.ID, result)
		}

	case "update_user_points":
		userID, _ := args["user_id"].(string)
		points, pointsOk := args["points"].(float64)
		operation, _ := args["operation"].(string)

		// Validate user_id is required
		if userID == "" {
			return errorResponse(msg.ID, -32602, "validation error: user_id is required")
		}
		// Validate points is provided
		if !pointsOk {
			return errorResponse(msg.ID, -32602, "validation error: points field is required")
		}
		// Validate operation is one of allowed values
		if operation == "" {
			operation = "add"
		} else if operation != "add" && operation != "subtract" && operation != "set" {
			return errorResponse(msg.ID, -32602, "validation error: operation must be 'add', 'subtract', or 'set'")
		}
		newPoints, err := s.backend.UpdateUserPoints(ctx, userID, int(points), operation)
		if err == nil {
			result = map[string]any{
				"success":    true,
				"new_points": newPoints,
			}
			return toolSuccessResponse(msg.ID, result)
		}

	case "list_users":
		limit := 50
		offset := 0
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}
		if o, ok := args["offset"].(float64); ok {
			offset = int(o)
		}
		// Validate pagination parameters
		if limit < 0 {
			return errorResponse(msg.ID, -32602, "validation error: limit cannot be negative")
		}
		if offset < 0 {
			return errorResponse(msg.ID, -32602, "validation error: offset cannot be negative")
		}
		if limit > 100 {
			limit = 100 // Enforce max limit
		}
		result, err = s.backend.ListUsers(ctx, limit, offset)
		if err == nil {
			return toolSuccessResponse(msg.ID, map[string]any{
				"users": result,
				"count": len(result.([]backend.UserInfo)),
			})
		}

	case "get_user_profile":
		userID, _ := args["user_id"].(string)
		if userID == "" {
			return errorResponse(msg.ID, -32602, "user_id is required")
		}
		result, err = s.backend.GetUserProfile(ctx, userID)
		if err == nil {
			return toolSuccessResponse(msg.ID, result)
		}

	case "get_analytics_summary":
		result, err = s.backend.GetAnalyticsSummary(ctx)
		if err == nil {
			return toolSuccessResponse(msg.ID, result)
		}

	case "list_event_types":
		result, err = s.backend.ListEventTypes(ctx)
		if err == nil {
			eventTypes, _ := result.([]map[string]any)
			return toolSuccessResponse(msg.ID, map[string]any{
				"event_types": eventTypes,
				"count":       len(eventTypes),
			})
		}

	default:
		return errorResponse(msg.ID, -32601, fmt.Sprintf("Unknown tool: %s", params.Name))
	}

	if err != nil {
		return errorResponse(msg.ID, -32000, err.Error())
	}

	return toolSuccessResponse(msg.ID, result)
}

func errorResponse(id any, code int, message string) *jsonRPCMessage {
	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &jsonRPCError{
			Code:    code,
			Message: message,
		},
	}
}

// isValidationError checks if the error is a validation error
func isValidationError(err string) bool {
	return strings.HasPrefix(err, "validation error:")
}

// isGenericEventType checks if the event type is a generic (non-sports) event.
// This delegates to the backend service which uses the event type registry.
// Unknown event types are treated as generic (safe default - don't require sport fields).
func (s *MCPServer) isGenericEventType(ctx context.Context, eventType string) bool {
	if s.backend == nil {
		// No backend - assume generic (safe default)
		return true
	}
	// Delegate to backend which queries the registry
	return !s.backend.IsSportEvent(ctx, eventType)
}

func formatJSON(v any) string {
	data, _ := json.MarshalIndent(v, "", "  ")
	return string(data)
}

// ==================== Resources Handlers ====================

func (s *MCPServer) handleResourcesList(msg jsonRPCMessage) *jsonRPCMessage {
	resources := []map[string]any{
		{
			"uri":         "rules://list",
			"name":        "List of all gamification rules",
			"description": "Returns a list of all gamification rules with their configuration",
			"mimeType":    "application/json",
		},
		{
			"uri":         "rules://{id}",
			"name":        "Single rule by ID",
			"description": "Returns detailed information about a specific rule",
			"mimeType":    "application/json",
		},
		{
			"uri":         "analytics://summary",
			"name":        "Analytics summary",
			"description": "Returns analytics summary including total users, badges, points, and active rules",
			"mimeType":    "application/json",
		},
		{
			"uri":         "users://{id}",
			"name":        "User profile by ID",
			"description": "Returns detailed user profile including points, badges, and recent activity",
			"mimeType":    "application/json",
		},
		{
			"uri":         "docs://real-time-badge-flow",
			"name":        "Real-time badge flow documentation",
			"description": "Documentation about the real-time badge awarding flow",
			"mimeType":    "text/markdown",
		},
		{
			"uri":         "openapi://current",
			"name":        "OpenAPI specification",
			"description": "Current OpenAPI specification for the gamification API",
			"mimeType":    "application/json",
		},
	}

	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]any{
			"resources": resources,
		},
	}
}

func (s *MCPServer) handleResourcesRead(ctx context.Context, msg jsonRPCMessage) *jsonRPCMessage {
	var params struct {
		URI string `json:"uri"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return errorResponse(msg.ID, -32602, "Invalid params")
	}

	var result any
	var mimeType string
	var err error

	switch {
	case params.URI == "rules://list":
		result, err = s.backend.ListRules(ctx, "")
		mimeType = "application/json"

	case params.URI == "analytics://summary":
		result, err = s.backend.GetAnalyticsSummary(ctx)
		mimeType = "application/json"

	case len(params.URI) > 7 && params.URI[:7] == "rules://":
		ruleID := params.URI[7:]
		result, err = s.backend.GetRule(ctx, ruleID)
		mimeType = "application/json"

	case len(params.URI) > 7 && params.URI[:7] == "users://":
		userID := params.URI[7:]
		result, err = s.backend.GetUserProfile(ctx, userID)
		mimeType = "application/json"

	case params.URI == "docs://real-time-badge-flow":
		result = resources.RealTimeBadgeFlow
		mimeType = "text/markdown"

	case params.URI == "openapi://current":
		result = resources.OpenAPISpec
		mimeType = "application/json"

	default:
		return errorResponse(msg.ID, -32602, fmt.Sprintf("Unknown resource URI: %s", params.URI))
	}

	if err != nil {
		return errorResponse(msg.ID, -32000, err.Error())
	}

	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]any{
			"contents": []map[string]any{
				{
					"uri":      params.URI,
					"mimeType": mimeType,
					"text":     result,
				},
			},
		},
	}
}

// ==================== Prompts Handlers ====================

func (s *MCPServer) handlePromptsList(msg jsonRPCMessage) *jsonRPCMessage {
	prompts := []map[string]any{
		{
			"name":        "debug-badge-flow",
			"description": "Analyze why a badge was or was not awarded to a user. Use this to debug badge flow issues.",
			"arguments": []map[string]any{
				{"name": "user_id", "description": "The user ID to analyze", "required": true},
				{"name": "badge_id", "description": "The badge ID that should have been awarded", "required": false},
				{"name": "event_id", "description": "The event ID that triggered the badge check", "required": false},
			},
		},
		{
			"name":        "draft-rule-from-text",
			"description": "Generate a rule draft from natural language description. Use this to create new gamification rules.",
			"arguments": []map[string]any{
				{"name": "description", "description": "Natural language description of the rule", "required": true},
			},
		},
		{
			"name":        "analyze-user-state",
			"description": "Analyze and interpret a user's points and badge status. Use this to provide insights about user engagement.",
			"arguments": []map[string]any{
				{"name": "user_id", "description": "The user ID to analyze", "required": true},
			},
		},
	}

	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]any{
			"prompts": prompts,
		},
	}
}

func (s *MCPServer) handlePromptsGet(msg jsonRPCMessage) *jsonRPCMessage {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(msg.Params, &params); err != nil {
		return errorResponse(msg.ID, -32602, "Invalid params")
	}

	var args map[string]any
	if len(params.Arguments) > 0 {
		json.Unmarshal(params.Arguments, &args)
	}

	var text string
	switch params.Name {
	case "debug-badge-flow":
		userID := getStringArg(args, "user_id")
		badgeID := getStringArg(args, "badge_id")
		eventID := getStringArg(args, "event_id")
		text = fmt.Sprintf(`Please analyze why badge %s was or was not awarded to user %s for event %s.

Please check:
1. Does the user have the required badge criteria?
2. Did the event fire correctly?
3. Are there any rule conditions preventing the award?
4. Is there a cooldown or duplicate check blocking the award?`, badgeID, userID, eventID)

	case "draft-rule-from-text":
		desc := getStringArg(args, "description")
		text = fmt.Sprintf(`Please create a gamification rule based on this description:

%s

Please output a JSON rule structure with the following fields:
- id: unique rule identifier
- name: rule name
- description: rule description
- event_type: the event type that triggers this rule
- points: points to award
- conditions: array of conditions (if any)
- rewards: badge rewards (if any)
- enabled: whether the rule is active`, desc)

	case "analyze-user-state":
		userID := getStringArg(args, "user_id")
		text = fmt.Sprintf(`Please analyze the current state of user %s and provide:
1. An assessment of the user's engagement level
2. Suggested next badges they could earn
3. Recommendations for increasing engagement
4. Summary of their gamification journey so far`, userID)

	default:
		return errorResponse(msg.ID, -32601, fmt.Sprintf("Unknown prompt: %s", params.Name))
	}

	return &jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]any{
			"messages": []map[string]any{
				{
					"role": "user",
					"content": map[string]any{
						"type": "text",
						"text": text,
					},
				},
			},
		},
	}
}

func getStringArg(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

// ==================== Streamable HTTP MCP Transport ====================

func (s *MCPServer) newStreamableHTTPHandler() http.Handler {
	remoteServer := s.buildRemoteSDKServer()
	return mcpsdk.NewStreamableHTTPHandler(
		func(_ *http.Request) *mcpsdk.Server { return remoteServer },
		&mcpsdk.StreamableHTTPOptions{
			Stateless:    true,
			JSONResponse: true,
		},
	)
}

func (s *MCPServer) buildRemoteSDKServer() *mcpsdk.Server {
	server := mcpsdk.NewServer(
		&mcpsdk.Implementation{
			Name:    "gamification-mcp",
			Version: version,
		},
		&mcpsdk.ServerOptions{
			Instructions: "Internal trusted gamification MCP server for rules, badges, users, analytics, and event testing.",
		},
	)

	s.registerRemoteTools(server)
	s.registerRemoteResources(server)
	s.registerRemotePrompts(server)

	return server
}

func (s *MCPServer) registerRemoteTools(server *mcpsdk.Server) {
	server.AddTool(&mcpsdk.Tool{
		Name:        "list_rules",
		Description: "Lists active rules from Redis, filtered by event type. Returns only enabled/active rules.",
		InputSchema: schemaObject(map[string]any{
			"event_type": schemaString("Optional filter by event type (e.g., goal, corner, foul)"),
		}),
		OutputSchema: listRulesOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		eventType := getStringArg(args, "event_type")
		rules, err := s.backend.ListRules(ctx, eventType)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(map[string]any{
			"rules":             rules,
			"count":             len(rules),
			"event_type_filter": eventType,
		}), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:        "get_rule",
		Description: "Get detailed information about a specific rule by its ID.",
		InputSchema: schemaObject(map[string]any{
			"rule_id": schemaString("The unique identifier of the rule"),
		}, "rule_id"),
		OutputSchema: getRuleOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		ruleID := getStringArg(args, "rule_id")
		if ruleID == "" {
			return nil, sdkInvalidParamsError("rule_id is required")
		}

		rule, err := s.backend.GetRule(ctx, ruleID)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(map[string]any{"rule": rule}), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:        "test_event",
		Description: "Test how an event would be processed by the rule engine. Use dry_run=true to evaluate without executing actions. For generic events (daily_login, app_shared, etc.), only event_type is required. For sports events, match_id and player_id may be required.",
		InputSchema: schemaObject(map[string]any{
			"event": schemaObject(map[string]any{
				"event_id":   schemaString("Unique identifier for the event. If not provided, a UUID will be auto-generated."),
				"event_type": schemaString("The type of event (e.g., goal, corner, foul, daily_login, app_shared, purchase_completed)"),
				"match_id":   schemaString("The match identifier (optional for generic events)"),
				"team_id":    schemaString("The team identifier (optional)"),
				"player_id":  schemaString("The player identifier (optional for generic events)"),
				"minute":     schemaInteger("Match minute when event occurred (optional)"),
				"metadata": map[string]any{
					"type":        "object",
					"description": "Additional event metadata (optional)",
				},
			}, "event_type"),
			"dry_run": schemaBoolean("If true, only evaluate rules without executing actions (default: true)"),
		}, "event"),
		OutputSchema: testEventOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		eventArg, _ := args["event"].(map[string]any)
		if eventArg == nil {
			return nil, sdkInvalidParamsError("event is required")
		}

		dryRun := true
		if v, ok := args["dry_run"].(bool); ok {
			dryRun = v
		}

		result, err := s.backend.TestEvent(ctx, eventArg, dryRun)
		if err != nil {
			if isValidationError(err.Error()) {
				return nil, sdkInvalidParamsError(err.Error())
			}
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(result), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:        "assign_badge_to_user",
		Description: "Manually assign a badge to a user. NOTE: This is a write operation that bypasses normal authorization.",
		InputSchema: schemaObject(map[string]any{
			"user_id":  schemaString("The user ID to assign the badge to"),
			"badge_id": schemaString("The badge ID to assign"),
			"reason":   schemaString("Reason for the badge assignment (optional)"),
		}, "user_id", "badge_id"),
		OutputSchema: assignBadgeOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		userID := getStringArg(args, "user_id")
		badgeID := getStringArg(args, "badge_id")
		reason := getStringArg(args, "reason")
		if userID == "" || badgeID == "" {
			return nil, sdkInvalidParamsError("user_id and badge_id are required")
		}
		if reason == "" {
			reason = "Manual badge assignment via MCP"
		}

		if err := s.backend.AssignBadgeToUser(ctx, userID, badgeID, reason); err != nil {
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(map[string]any{
			"success": true,
			"message": fmt.Sprintf("Badge %s assigned to user %s", badgeID, userID),
		}), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:        "update_user_points",
		Description: "Add, subtract, or set user points. Supports 'add' (default), 'subtract', and 'set' operations. NOTE: This is a write operation that bypasses normal authorization.",
		InputSchema: schemaObject(map[string]any{
			"user_id": schemaString("The user ID to update points for"),
			"points":  schemaInteger("The number of points to add, subtract, or set"),
			"operation": map[string]any{
				"type":        "string",
				"description": "Operation: 'add' (default), 'subtract', or 'set'",
				"enum":        []string{"add", "subtract", "set"},
			},
		}, "user_id", "points"),
		OutputSchema: updateUserPointsOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		userID := getStringArg(args, "user_id")
		if userID == "" {
			return nil, sdkInvalidParamsError("validation error: user_id is required")
		}

		pointsValue, ok := args["points"].(float64)
		if !ok {
			return nil, sdkInvalidParamsError("validation error: points field is required")
		}

		operation := getStringArg(args, "operation")
		if operation == "" {
			operation = "add"
		}
		if operation != "add" && operation != "subtract" && operation != "set" {
			return nil, sdkInvalidParamsError("validation error: operation must be 'add', 'subtract', or 'set'")
		}

		newPoints, err := s.backend.UpdateUserPoints(ctx, userID, int(pointsValue), operation)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(map[string]any{
			"success":    true,
			"new_points": newPoints,
		}), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:        "list_users",
		Description: "List all users with pagination support.",
		InputSchema: schemaObject(map[string]any{
			"limit":  schemaInteger("Number of users to return (default: 50)"),
			"offset": schemaInteger("Number of users to skip (default: 0)"),
		}),
		OutputSchema: listUsersOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		limit := 50
		offset := 0
		if v, ok := args["limit"].(float64); ok {
			limit = int(v)
		}
		if v, ok := args["offset"].(float64); ok {
			offset = int(v)
		}
		if limit < 0 {
			return nil, sdkInvalidParamsError("validation error: limit cannot be negative")
		}
		if offset < 0 {
			return nil, sdkInvalidParamsError("validation error: offset cannot be negative")
		}
		if limit > 100 {
			limit = 100
		}

		users, err := s.backend.ListUsers(ctx, limit, offset)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(map[string]any{
			"users": users,
			"count": len(users),
		}), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:        "get_user_profile",
		Description: "Get detailed profile of a user including points, badges, and recent activity.",
		InputSchema: schemaObject(map[string]any{
			"user_id": schemaString("The user ID to get profile for"),
		}, "user_id"),
		OutputSchema: userProfileOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		args, err := parseSDKToolArguments(req.Params.Arguments)
		if err != nil {
			return nil, err
		}

		userID := getStringArg(args, "user_id")
		if userID == "" {
			return nil, sdkInvalidParamsError("user_id is required")
		}

		profile, err := s.backend.GetUserProfile(ctx, userID)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}

		return sdkToolSuccessResult(profile), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:         "get_analytics_summary",
		Description:  "Get analytics summary including total users, badges, points, and active rules.",
		InputSchema:  schemaObject(map[string]any{}),
		OutputSchema: analyticsSummaryOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		summary, err := s.backend.GetAnalyticsSummary(ctx)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}
		return sdkToolSuccessResult(summary), nil
	})

	server.AddTool(&mcpsdk.Tool{
		Name:         "list_event_types",
		Description:  "List all registered event types from the event type registry.",
		InputSchema:  schemaObject(map[string]any{}),
		OutputSchema: listEventTypesOutputSchema(),
	}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
		eventTypes, err := s.backend.ListEventTypes(ctx)
		if err != nil {
			return sdkToolErrorResult(err), nil
		}
		return sdkToolSuccessResult(map[string]any{
			"event_types": eventTypes,
			"count":       len(eventTypes),
		}), nil
	})
}

func (s *MCPServer) registerRemoteResources(server *mcpsdk.Server) {
	server.AddResource(&mcpsdk.Resource{
		URI:         "rules://list",
		Name:        "List of all gamification rules",
		Description: "Returns a list of all gamification rules with their configuration",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		rules, err := s.backend.ListRules(ctx, "")
		if err != nil {
			return nil, err
		}
		return sdkJSONResourceResult(req.Params.URI, rules), nil
	})

	server.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		URITemplate: "rules://{id}",
		Name:        "Single rule by ID",
		Description: "Returns detailed information about a specific rule",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		ruleID := strings.TrimPrefix(req.Params.URI, "rules://")
		rule, err := s.backend.GetRule(ctx, ruleID)
		if err != nil {
			return nil, err
		}
		return sdkJSONResourceResult(req.Params.URI, rule), nil
	})

	server.AddResource(&mcpsdk.Resource{
		URI:         "analytics://summary",
		Name:        "Analytics summary",
		Description: "Returns analytics summary including total users, badges, points, and active rules",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		summary, err := s.backend.GetAnalyticsSummary(ctx)
		if err != nil {
			return nil, err
		}
		return sdkJSONResourceResult(req.Params.URI, summary), nil
	})

	server.AddResourceTemplate(&mcpsdk.ResourceTemplate{
		URITemplate: "users://{id}",
		Name:        "User profile by ID",
		Description: "Returns detailed user profile including points, badges, and recent activity",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		userID := strings.TrimPrefix(req.Params.URI, "users://")
		profile, err := s.backend.GetUserProfile(ctx, userID)
		if err != nil {
			return nil, err
		}
		return sdkJSONResourceResult(req.Params.URI, profile), nil
	})

	server.AddResource(&mcpsdk.Resource{
		URI:         "docs://real-time-badge-flow",
		Name:        "Real-time badge flow documentation",
		Description: "Documentation about the real-time badge awarding flow",
		MIMEType:    "text/markdown",
	}, func(_ context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return sdkTextResourceResult(req.Params.URI, "text/markdown", resources.RealTimeBadgeFlow), nil
	})

	server.AddResource(&mcpsdk.Resource{
		URI:         "openapi://current",
		Name:        "OpenAPI specification",
		Description: "Current OpenAPI specification for the gamification API",
		MIMEType:    "application/json",
	}, func(_ context.Context, req *mcpsdk.ReadResourceRequest) (*mcpsdk.ReadResourceResult, error) {
		return sdkTextResourceResult(req.Params.URI, "application/json", resources.OpenAPISpec), nil
	})
}

func (s *MCPServer) registerRemotePrompts(server *mcpsdk.Server) {
	server.AddPrompt(&mcpsdk.Prompt{
		Name:        "debug-badge-flow",
		Description: "Analyze why a badge was or was not awarded to a user. Use this to debug badge flow issues.",
		Arguments: []*mcpsdk.PromptArgument{
			{Name: "user_id", Description: "The user ID to analyze", Required: true},
			{Name: "badge_id", Description: "The badge ID that should have been awarded", Required: false},
			{Name: "event_id", Description: "The event ID that triggered the badge check", Required: false},
		},
	}, func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		userID := req.Params.Arguments["user_id"]
		badgeID := req.Params.Arguments["badge_id"]
		eventID := req.Params.Arguments["event_id"]
		text := fmt.Sprintf(`Please analyze why badge %s was or was not awarded to user %s for event %s.

Please check:
1. Does the user have the required badge criteria?
2. Did the event fire correctly?
3. Are there any rule conditions preventing the award?
4. Is there a cooldown or duplicate check blocking the award?`, badgeID, userID, eventID)

		return &mcpsdk.GetPromptResult{
			Description: "Analyze badge award flow for a specific user and event",
			Messages: []*mcpsdk.PromptMessage{
				{Role: "user", Content: &mcpsdk.TextContent{Text: text}},
			},
		}, nil
	})

	server.AddPrompt(&mcpsdk.Prompt{
		Name:        "draft-rule-from-text",
		Description: "Generate a rule draft from natural language description. Use this to create new gamification rules.",
		Arguments: []*mcpsdk.PromptArgument{
			{Name: "description", Description: "Natural language description of the rule", Required: true},
		},
	}, func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		desc := req.Params.Arguments["description"]
		text := fmt.Sprintf(`Please create a gamification rule based on this description:

%s

Please output a JSON rule structure with the following fields:
- id: unique rule identifier
- name: rule name
- description: rule description
- event_type: the event type that triggers this rule
- points: points to award
- conditions: array of conditions (if any)
- rewards: badge rewards (if any)
- enabled: whether the rule is active`, desc)

		return &mcpsdk.GetPromptResult{
			Description: "Generate a rule draft from a natural-language description",
			Messages: []*mcpsdk.PromptMessage{
				{Role: "user", Content: &mcpsdk.TextContent{Text: text}},
			},
		}, nil
	})

	server.AddPrompt(&mcpsdk.Prompt{
		Name:        "analyze-user-state",
		Description: "Analyze and interpret a user's points and badge status. Use this to provide insights about user engagement.",
		Arguments: []*mcpsdk.PromptArgument{
			{Name: "user_id", Description: "The user ID to analyze", Required: true},
		},
	}, func(_ context.Context, req *mcpsdk.GetPromptRequest) (*mcpsdk.GetPromptResult, error) {
		userID := req.Params.Arguments["user_id"]
		text := fmt.Sprintf(`Please analyze the current state of user %s and provide:
1. An assessment of the user's engagement level
2. Suggested next badges they could earn
3. Recommendations for increasing engagement
4. Summary of their gamification journey so far`, userID)

		return &mcpsdk.GetPromptResult{
			Description: "Analyze a user's gamification state and engagement",
			Messages: []*mcpsdk.PromptMessage{
				{Role: "user", Content: &mcpsdk.TextContent{Text: text}},
			},
		}, nil
	})
}

func parseSDKToolArguments(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, sdkInvalidParamsError("Invalid tool arguments")
	}
	if args == nil {
		args = map[string]any{}
	}
	return args, nil
}

func sdkInvalidParamsError(message string) error {
	return &mcpjsonrpc.Error{
		Code:    mcpjsonrpc.CodeInvalidParams,
		Message: message,
	}
}

func sdkToolSuccessResult(structuredContent any) *mcpsdk.CallToolResult {
	return &mcpsdk.CallToolResult{
		Content: []mcpsdk.Content{
			&mcpsdk.TextContent{Text: formatJSON(structuredContent)},
		},
		StructuredContent: structuredContent,
		IsError:           false,
	}
}

func sdkToolErrorResult(err error) *mcpsdk.CallToolResult {
	result := &mcpsdk.CallToolResult{}
	result.SetError(err)
	return result
}

func sdkJSONResourceResult(uri string, value any) *mcpsdk.ReadResourceResult {
	return sdkTextResourceResult(uri, "application/json", formatJSON(value))
}

func sdkTextResourceResult(uri, mimeType, text string) *mcpsdk.ReadResourceResult {
	return &mcpsdk.ReadResourceResult{
		Contents: []*mcpsdk.ResourceContents{
			{
				URI:      uri,
				MIMEType: mimeType,
				Text:     text,
			},
		},
	}
}
