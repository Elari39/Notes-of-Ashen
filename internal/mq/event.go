package mq

import (
	"context"
	"database/sql"
	"encoding/json"
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
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(conf config.RabbitMQConf) *Publisher {
	p := &Publisher{conf: conf}
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
	if p.ch == nil {
		if err := p.connect(); err != nil {
			logx.Errorf("rabbitmq reconnect failed: %v", err)
			return
		}
	}
	err = p.ch.PublishWithContext(ctx, p.conf.Exchange, p.conf.RoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
	if err != nil {
		logx.Errorf("publish event failed: %v", err)
	}
}

func (p *Publisher) Close() {
	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

func StartConsumer(conf config.RabbitMQConf, db *sql.DB) {
	if !conf.Enabled {
		return
	}
	go func() {
		conn, err := amqp.Dial(conf.URL)
		if err != nil {
			logx.Errorf("rabbitmq consumer connect failed: %v", err)
			return
		}
		defer conn.Close()

		ch, err := conn.Channel()
		if err != nil {
			logx.Errorf("rabbitmq consumer channel failed: %v", err)
			return
		}
		defer ch.Close()

		if err := ch.ExchangeDeclare(conf.Exchange, "direct", true, false, false, false, nil); err != nil {
			logx.Errorf("rabbitmq consumer exchange failed: %v", err)
			return
		}
		if _, err := ch.QueueDeclare(conf.Queue, true, false, false, false, nil); err != nil {
			logx.Errorf("rabbitmq consumer queue failed: %v", err)
			return
		}
		if err := ch.QueueBind(conf.Queue, conf.RoutingKey, conf.Exchange, false, nil); err != nil {
			logx.Errorf("rabbitmq consumer bind failed: %v", err)
			return
		}

		msgs, err := ch.Consume(conf.Queue, "", false, false, false, false, nil)
		if err != nil {
			logx.Errorf("rabbitmq consumer consume failed: %v", err)
			return
		}

		for msg := range msgs {
			var event Event
			if err := json.Unmarshal(msg.Body, &event); err != nil {
				logx.Errorf("unmarshal operation event failed: %v", err)
				_ = msg.Nack(false, false)
				continue
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
INSERT INTO operation_logs (user_id, event_type, resource_type, resource_id, metadata, ip, user_agent)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
				userID, event.EventType, event.ResourceType, resourceID, metadataValue, event.IP, event.UserAgent)
			if err != nil {
				logx.Errorf("write operation log failed: %v", err)
				_ = msg.Nack(false, true)
				continue
			}
			_ = msg.Ack(false)
		}
	}()
}
