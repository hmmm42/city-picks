package db

import (
	"fmt"

	"github.com/hmmm42/city-picks/dal/query"
	"github.com/hmmm42/city-picks/internal/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DBEngine *gorm.DB

func NewMySQL(config *config.MySQLSetting) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.DBName,
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)

	// 设置了才能使用 query 包
	query.SetDefault(db)
	return db, nil
}
