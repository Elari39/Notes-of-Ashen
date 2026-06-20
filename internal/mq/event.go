package mq

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"notes-of-ashen/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"
)

type Event struct {
	UserID       uint64            `json:"userId,omitempty"`
	EventType    string            `json:"eventType"`
	ResourceType string            `json:"resourceType,omitempty"`
	ResourceID   uint64            `json:"resourceId,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IP           string            `json:"ip,omitempty"`
	UserAgent    string            `json:"userAgent,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
}

type Publisher struct {
	conf config.RabbitMQConf
	db   *sql.DB
	mu   sync.Mutex
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(conf config.RabbitMQConf, db *sql.DB) *Publisher {
	p := &Publisher{conf: conf, db: db}
	if !conf.Enabled {
		return p
	}
	if err := p.connect(); err != nil {
		logx.Errorf("rabbitmq connect failed: %v", err)
	}
	return p
}

func (p *Publisher) connect() error {
	conn, err := amqp.Dial(p.conf.URL)
	if err != nil {
		return err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return err
	}
	if err := ch.ExchangeDeclare(p.conf.Exchange, "direct", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	if _, err := ch.QueueDeclare(p.conf.Queue, true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	if err := ch.QueueBind(p.conf.Queue, p.conf.RoutingKey, p.conf.Exchange, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return err
	}
	p.conn = conn
	p.ch = ch
	return nil
}

func (p *Publisher) Publish(ctx context.Context, event Event) {
	if !p.conf.Enabled {
		// MQ 未启用时，audit / 操作事件必须同步落库，否则 /admin/logs 永远为空。
		if p.db == nil {
			logx.Errorf("operation log dropped: rabbitmq disabled and db writer unavailable")
			return
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = time.Now()
		}
		if err := writeOperationLogEvent(p.db, event); err != nil {
			logx.Errorf("write operation log directly failed: %v", err)
		}
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	body, err := json.Marshal(event)
	if err != nil {
		logx.Errorf("marshal event failed: %v", err)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.ch == nil {
		if err := p.connect(); err != nil {
			logx.Errorf("rabbitmq reconnect failed: %v", err)
			return
		}
	}
	if err := p.publish(ctx, body); err != nil {
		logx.Errorf("publish event failed: %v", err)
		p.resetConnection()
	}
}

func (p *Publisher) publish(ctx context.Context, body []byte) error {
	return p.ch.PublishWithContext(ctx, p.conf.Exchange, p.conf.RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Publisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resetConnection()
}

func (p *Publisher) resetConnection() {
	if p.ch != nil {
		_ = p.ch.Close()
		p.ch = nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}

func StartConsumer(conf config.RabbitMQConf, db *sql.DB) {
	if !conf.Enabled {
		return
	}
	go func() {
		backoff := time.Second
		for {
			if err := consumeOperationLogs(conf, db); err != nil {
				logx.Errorf("rabbitmq consumer stopped: %v", err)
			}
			time.Sleep(backoff)
			backoff = nextConsumerBackoff(backoff)
		}
	}()
}

func nextConsumerBackoff(current time.Duration) time.Duration {
	if current <= 0 {
		current = time.Second
	}
	next := current * 2
	if next > 30*time.Second {
		next = 30 * time.Second
	}
	// 加随机抖动 [0.5*next, 1.5*next)，避免多 consumer 在 broker 恢复后同时重连形成惊群。
	jitter := time.Duration(rand.Int63n(int64(next)))
	return next/2 + jitter
}

func consumeOperationLogs(conf config.RabbitMQConf, db *sql.DB) error {
	conn, err := amqp.Dial(conf.URL)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("channel failed: %w", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare(conf.Exchange, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("exchange declare failed: %w", err)
	}
	if _, err := ch.QueueDeclare(conf.Queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("queue declare failed: %w", err)
	}
	if err := ch.QueueBind(conf.Queue, conf.RoutingKey, conf.Exchange, false, nil); err != nil {
		return fmt.Errorf("queue bind failed: %w", err)
	}

	msgs, err := ch.Consume(conf.Queue, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume failed: %w", err)
	}

	for msg := range msgs {
		writeOperationLogMessage(db, msg)
	}
	return fmt.Errorf("delivery channel closed")
}

func writeOperationLogMessage(db *sql.DB, msg amqp.Delivery) {
	var event Event
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		logx.Errorf("unmarshal operation event failed: %v", err)
		_ = msg.Nack(false, false)
		return
	}
	if err := writeOperationLogEvent(db, event); err != nil {
		logx.Errorf("write operation log failed: %v", err)
		_ = msg.Nack(false, true)
		return
	}
	_ = msg.Ack(false)
}

// writeOperationLogEvent 将单个操作事件写入 operation_logs 表。
// 消费端（MQ 启用）与禁用兜底（MQ 关闭）共用此函数，确保审计日志在任何情况下都不丢失。
func writeOperationLogEvent(db *sql.DB, event Event) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	metadata, _ := json.Marshal(event.Metadata)
	var metadataValue interface{}
	if len(event.Metadata) > 0 {
		metadataValue = string(metadata)
	}
	var userID interface{}
	if event.UserID > 0 {
		userID = event.UserID
	}
	var resourceID interface{}
	if event.ResourceID > 0 {
		resourceID = event.ResourceID
	}
	_, err := db.Exec(`
INSERT INTO operation_logs (user_id, event_type, resource_type, resource_id, metadata, ip, user_agent, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		userID, event.EventType, event.ResourceType, resourceID, metadataValue, event.IP, event.UserAgent, event.CreatedAt)
	return err
}
