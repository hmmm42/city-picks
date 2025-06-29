package repository

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	OrderStreamKey      = "stream:orders"
	OrderGroup          = "group:orders"
	DeadLetterStreamKey = "stream:orders:dead"
)

type MessageQueue interface {
	AddToStream(ctx context.Context, streamKey string, values map[string]any) (string, error)
	AddOrderToStream(ctx context.Context, values map[string]any) (string, error)

	CreateGroup(ctx context.Context) error
	ReadPendingMessages(ctx context.Context, consumerName string) ([]redis.XMessage, error)
	Ack(ctx context.Context, streamKey, groupName, msgID string) error
}

type messageQueue struct {
	rdb *redis.Client
}

func (m *messageQueue) AddToStream(ctx context.Context, streamKey string, values map[string]any) (string, error) {
	return m.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: values,
	}).Result()
}

func (m *messageQueue) AddOrderToStream(ctx context.Context, values map[string]any) (string, error) {
	return m.AddToStream(ctx, OrderStreamKey, values)
}

func (m *messageQueue) CreateGroup(ctx context.Context) error {
	_, err := m.rdb.XGroupCreateMkStream(ctx, OrderStreamKey, OrderGroup, "0").Result()
	// 0 标识从头开始消费, $ 标识从最新消息开始消费
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (m *messageQueue) ReadPendingMessages(ctx context.Context, consumerName string) ([]redis.XMessage, error) {
	streams, err := m.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    OrderGroup,
		Consumer: consumerName,
		Streams:  []string{OrderStreamKey, ">"}, // ">" 表示只读取未被消费的消息
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

func (m *messageQueue) Ack(ctx context.Context, streamKey, groupName, msgID string) error {
	return m.rdb.XAck(ctx, streamKey, msgID, groupName).Err()
}

// ClaimMessage 从全体闲置消息中认领一条消息
func (m *messageQueue) ClaimMessage(ctx context.Context, consumerName string, minIdleTime time.Duration, count int64) ([]redis.XMessage, error) {
	msgs, _, err := m.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   OrderStreamKey,
		Group:    OrderGroup,
		Consumer: consumerName,
		MinIdle:  minIdleTime,
		Start:    "0-0", // 每次都从头开始扫描待处理列表
		Count:    count,
	}).Result()

	if errors.Is(err, redis.Nil) {
		return nil, nil // 没有待处理消息
	}
	if err != nil {
		return nil, err
	}
	return msgs, nil
}

func NewMessageQueue(rdb *redis.Client) MessageQueue {
	return &messageQueue{
		rdb: rdb,
	}
}
