package mq

import (
	"context"
	"log/slog"
	"strconv"

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

	go func() {
		for {
			select {
			case <-ctx.Done():
				slog.Info("Order consumer stopped", "consumer", c.consumerName)
				return
			default:
				messages, err := c.mq.ReadPendingMessages(ctx, c.consumerName)
				if err != nil {
					slog.Error("failed to read pending messages", "err", err)
					continue
				}
				if len(messages) == 0 {
					continue
				}
				for _, msg := range messages {
					c.handleMessage(ctx, msg)
				}
			}
		}
	}()
}

func (c *OrderConsumer) handleMessage(ctx context.Context, msg redis.XMessage) {
	//slog.Debug("Received message", "messageID", msg.ID, "values", msg.Values)
	voucherID, _ := strconv.ParseUint(msg.Values["voucherID"].(string), 10, 64)
	userID, _ := strconv.ParseUint(msg.Values["userID"].(string), 10, 64)
	orderID, _ := strconv.ParseInt(msg.Values["orderID"].(string), 10, 64)

	order := &model.TbVoucherOrder{
		ID:        orderID,
		VoucherID: voucherID,
		UserID:    userID,
	}

	if err := c.voucherService.CreateVoucherOrderDB(ctx, order); err != nil {
		slog.Error("failed to create voucher order", "err", err, "orderID", order.ID, "voucherID", voucherID, "userID", userID)
		return
	}
	if err := c.mq.Ack(ctx, msg.ID); err != nil {
		slog.Error("failed to acknowledge message", "err", err, "messageID", msg.ID)
	}

}
