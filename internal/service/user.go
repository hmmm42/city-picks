package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strconv"
	"time"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/internal/repository"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	userNickNamePrefix = "user_"
	phoneKeyPrefix     = "phone:"
)

type UserService interface {
	GetVerificationCode(ctx context.Context, phone string) (string, error)
	LoginWithCode(ctx context.Context, phone, code string) (*model.TbUser, error)
	LoginWithPwd(ctx context.Context, phone, password string) (*model.TbUser, error)
}

type userService struct {
	userRepo    repository.UserRepo
	redisClient *redis.Client
	logger      *slog.Logger
}

func (u userService) GetVerificationCode(ctx context.Context, phone string) (string, error) {
	code := fmt.Sprintf("%06v", rand.Int31n(1000000))
	key := phoneKeyPrefix + phone

	success, err := u.redisClient.SetNX(ctx, key, code, 5*time.Minute).Result()
	if err != nil {
		u.logger.Error("redis SetNX error", "err", err)
		return "", err
	}
	if !success {
		return "", fmt.Errorf("verification code request too frequent")
	}
	return code, nil
}

func (u userService) LoginWithCode(ctx context.Context, phone, code string) (*model.TbUser, error) {
	key := phoneKeyPrefix + phone
	storedCode, err := u.redisClient.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("verification code expired or not found: %w", err)
	}
	if err != nil {
		u.logger.Error("redis Get error", "err", err)
		return nil, err
	}

	if storedCode != code {
		return nil, fmt.Errorf("incorrect verification code")
	}

	user, err := u.userRepo.FindByPhone(ctx, phone)
	if errors.Is(err, gorm.ErrRecordNotFound) { // 新用户
		newUser := &model.TbUser{
			Phone:    phone,
			NickName: userNickNamePrefix + strconv.Itoa(rand.Intn(100000)), // 随机昵称
		}

		if err = u.userRepo.Create(ctx, newUser); err != nil {
			u.logger.Error("failed to create new user", "err", err)
			return nil, fmt.Errorf("failed to create new user: %w", err)
		}
		return newUser, nil
	}

	if err != nil {
		u.logger.Error("failed to find user by phone", "err", err)
		return nil, err
	}

	// 删除验证码，防止重复使用
	err = u.redisClient.Del(ctx, key).Err()
	return user, err
}

func (u userService) LoginWithPwd(ctx context.Context, phone, password string) (*model.TbUser, error) {
	user, err := u.userRepo.FindByPhone(ctx, phone)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("user not found with phone: %s", phone)
	}
	if err != nil {
		u.logger.Error("failed to find user by phone", "err", err)
		return nil, err
	}
	if user.Password != password {
		return nil, fmt.Errorf("incorrect password for phone: %s", phone)
	}
	return user, nil
}

func NewUserService(userRepo repository.UserRepo, redisClient *redis.Client, logger *slog.Logger) UserService {
	return &userService{
		userRepo:    userRepo,
		redisClient: redisClient,
		logger:      logger,
	}
}
