package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"gamification/config"
	"gamification/models"
	"github.com/segmentio/kafka-go"
)

// Consumer wraps the Kafka consumer
type Consumer struct {
	reader   *kafka.Reader
	config   *config.KafkaConfig
	handlers []EventHandler
	// Badge handler for WebSocket broadcasting
	badgeHandler BadgeHandler
}

// EventHandler is a function that processes match events
type EventHandler func(ctx context.Context, event *models.MatchEvent) error

// BadgeHandler is a function that handles badge earned events
type BadgeHandler func(userID, badgeID, badgeName, description string, points int)

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.KafkaConfig) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		GroupID:        cfg.GroupID,
		Topic:          cfg.Topic,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		MaxWait:        cfg.MaxWait,
		StartOffset:    getStartOffset(cfg.StartOffset),
		CommitInterval: cfg.CommitInterval,
	})

	return &Consumer{
		reader: reader,
		config: cfg,
	}
}

// getStartOffset converts string offset to kafka offset
func getStartOffset(offset string) int64 {
	switch offset {
	case "earliest":
		return kafka.FirstOffset
	case "latest":
		return kafka.LastOffset
	default:
		return kafka.FirstOffset
	}
}

// RegisterHandler registers an event handler
func (c *Consumer) RegisterHandler(handler EventHandler) {
	c.handlers = append(c.handlers, handler)
}

// RegisterBadgeHandler registers a handler for badge earned events
func (c *Consumer) RegisterBadgeHandler(handler BadgeHandler) {
	c.badgeHandler = handler
}

// Start begins consuming messages
func (c *Consumer) Start(ctx context.Context) error {
	log.Println("Starting Kafka consumer...")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			if err := c.processMessage(ctx, msg); err != nil {
				log.Printf("Error processing message: %v", err)
			}
		}
	}
}

// processMessage processes a single Kafka message
func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) error {
	var event models.MatchEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("failed to unmarshal event: %w", err)
	}

	// Execute all handlers
	for _, handler := range c.handlers {
		if err := handler(ctx, &event); err != nil {
			log.Printf("Error in handler: %v", err)
		}
	}

	return nil
}

// PublishBadgeEarned publishes a badge earned event to the WebSocket handler
func (c *Consumer) PublishBadgeEarned(userID, badgeID, badgeName, description string, points int) {
	if c.badgeHandler != nil {
		c.badgeHandler(userID, badgeID, badgeName, description, points)
	}
}

// Close closes the Kafka consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}

// Producer for sending events to downstream systems
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg *config.KafkaConfig) *Producer {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Brokers,
		Balancer: &kafka.LeastBytes{},
	})
	return &Producer{writer: writer}
}

// Send sends a message to a topic
func (p *Producer) Send(ctx context.Context, topic string, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
	})
}

// SendEvent sends a MatchEvent to a topic
func (p *Producer) SendEvent(ctx context.Context, topic string, event *models.MatchEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return p.Send(ctx, topic, []byte(event.EventID), data)
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	return p.writer.Close()
}

// EventPublisher is an interface for publishing rule engine results
type EventPublisher interface {
	PublishRuleTriggered(ctx context.Context, result *models.RuleEngineResult) error
	PublishUserAction(ctx context.Context, userID, actionType, matchID string) error
}

// DefaultProducer wraps the Kafka producer for rule engine results
type DefaultProducer struct {
	producer *Producer
}

// NewDefaultProducer creates a new default producer
func NewDefaultProducer(cfg *config.KafkaConfig) *DefaultProducer {
	return &DefaultProducer{
		producer: NewProducer(cfg),
	}
}

// PublishRuleTriggered publishes a rule triggered event
func (p *DefaultProducer) PublishRuleTriggered(ctx context.Context, result *models.RuleEngineResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	return p.producer.Send(ctx, "rule-triggered", []byte(result.Event.EventID), data)
}

// PublishUserAction publishes a user action event
func (p *DefaultProducer) PublishUserAction(ctx context.Context, userID, actionType, matchID string) error {
	data := map[string]string{
		"user_id":     userID,
		"action_type": actionType,
		"match_id":    matchID,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}
	payload, _ := json.Marshal(data)
	return p.producer.Send(ctx, "user-actions", []byte(userID), payload)
}

// Close closes the producer
func (p *DefaultProducer) Close() error {
	return p.producer.Close()
}
