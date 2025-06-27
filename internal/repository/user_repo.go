package repository

import (
	"context"

	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"gorm.io/gorm"
)

type UserRepo interface {
	FindByPhone(ctx context.Context, phone string) (*model.TbUser, error)
	Create(ctx context.Context, user *model.TbUser) error
}

func NewUserRepo(db *gorm.DB) UserRepo {
	return &userRepo{q: query.Use(db)}
}

type userRepo struct {
	q *query.Query
}

func (r *userRepo) FindByPhone(ctx context.Context, phone string) (*model.TbUser, error) {
	u := r.q.TbUser
	return u.WithContext(ctx).Where(u.Phone.Eq(phone)).First()
}

func (r *userRepo) Create(ctx context.Context, user *model.TbUser) error {
	return r.q.TbUser.WithContext(ctx).Create(user)
}
