package persistent

import (
	"fmt"

	"github.com/hmmm42/city-picks/dal/query"
	"github.com/hmmm42/city-picks/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewMySQL(opts *config.MySQLSetting) (*gorm.DB, func(), error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		opts.User,
		opts.Password,
		opts.Host,
		opts.Port,
		opts.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)

	// 设置了才能使用 query 包
	query.SetDefault(db)

	cleanup := func() {
		_ = sqlDB.Close()
	}
	return db, cleanup, nil
}
