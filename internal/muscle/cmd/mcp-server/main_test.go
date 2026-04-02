package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gamification/config"
	"gamification/mcp/backend"
	redisclient "gamification/redis"

	"github.com/alicebob/miniredis/v2"
	redigo "github.com/redis/go-redis/v9"
)

// TestInitializeRequest tests that initialize request returns proper response
func TestInitializeRequest(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	}

	resp := server.handleInitialize(msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("Expected ID 1, got %v", resp.ID)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	if result["protocolVersion"] == nil {
		t.Error("Expected protocolVersion in result")
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("Expected serverInfo in result")
	}

	if serverInfo["name"] != "gamification-mcp" {
		t.Errorf("Expected server name gamification-mcp, got %v", serverInfo["name"])
	}
}

func TestStreamableHTTPInitialize(t *testing.T) {
	server := &MCPServer{backend: nil}
	httpServer := httptest.NewServer(server.newStreamableHTTPHandler())
	defer httpServer.Close()

	req, err := http.NewRequest("POST", httpServer.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call streamable HTTP handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	result, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload, got %v", body)
	}

	if result["protocolVersion"] == nil {
		t.Fatalf("expected protocolVersion in initialize response")
	}
}

func TestStreamableHTTPToolsList(t *testing.T) {
	server := &MCPServer{backend: nil}
	httpServer := httptest.NewServer(server.newStreamableHTTPHandler())
	defer httpServer.Close()

	req, err := http.NewRequest("POST", httpServer.URL, strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to call streamable HTTP handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	result, ok := body["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result payload, got %v", body)
	}

	tools, ok := result["tools"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatalf("expected non-empty tools list, got %v", result["tools"])
	}
}

// TestNotificationsInitialized tests that notifications/initialized returns no response
func TestNotificationsInitialized(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	}

	// This is a notification (no ID), should return nil (no response)
	resp := server.processMessage(nil, msg)

	if resp != nil {
		t.Errorf("Expected no response for notification, got %v", resp)
	}
}

// TestUnknownNotification tests that unknown notification returns no response
func TestUnknownNotification(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		Method:  "notifications/unknown",
	}

	resp := server.processMessage(nil, msg)

	if resp != nil {
		t.Errorf("Expected no response for unknown notification, got %v", resp)
	}
}

// TestUnknownMethod tests that unknown method returns -32601 error
func TestUnknownMethod(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "unknown/method",
	}

	resp := server.processMessage(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}

	if resp.Error.Message != "Method not found" {
		t.Errorf("Expected 'Method not found', got %s", resp.Error.Message)
	}
}

// TestUpdateUserPointsMissingPoints tests that missing points returns -32602 error
func TestUpdateUserPointsMissingPoints(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"update_user_points","arguments":{"user_id":"user123"}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}
}

// TestUpdateUserPointsInvalidOperation tests that invalid operation returns -32602 error
func TestUpdateUserPointsInvalidOperation(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"update_user_points","arguments":{"user_id":"user123","points":100,"operation":"invalid"}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}
}

// TestListUsersNegativeLimit tests that negative limit returns -32602 error
func TestListUsersNegativeLimit(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_users","arguments":{"limit":-1}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}
}

// TestListUsersNegativeOffset tests that negative offset returns -32602 error
func TestListUsersNegativeOffset(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"list_users","arguments":{"offset":-1}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}
}

// TestTestEventMissingEventType tests that missing event_type returns validation error
func TestTestEventMissingEventType(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"match_id":"match123","player_id":"player456"}}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	// Should return -32602 for validation errors
	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
	}

	if resp.Error.Message != "validation error: event_type is required" {
		t.Errorf("Expected 'validation error: event_type is required', got %s", resp.Error.Message)
	}
}

// TestTestEventMissingMatchID tests that missing match_id returns validation error for sports events
// Note: With nil backend, we can't determine event type from registry, so we default to generic (safe assumption).
// The test expects either validation error (if backend available) or backend not available error (if nil backend).
func TestTestEventMissingMatchID(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"goal","player_id":"player456"}}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	// With nil backend, we can't query registry to know if "goal" is sport.
	// We default to generic (safe assumption), so no validation error about match_id.
	// Instead we get "Backend service not available" error.
	// This test verifies the behavior with nil backend.
	if resp.Error.Code == -32000 && strings.Contains(resp.Error.Message, "Backend service not available") {
		// Expected behavior with nil backend - generic event assumed, backend not available
		return
	}

	// If we had a backend, we'd expect validation error
	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602 or -32000, got %d", resp.Error.Code)
	}
}

// TestTestEventMissingPlayerID tests that missing player_id returns validation error for sports events
// Note: With nil backend, we can't determine event type from registry, so we default to generic (safe assumption).
func TestTestEventMissingPlayerID(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"goal","match_id":"match123"}}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("Expected error in response")
	}

	// With nil backend, we can't query registry to know if "goal" is sport.
	// We default to generic (safe assumption), so no validation error about player_id.
	if resp.Error.Code == -32000 && strings.Contains(resp.Error.Message, "Backend service not available") {
		// Expected behavior with nil backend
		return
	}

	if resp.Error.Code != -32602 {
		t.Errorf("Expected error code -32602 or -32000, got %d", resp.Error.Code)
	}
}

// TestTestEventGenericEventDailyLogin tests that daily_login (generic event) doesn't require match_id or player_id
func TestTestEventGenericEventDailyLogin(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"daily_login"}}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Should NOT return a validation error about missing match_id or player_id
	// May return other errors (e.g., rule engine not available), but not validation errors
	if resp.Error != nil && resp.Error.Code == -32602 {
		if strings.Contains(resp.Error.Message, "match_id") || strings.Contains(resp.Error.Message, "player_id") {
			t.Errorf("Generic event daily_login should not require match_id or player_id, got: %s", resp.Error.Message)
		}
	}
}

// TestTestEventGenericEventAppShared tests that app_shared (generic event) doesn't require match_id or player_id
func TestTestEventGenericEventAppShared(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"app_shared"}}}`),
	}

	resp := server.handleToolsCall(nil, msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Should NOT return a validation error about missing match_id or player_id
	if resp.Error != nil && resp.Error.Code == -32602 {
		if strings.Contains(resp.Error.Message, "match_id") || strings.Contains(resp.Error.Message, "player_id") {
			t.Errorf("Generic event app_shared should not require match_id or player_id, got: %s", resp.Error.Message)
		}
	}
}

// TestResourcesReadRealTimeBadgeFlow tests that docs://real-time-badge-flow returns correct mimeType
func TestResourcesReadRealTimeBadgeFlow(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri":"docs://real-time-badge-flow"}`),
	}

	resp := server.handleResourcesRead(nil, msg)

	// Fail-fast: check response is not nil
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Fail-fast: check for error
	if resp.Error != nil {
		t.Fatalf("Expected no error, got: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	contents, ok := result["contents"].([]map[string]any)
	if !ok || len(contents) == 0 {
		t.Fatal("Expected contents array")
	}

	mimeType, ok := contents[0]["mimeType"].(string)
	if !ok {
		t.Fatal("Expected mimeType in contents")
	}

	if mimeType != "text/markdown" {
		t.Errorf("Expected mimeType text/markdown, got %s", mimeType)
	}

	// Verify content contains expected header
	text, ok := contents[0]["text"].(string)
	if !ok || text == "" {
		t.Error("Expected non-empty text content")
	}

	if !strings.Contains(text, "# Real-time Badge") {
		t.Error("Expected content to contain '# Real-time Badge' header")
	}
}

// TestResourcesReadOpenAPI tests that openapi://current returns correct mimeType
func TestResourcesReadOpenAPI(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/read",
		Params:  json.RawMessage(`{"uri":"openapi://current"}`),
	}

	resp := server.handleResourcesRead(nil, msg)

	// Fail-fast: check response is not nil
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Fail-fast: check for error
	if resp.Error != nil {
		t.Fatalf("Expected no error, got: %s", resp.Error.Message)
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	contents, ok := result["contents"].([]map[string]any)
	if !ok || len(contents) == 0 {
		t.Fatal("Expected contents array")
	}

	mimeType, ok := contents[0]["mimeType"].(string)
	if !ok {
		t.Fatal("Expected mimeType in contents")
	}

	if mimeType != "application/json" {
		t.Errorf("Expected mimeType application/json, got %s", mimeType)
	}

	// Verify content contains OpenAPI/swagger
	text, ok := contents[0]["text"].(string)
	if !ok || text == "" {
		t.Error("Expected non-empty text content")
	}

	lowerText := strings.ToLower(text)
	if !strings.Contains(lowerText, "swagger") && !strings.Contains(lowerText, "openapi") {
		t.Error("Expected content to contain 'swagger' or 'openapi' field")
	}
}

// TestToolsList tests that tools/list returns tool list
func TestToolsList(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := server.handleToolsList(msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	tools, ok := result["tools"].([]map[string]any)
	if !ok {
		t.Fatal("Expected tools in result")
	}

	if len(tools) == 0 {
		t.Error("Expected non-empty tools list")
	}
}

// TestToolsListIncludesOutputSchema tests that all tools expose outputSchema
func TestToolsListIncludesOutputSchema(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := server.handleToolsList(msg)
	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	tools, ok := result["tools"].([]map[string]any)
	if !ok {
		t.Fatal("Expected tools in result")
	}

	if len(tools) == 0 {
		t.Fatal("Expected non-empty tools list")
	}

	for _, tool := range tools {
		name, _ := tool["name"].(string)
		outputSchema, ok := tool["outputSchema"].(map[string]any)
		if !ok {
			t.Fatalf("Expected outputSchema for tool %s", name)
		}
		if outputSchema["type"] != "object" {
			t.Fatalf("Expected outputSchema.type=object for tool %s, got %v", name, outputSchema["type"])
		}
		if _, ok := outputSchema["properties"].(map[string]any); !ok {
			t.Fatalf("Expected outputSchema.properties for tool %s", name)
		}
	}
}

// TestToolSuccessResponseIncludesStructuredContent tests the MCP result envelope for successful tools
func TestToolSuccessResponseIncludesStructuredContent(t *testing.T) {
	resp := toolSuccessResponse(1, map[string]any{
		"success": true,
		"count":   2,
	})

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	isError, ok := result["isError"].(bool)
	if !ok {
		t.Fatal("Expected isError flag in result")
	}
	if isError {
		t.Fatal("Expected isError=false for successful tool result")
	}

	structured, ok := result["structuredContent"].(map[string]any)
	if !ok {
		t.Fatal("Expected structuredContent in result")
	}
	if structured["success"] != true {
		t.Fatalf("Expected structuredContent.success=true, got %v", structured["success"])
	}

	content, ok := result["content"].([]map[string]any)
	if !ok || len(content) != 1 {
		t.Fatal("Expected a single text content block")
	}
	if content[0]["type"] != "text" {
		t.Fatalf("Expected text content block, got %v", content[0]["type"])
	}
	text, _ := content[0]["text"].(string)
	if !strings.Contains(text, "\"success\": true") {
		t.Fatalf("Expected serialized structured content in text block, got %s", text)
	}
}

// TestResourcesList tests that resources/list returns resource list
func TestResourcesList(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "resources/list",
	}

	resp := server.handleResourcesList(msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	resources, ok := result["resources"].([]map[string]any)
	if !ok {
		t.Fatal("Expected resources in result")
	}

	if len(resources) == 0 {
		t.Error("Expected non-empty resources list")
	}
}

// TestPromptsList tests that prompts/list returns prompt list
func TestPromptsList(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "prompts/list",
	}

	resp := server.handlePromptsList(msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	prompts, ok := result["prompts"].([]map[string]any)
	if !ok {
		t.Fatal("Expected prompts in result")
	}

	if len(prompts) == 0 {
		t.Error("Expected non-empty prompts list")
	}
}

// TestTestEventSchemaProperties tests that test_event tool has correct schema
func TestTestEventSchemaProperties(t *testing.T) {
	server := &MCPServer{backend: nil}

	msg := jsonRPCMessage{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}

	resp := server.handleToolsList(msg)

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	result, ok := resp.Result.(map[string]any)
	if !ok {
		t.Fatal("Expected result to be map[string]any")
	}

	// Serialize and deserialize to get proper []any type
	resultJSON, _ := json.Marshal(result)
	var parsedResult map[string]any
	json.Unmarshal(resultJSON, &parsedResult)

	toolsAny, ok := parsedResult["tools"].([]any)
	if !ok {
		t.Fatal("Expected tools in result")
	}

	// Find test_event tool
	var testEventTool map[string]any
	for _, toolVal := range toolsAny {
		tm, ok := toolVal.(map[string]any)
		if !ok {
			continue
		}
		if tm["name"] == "test_event" {
			testEventTool = tm
			break
		}
	}

	if testEventTool == nil {
		t.Fatal("test_event tool not found")
	}

	// Check input schema
	inputSchema, ok := testEventTool["inputSchema"].(map[string]any)
	if !ok {
		t.Fatal("Expected inputSchema")
	}

	eventProps, ok := inputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties in inputSchema")
	}

	event, ok := eventProps["event"].(map[string]any)
	if !ok {
		t.Fatal("Expected event property")
	}

	_, ok = event["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected event properties")
	}

	// Check event_id is optional (not in required)
	required, ok := event["required"].([]any)
	if ok {
		for _, r := range required {
			if r == "event_id" {
				t.Error("event_id should not be in required list")
			}
		}
	}

	// Check event_type is required (match_id and player_id are optional - validated at runtime for sports events)
	requiredMap := make(map[string]bool)
	for _, r := range required {
		requiredMap[r.(string)] = true
	}

	if !requiredMap["event_type"] {
		t.Error("event_type should be required")
	}
	// match_id and player_id are NOT required in schema - they're validated at runtime for sports events
	// This allows generic events (daily_login, etc.) to work without these fields
}

// testRedisClient creates a Redis client for testing from miniredis
func testRedisClient(mr *miniredis.Miniredis) *redisclient.Client {
	rdb := redigo.NewClient(&redigo.Options{
		Addr: mr.Addr(),
	})
	return redisclient.NewTestClient(rdb, &config.RedisConfig{Host: "localhost", Port: 6379})
}

// TestTestEventRegistryBackedValidation tests that validation uses the real Redis registry
// to determine whether an event type requires sport-specific fields (match_id, player_id)
func TestTestEventRegistryBackedValidation(t *testing.T) {
	// Start miniredis
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	// Create real Redis client
	rClient := testRedisClient(mr)
	ctx := context.Background()

	// Seed sport event type
	sportEvent := &redisclient.EventType{
		Key:         "goal",
		Name:        "Goal",
		Description: "A goal scored",
		Category:    "sport",
		Enabled:     true,
	}
	_, err = rClient.CreateEventType(ctx, sportEvent)
	if err != nil {
		t.Fatalf("Failed to create sport event type: %v", err)
	}

	// Seed custom event type
	customEvent := &redisclient.EventType{
		Key:         "purchase_completed",
		Name:        "Purchase Completed",
		Description: "A purchase was completed",
		Category:    "custom",
		Enabled:     true,
	}
	_, err = rClient.CreateEventType(ctx, customEvent)
	if err != nil {
		t.Fatalf("Failed to create custom event type: %v", err)
	}

	// Seed engagement event type
	engagementEvent := &redisclient.EventType{
		Key:         "daily_login",
		Name:        "Daily Login",
		Description: "User logged in daily",
		Category:    "engagement",
		Enabled:     true,
	}
	_, err = rClient.CreateEventType(ctx, engagementEvent)
	if err != nil {
		t.Fatalf("Failed to create engagement event type: %v", err)
	}

	// Create backend service with real Redis client
	backendService := backend.NewService(rClient, nil)

	// Create MCP server with real backend
	server := NewMCPServer(backendService)

	// Test 1: Sport event (goal) WITHOUT match_id should return -32602 validation error
	t.Run("sport event without match_id returns validation error", func(t *testing.T) {
		msg := jsonRPCMessage{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"goal","player_id":"player456"}}}`),
		}

		resp := server.handleToolsCall(ctx, msg)

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("Expected error in response")
		}

		if resp.Error.Code != -32602 {
			t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
		}

		if !strings.Contains(resp.Error.Message, "match_id is required") {
			t.Errorf("Expected 'match_id is required' error, got: %s", resp.Error.Message)
		}
	})

	// Test 2: Sport event (goal) WITHOUT player_id should return -32602 validation error
	t.Run("sport event without player_id returns validation error", func(t *testing.T) {
		msg := jsonRPCMessage{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"goal","match_id":"match123"}}}`),
		}

		resp := server.handleToolsCall(ctx, msg)

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("Expected error in response")
		}

		if resp.Error.Code != -32602 {
			t.Errorf("Expected error code -32602, got %d", resp.Error.Code)
		}

		if !strings.Contains(resp.Error.Message, "player_id is required") {
			t.Errorf("Expected 'player_id is required' error, got: %s", resp.Error.Message)
		}
	})

	// Test 3: Custom event (purchase_completed) WITHOUT match_id should NOT return validation error
	t.Run("custom event without match_id passes validation", func(t *testing.T) {
		msg := jsonRPCMessage{
			JSONRPC: "2.0",
			ID:      3,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"purchase_completed","subject_id":"user123"}}}`),
		}

		resp := server.handleToolsCall(ctx, msg)

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Should NOT get validation error about match_id/player_id
		if resp.Error != nil && resp.Error.Code == -32602 {
			if strings.Contains(resp.Error.Message, "match_id") || strings.Contains(resp.Error.Message, "player_id") {
				t.Errorf("Custom event should NOT require match_id/player_id, got: %s", resp.Error.Message)
			}
		}
	})

	// Test 4: Engagement event (daily_login) WITHOUT match_id should NOT return validation error
	t.Run("engagement event without match_id passes validation", func(t *testing.T) {
		msg := jsonRPCMessage{
			JSONRPC: "2.0",
			ID:      4,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"daily_login","subject_id":"user456"}}}`),
		}

		resp := server.handleToolsCall(ctx, msg)

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Should NOT get validation error about match_id/player_id
		if resp.Error != nil && resp.Error.Code == -32602 {
			if strings.Contains(resp.Error.Message, "match_id") || strings.Contains(resp.Error.Message, "player_id") {
				t.Errorf("Engagement event should NOT require match_id/player_id, got: %s", resp.Error.Message)
			}
		}
	})

	// Test 5: Unknown event type should NOT require match_id/player_id (defaults to generic)
	t.Run("unknown event type defaults to generic and passes validation", func(t *testing.T) {
		msg := jsonRPCMessage{
			JSONRPC: "2.0",
			ID:      5,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"completely_unknown_event_type"}}}`),
		}

		resp := server.handleToolsCall(ctx, msg)

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Should NOT get validation error about match_id/player_id
		if resp.Error != nil && resp.Error.Code == -32602 {
			if strings.Contains(resp.Error.Message, "match_id") || strings.Contains(resp.Error.Message, "player_id") {
				t.Errorf("Unknown event type should default to generic (no validation), got: %s", resp.Error.Message)
			}
		}
	})

	// Test 6: Add a new event type at runtime and verify it behaves as generic
	t.Run("dynamically added event type is treated as generic", func(t *testing.T) {
		// Add new event type with custom category
		newEvent := &redisclient.EventType{
			Key:         "new_custom_event",
			Name:        "New Custom Event",
			Description: "A new custom event",
			Category:    "custom",
			Enabled:     true,
		}
		_, err = rClient.CreateEventType(ctx, newEvent)
		if err != nil {
			t.Fatalf("Failed to create new event type: %v", err)
		}

		msg := jsonRPCMessage{
			JSONRPC: "2.0",
			ID:      6,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_event","arguments":{"event":{"event_type":"new_custom_event"}}}`),
		}

		resp := server.handleToolsCall(ctx, msg)

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Should NOT get validation error about match_id/player_id
		if resp.Error != nil && resp.Error.Code == -32602 {
			if strings.Contains(resp.Error.Message, "match_id") || strings.Contains(resp.Error.Message, "player_id") {
				t.Errorf("New custom event should be treated as generic, got: %s", resp.Error.Message)
			}
		}
	})
}
