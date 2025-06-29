package mq

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/hmmm42/city-picks/internal/service"
	"github.com/redis/go-redis/v9"
)

type OrderConsumer struct {
	consumerName   string
	mq             repository.MessageQueue
	voucherService service.VoucherService
}

const (
	maxRetries        = 3
	checkIdleInterval = 2 * time.Second
)

func NewOrderConsumer(mq repository.MessageQueue, svc service.VoucherService) *OrderConsumer {
	return &OrderConsumer{
		// TODO: 这里的 consumerName 可以从配置中获取
		consumerName:   "consumer-1",
		mq:             mq,
		voucherService: svc,
	}
}

func (c *OrderConsumer) Start(ctx context.Context) {
	if err := c.mq.CreateGroup(ctx); err != nil {
		panic(err)
	}

	slog.Info("Order consumer started", "consumer", c.consumerName)

	go c.ConsumeMessages(ctx)
}

func (c *OrderConsumer) ConsumeMessages(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("Order consumer stopped", "consumer", c.consumerName)
			return
		default:
			msgs, err := c.mq.ReadPendingMessages(ctx, c.consumerName)
			if err != nil {
				slog.Error("failed to read pending messages", "err", err)
				time.Sleep(checkIdleInterval) // 避免错误循环
				continue
			}
			if len(msgs) == 0 {
				//time.Sleep(checkIdleInterval) // 没有消息时稍作等待
				continue
			}

			for _, msg := range msgs {
				var retryCount int
				if count, ok := msg.Values["retry_count"].(string); ok {
					retryCount, _ = strconv.Atoi(count)
				}

				if retryCount >= maxRetries {
					slog.Warn("Message has reached max retries, moving to DLQ", "messageID", msg.ID, "retryCount", retryCount)
					c.moveToDLQ(ctx, msg, fmt.Errorf("reached max retries: (%d)", retryCount))
					continue
				}

				if err = c.handleOrderMsg(ctx, msg); err != nil {
					slog.Error("failed to handle order message, requeueing", "messageID", msg.ID, "error", err)
					c.requeueMessage(ctx, msg, retryCount+1)
					continue
				}

				if err = c.mq.Ack(ctx, repository.OrderStreamKey, repository.OrderGroup, msg.ID); err != nil {
					slog.Error("failed to ACK message", "err", err, "messageID", msg.ID)
					// 如果 ACK 失败，我们选择不重试，可能是因为消息已经被处理过了
				}
			}
		}
	}
}

func (c *OrderConsumer) handleOrderMsg(ctx context.Context, msg redis.XMessage) error {
	//slog.Debug("Received message", "messageID", msg.ID, "values", msg.Values)
	voucherID, _ := strconv.ParseUint(msg.Values["voucherID"].(string), 10, 64)
	userID, _ := strconv.ParseUint(msg.Values["userID"].(string), 10, 64)
	orderID, _ := strconv.ParseInt(msg.Values["orderID"].(string), 10, 64)

	order := &model.TbVoucherOrder{
		ID:        orderID,
		VoucherID: voucherID,
		UserID:    userID,
	}

	return c.voucherService.CreateVoucherOrderDB(ctx, order)
}

//func (c *OrderConsumer) processNewMsgs(ctx context.Context) {
//	for {
//		select {
//		case <-ctx.Done():
//			slog.Info("Order consumer stopped", "consumer", c.consumerName)
//			return
//		default:
//			messages, err := c.mq.ReadPendingMessages(ctx, c.consumerName)
//			if err != nil {
//				slog.Error("failed to read pending messages", "err", err)
//				time.Sleep(1 * time.Second) // 避免错误循环
//				continue
//			}
//			if len(messages) == 0 {
//				continue
//			}
//			for _, msg := range messages {
//				if err = c.handleOrderMsg(ctx, msg); err != nil {
//					slog.Error("failed to handle new message, will be retried later")
//				}
//			}
//		}
//	}
//}
//
//func (c *OrderConsumer) recoverAndHandleDeadLetters(ctx context.Context) {
//	ticker := time.NewTicker(checkIdleInterval)
//	defer ticker.Stop()
//
//	for {
//		select {
//		case <-ctx.Done():
//			slog.Info("Order consumer stopped", "consumer", c.consumerName)
//			return
//		case <-ticker.C:
//			claimedMessages, err := c.mq.ClaimMessage(ctx, c.consumerName, minIdleTime, 10)
//			if err != nil {
//				slog.Error("failed to claim messages", "err", err)
//				continue
//			}
//			if len(claimedMessages) == 0 {
//				continue
//			}
//			slog.Info("Claimed messages", "count", len(claimedMessages))
//			for _, msg := range claimedMessages {
//				if err = c.handleOrderMsg(ctx, msg); err != nil {
//					slog.Error("failed to handle message", "err", err)
//
//				}
//			}
//		}
//	}
//}

func (c *OrderConsumer) requeueMessage(ctx context.Context, msg redis.XMessage, newRetryCount int) {
	requeueValues := make(map[string]any)
	for k, v := range msg.Values {
		requeueValues[k] = v
	}
	// 更新或添加重试次数字段
	requeueValues["retry_count"] = strconv.Itoa(newRetryCount)

	// 将新消息添加到流的末尾
	if _, err := c.mq.AddToStream(ctx, repository.OrderStreamKey, requeueValues); err != nil {
		slog.Error("failed to requeue message", "err", err, "originalMessageID", msg.ID)
		// 如果重入队列失败，我们选择不ACK原消息，让它被其他消费者认领，这是降级策略
		return
	}

	// 确认原消息，将其从PEL中移除
	if err := c.mq.Ack(ctx, repository.OrderStreamKey, repository.OrderGroup, msg.ID); err != nil {
		slog.Error("failed to ACK message after requeueing", "err", err, "messageID", msg.ID)
	}
}

func (c *OrderConsumer) moveToDLQ(ctx context.Context, msg redis.XMessage, processErr error) {
	slog.Warn("Message failed after max retries, moving to DLQ", "messageID", msg.ID, "error", processErr.Error())

	// 在消息中添加失败信息
	dlqValues := map[string]any{
		"original_id": msg.ID,
		"consumer":    c.consumerName,
		"error":       processErr.Error(),
		"failed_at":   time.Now().Format(time.RFC3339),
	}
	for k, v := range msg.Values {
		dlqValues[k] = v
	}

	// 写入死信队列
	if _, err := c.mq.AddToStream(ctx, repository.DeadLetterStreamKey, dlqValues); err != nil {
		slog.Error("failed to add message to DLQ", "err", err, "originalMessageID", msg.ID)
		// 如果连DLQ都失败，还是要尝试ACK原消息，防止阻塞
	}

	// 从主队列中 ACK，移除该消息
	if err := c.mq.Ack(ctx, repository.OrderStreamKey, repository.OrderGroup, msg.ID); err != nil {
		slog.Error("failed to ACK message after moving to DLQ", "err", err, "messageID", msg.ID)
	}
}
