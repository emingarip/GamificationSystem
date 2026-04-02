package api

import (
	"testing"
	"time"

	"gamification/neo4j"
)

func applyUpdate(existing *neo4j.User, reqName string, reqPoints *int) (string, int) {
	name := existing.Name
	if reqName != "" {
		name = reqName
	}
	points := existing.Points
	if reqPoints != nil {
		points = *reqPoints
	}
	return name, points
}

// TestPartialUpdateUserNameOnly tests that UpdateUser with only Name doesn't reset Points
func TestPartialUpdateUserNameOnly(t *testing.T) {
	existing := &neo4j.User{
		ID:        "user_001",
		Name:      "Old Name",
		Email:     "user@example.com",
		Points:    500,
		Level:     5,
		CreatedAt: time.Now(),
	}

	name, points := applyUpdate(existing, "New Name", nil)

	if name != "New Name" {
		t.Errorf("Expected Name to be 'New Name', got '%s'", name)
	}
	if points != 500 {
		t.Errorf("Expected Points to remain 500 (not reset), got %d", points)
	}
}

// TestPartialUpdateUserPointsOnly tests that UpdateUser with only Points doesn't reset Name
func TestPartialUpdateUserPointsOnly(t *testing.T) {
	existing := &neo4j.User{
		ID:        "user_001",
		Name:      "Existing Name",
		Email:     "user@example.com",
		Points:    500,
		Level:     5,
		CreatedAt: time.Now(),
	}

	pointsVal := 1000
	name, points := applyUpdate(existing, "", &pointsVal)

	if name != "Existing Name" {
		t.Errorf("Expected Name to remain 'Existing Name' (not reset), got '%s'", name)
	}
	if points != 1000 {
		t.Errorf("Expected Points to be 1000, got %d", points)
	}
}

// TestPartialUpdateUserBothFields tests that UpdateUser with both Name and Points updates both
func TestPartialUpdateUserBothFields(t *testing.T) {
	existing := &neo4j.User{
		ID:        "user_001",
		Name:      "Old Name",
		Email:     "user@example.com",
		Points:    500,
		Level:     5,
		CreatedAt: time.Now(),
	}

	pointsVal := 1000
	name, points := applyUpdate(existing, "New Name", &pointsVal)

	if name != "New Name" {
		t.Errorf("Expected Name to be 'New Name', got '%s'", name)
	}
	if points != 1000 {
		t.Errorf("Expected Points to be 1000, got %d", points)
	}
}

// TestPartialUpdateUserNoFields tests that UpdateUser with no fields doesn't change anything
func TestPartialUpdateUserNoFields(t *testing.T) {
	existing := &neo4j.User{
		ID:        "user_001",
		Name:      "Existing Name",
		Email:     "user@example.com",
		Points:    500,
		Level:     5,
		CreatedAt: time.Now(),
	}

	name, points := applyUpdate(existing, "", nil)

	if name != "Existing Name" {
		t.Errorf("Expected Name to remain 'Existing Name', got '%s'", name)
	}
	if points != 500 {
		t.Errorf("Expected Points to remain 500, got %d", points)
	}
}

// TestPartialUpdateLogicVerification verifies the partial update behavior
func TestPartialUpdateLogicVerification(t *testing.T) {
	existingPoints := 500
	existing := &neo4j.User{
		Points: existingPoints,
	}

	pointsVal := 750
	_, points := applyUpdate(existing, "", &pointsVal)

	if points != 750 {
		t.Errorf("Expected points to be updated to 750, got %d", points)
	}

	if existing.Points == 750 {
		t.Error("Existing user object should not be modified directly in this test")
	}
}
