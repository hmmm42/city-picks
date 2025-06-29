package repository

import (
	"context"

	"github.com/redis/go-redis/v9"
)

const (
	orderStreamKey = "stream:orders"
	orderGroup     = "group:orders"
)

type MessageQueue interface {
	// 生产者方法
	AddOrderToStream(ctx context.Context, values map[string]any) (string, error)

	// 消费者方法
	CreateGroup(ctx context.Context) error
	ReadPendingMessages(ctx context.Context, consumerName string) ([]redis.XMessage, error)
	Ack(ctx context.Context, msgID string) error
}

type messageQueue struct {
	rdb *redis.Client
}

func (m *messageQueue) AddOrderToStream(ctx context.Context, values map[string]any) (string, error) {
	return m.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: orderStreamKey,
		Values: values,
	}).Result()
}

func (m *messageQueue) CreateGroup(ctx context.Context) error {
	_, err := m.rdb.XGroupCreateMkStream(ctx, orderStreamKey, orderGroup, "0").Result()
	// 0 标识从头开始消费, $ 标识从最新消息开始消费
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (m *messageQueue) ReadPendingMessages(ctx context.Context, consumerName string) ([]redis.XMessage, error) {
	streams, err := m.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    orderGroup,
		Consumer: consumerName,
		Streams:  []string{orderStreamKey, ">"}, // ">" 表示只读取未被消费的消息
		Count:    1,                             // 每次读取1条消息
		Block:    0,                             // 阻塞时间为0表示立即返回
	}).Result()

	if err != nil {
		return nil, err
	}
	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil // 没有新消息
	}

	return streams[0].Messages, nil
}

func (m *messageQueue) Ack(ctx context.Context, msgID string) error {
	return m.rdb.XAck(ctx, orderStreamKey, msgID, orderGroup).Err()
}

func NewMessageQueue(rdb *redis.Client) MessageQueue {
	return &messageQueue{
		rdb: rdb,
	}
}
