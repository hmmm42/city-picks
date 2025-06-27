package shopservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hmmm42/city-picks/dal/model"
	"github.com/hmmm42/city-picks/dal/query"
	"github.com/hmmm42/city-picks/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	//config.InitConfig(config.GetDefaultConfigPath())
	_, _ = config.NewOptions()
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.MySQLOptions.User,
		config.MySQLOptions.Password,
		config.MySQLOptions.Host,
		config.MySQLOptions.Port,
		"",
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)

	testDB := viper.GetString("mysql.dbname") + "_shadow"
	assert.NoError(t, db.Exec("DROP DATABASE IF EXISTS "+testDB).Error)
	assert.NoError(t, db.Exec("CREATE DATABASE IF NOT EXISTS "+testDB).Error)

	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.MySQLOptions.User,
		config.MySQLOptions.Password,
		config.MySQLOptions.Host,
		config.MySQLOptions.Port,
		testDB,
	)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	assert.NoError(t, err)
	assert.NoError(t, db.AutoMigrate(&model.TbVoucher{}, &model.TbSeckillVoucher{}, &model.TbVoucherOrder{}))

	query.SetDefault(db)
	return db
}

func TestCreateDB(t *testing.T) {
	db := setupTestDB(t)
	assert.NotNil(t, db, "Database connection should not be nil")
	// Create a test voucher
	voucher := model.TbVoucher{
		ShopID:      1,
		Title:       "Test Voucher",
		SubTitle:    "Test SubTitle",
		Rules:       "Test Rules",
		PayValue:    100,
		ActualValue: 50,
		Type:        1, // Ordinary voucher
	}
	err := query.TbVoucher.Create(&voucher)
	assert.NoError(t, err, "Failed to create test voucher")

	// Query the voucher to ensure it was created
	voucherQuery := query.TbVoucher
	voucherGot, err := voucherQuery.Where(voucherQuery.ID.Eq(voucher.ID)).First()
	assert.NoError(t, err, "Failed to query test voucher")
	t.Log(voucherGot)

	seckillV := model.TbSeckillVoucher{
		VoucherID: voucherGot.ID,
		Stock:     100,
		BeginTime: time.Now().Add(-5 * time.Minute),
		EndTime:   time.Now().Add(10 * time.Minute),
	}
	err = query.TbSeckillVoucher.Create(&seckillV)
	assert.NoError(t, err, "Failed to create seckill voucher")
}

func TestSeckillOutOfStock(t *testing.T) {
	_ = setupTestDB(t)

	var (
		initialStock   int64 = 100 // 初始库存
		totalSeckill         = 300 // 模拟300个秒杀请求
		seckillSuccess int64       // 成功秒杀的计数器
	)

	// 创建测试优惠券
	voucher := model.TbVoucher{
		ShopID:      1,
		Title:       "Test Seckill Voucher",
		SubTitle:    "Test SubTitle",
		Rules:       "Test Rules",
		PayValue:    100,
		ActualValue: 50,
		Type:        1,
	}
	err := query.TbVoucher.Create(&voucher)
	assert.NoError(t, err)

	// 创建秒杀优惠券，库存设置为10
	seckillVoucher := model.TbSeckillVoucher{
		VoucherID: voucher.ID,
		Stock:     initialStock,
		BeginTime: time.Now().Add(-5 * time.Minute),
		EndTime:   time.Now().Add(10 * time.Minute),
	}
	err = query.TbSeckillVoucher.Create(&seckillVoucher)
	assert.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(totalSeckill)
	atomic.StoreInt64(&seckillSuccess, 0)
	for userID := range totalSeckill {
		go func() {
			defer wg.Done()

			req := seckillRequest{
				VoucherID: int64(voucher.ID),
				UserID:    int64(userID + 1), // 用户ID从1开始
			}
			jsonData, _ := json.Marshal(req)

			w := httptest.NewRecorder()
			gin.SetMode(gin.TestMode)
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest(http.MethodPost, "/voucher/seckill", bytes.NewBuffer(jsonData))
			c.Request.Header.Set("Content-Type", "application/json")

			SeckillVoucher(c)
			if w.Code == http.StatusOK {
				atomic.AddInt64(&seckillSuccess, 1)
			}
		}()
	}
	wg.Wait()

	t.Logf("Total seckill requests: %d, Successful seckills: %d", totalSeckill, seckillSuccess)
	assert.LessOrEqual(t, seckillSuccess, totalSeckill)

	orderCount, err := query.TbVoucherOrder.Where(
		query.TbVoucherOrder.VoucherID.Eq(voucher.ID)).Count()
	assert.NoError(t, err, "Failed to count voucher orders")
	assert.LessOrEqual(t, orderCount, initialStock, "Order count should not exceed initial stock")

	finalSeckill, err := query.TbSeckillVoucher.Where(
		query.TbSeckillVoucher.VoucherID.Eq(voucher.ID)).First()
	assert.NoError(t, err, "Failed to query seckill voucher")
	assert.Equal(t, initialStock-seckillSuccess, finalSeckill.Stock, "Final stock should match expected value after seckills")
}
