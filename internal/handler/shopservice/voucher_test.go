package shopservice

//func setupTestDB(t *testing.T) (*gorm.DB, *redis.Client) {
//	// Load configuration
//	_, _ = config.NewOptions()
//
//	// Setup MySQL
//	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
//		config.MySQLOptions.User,
//		config.MySQLOptions.Password,
//		config.MySQLOptions.Host,
//		config.MySQLOptions.Port,
//		"",
//	)
//	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
//	assert.NoError(t, err)
//
//	testDB := viper.GetString("mysql.dbname") + "_shadow"
//	assert.NoError(t, db.Exec("DROP DATABASE IF EXISTS "+testDB).Error)
//	assert.NoError(t, db.Exec("CREATE DATABASE IF NOT EXISTS "+testDB).Error)
//
//	dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
//		config.MySQLOptions.User,
//		config.MySQLOptions.Password,
//		config.MySQLOptions.Host,
//		config.MySQLOptions.Port,
//		testDB,
//	)
//	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
//	assert.NoError(t, err)
//	assert.NoError(t, db.AutoMigrate(&model.TbVoucher{}, &model.TbSeckillVoucher{}, &model.TbVoucherOrder{}))
//
//	query.SetDefault(db)
//	persistent.DBEngine = db // Set the global DBEngine for query.Use
//
//	// Setup Redis
//	redisClient, _ := cache.NewRedisClient(config.RedisOptions)
//	assert.NoError(t, redisClient.FlushDB(context.Background()).Err()) // Clear Redis for tests
//
//	return db, redisClient
//}

//func TestCreateVoucher(t *testing.T) {
//	db, redisClient := setupTestDB(t)
//	defer func() {
//		sqlDB, _ := db.DB()
//		sqlDB.Close()
//		redisClient.Close()
//	}()
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//
//	voucherRepo := repository.NewVoucherRepo(query.Use(db), logger)
//	voucherOrderRepo := repository.NewVoucherOrderRepo(query.Use(db), logger)
//	voucherService := service.NewVoucherService(voucherRepo, voucherOrderRepo, redisClient, logger)
//	voucherHandler := NewVoucherHandler(voucherService)
//
//	// Test creating an ordinary voucher
//	ordinaryVoucherReq := CreateVoucherRequest{
//		ShopID:      1,
//		Title:       "Test Ordinary Voucher",
//		SubTitle:    "Test SubTitle",
//		Rules:       "Test Rules",
//		PayValue:    100,
//		ActualValue: 50,
//		Type:        0, // Ordinary voucher
//	}
//	jsonData, _ := json.Marshal(ordinaryVoucherReq)
//
//	w := httptest.NewRecorder()
//	gin.SetMode(gin.TestMode)
//	c, _ := gin.CreateTestContext(w)
//	c.Request, _ = http.NewRequest(http.MethodPost, "/voucher", bytes.NewBuffer(jsonData))
//	c.Request.Header.Set("Content-Type", "application/json")
//
//	voucherHandler.CreateVoucher(c)
//	assert.Equal(t, http.StatusOK, w.Code)
//
//	// Verify ordinary voucher creation
//	voucherQuery := query.TbVoucher
//	voucherGot, err := voucherQuery.Where(voucherQuery.Title.Eq("Test Ordinary Voucher")).First()
//	assert.NoError(t, err)
//	assert.NotNil(t, voucherGot)
//
//	// Test creating a seckill voucher
//	seckillVoucherReq := CreateVoucherRequest{
//		ShopID:      2,
//		Title:       "Test Seckill Voucher",
//		SubTitle:    "Test SubTitle",
//		Rules:       "Test Rules",
//		PayValue:    200,
//		ActualValue: 100,
//		Type:        1, // Seckill voucher
//		Stock:       100,
//		BeginTime:   time.Now().Add(-5 * time.Minute).Format("2006-01-02 15:04:05"),
//		EndTime:     time.Now().Add(10 * time.Minute).Format("2006-01-02 15:04:05"),
//	}
//	jsonData, _ = json.Marshal(seckillVoucherReq)
//
//	w = httptest.NewRecorder()
//	c, _ = gin.CreateTestContext(w)
//	c.Request, _ = http.NewRequest(http.MethodPost, "/voucher", bytes.NewBuffer(jsonData))
//	c.Request.Header.Set("Content-Type", "application/json")
//
//	voucherHandler.CreateVoucher(c)
//	assert.Equal(t, http.StatusOK, w.Code)
//
//	// Verify seckill voucher creation
//	seckillVoucherQuery := query.TbSeckillVoucher
//	seckillVoucherGot, err := seckillVoucherQuery.Where(seckillVoucherQuery.Stock.Eq(100)).First()
//	assert.NoError(t, err)
//	assert.NotNil(t, seckillVoucherGot)
//}

//func TestSeckillVoucher(t *testing.T) {
//	db, redisClient := setupTestDB(t)
//	defer func() {
//		sqlDB, _ := db.DB()
//		sqlDB.Close()
//		redisClient.Close()
//	}()
//
//	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
//
//	voucherRepo := repository.NewVoucherRepo(query.Use(db), logger)
//	voucherOrderRepo := repository.NewVoucherOrderRepo(query.Use(db), logger)
//	voucherService := service.NewVoucherService(voucherRepo, voucherOrderRepo, redisClient, logger)
//	voucherHandler := NewVoucherHandler(voucherService)
//
//	var (
//		initialStock   int64 = 100 // 初始库存
//		totalSeckill         = 300 // 模拟300个秒杀请求
//		seckillSuccess int64       // 成功秒杀的计数器
//	)
//
//	// Create test voucher
//	voucher := model.TbVoucher{
//		ShopID:      1,
//		Title:       "Test Seckill Voucher",
//		SubTitle:    "Test SubTitle",
//		Rules:       "Test Rules",
//		PayValue:    100,
//		ActualValue: 50,
//		Type:        1,
//	}
//	err := query.TbVoucher.Create(&voucher)
//	assert.NoError(t, err)
//
//	// Create seckill voucher
//	seckillVoucher := model.TbSeckillVoucher{
//		VoucherID: voucher.ID,
//		Stock:     initialStock,
//		BeginTime: time.Now().Add(-5 * time.Minute),
//		EndTime:   time.Now().Add(10 * time.Minute),
//	}
//	err = query.TbSeckillVoucher.Create(&seckillVoucher)
//	assert.NoError(t, err)
//
//	var wg sync.WaitGroup
//	wg.Add(totalSeckill)
//	atomic.StoreInt64(&seckillSuccess, 0)
//
//	for i := 0; i < totalSeckill; i++ {
//		userID := uint64(i + 1) // User ID starts from 1
//		go func(uid uint64) {
//			defer wg.Done()
//
//			req := SeckillVoucherRequest{
//				VoucherID: voucher.ID,
//				UserID:    uid,
//			}
//			jsonData, _ := json.Marshal(req)
//
//			w := httptest.NewRecorder()
//			gin.SetMode(gin.TestMode)
//			c, _ := gin.CreateTestContext(w)
//			c.Request, _ = http.NewRequest(http.MethodPost, "/voucher/seckill", bytes.NewBuffer(jsonData))
//			c.Request.Header.Set("Content-Type", "application/json")
//			c.Set("userID", uid) // Mock userID in context
//
//			voucherHandler.SeckillVoucher(c)
//			if w.Code == http.StatusOK {
//				atomic.AddInt64(&seckillSuccess, 1)
//			}
//		}(userID)
//	}
//	wg.Wait()
//
//	t.Logf("Total seckill requests: %d, Successful seckills: %d", totalSeckill, seckillSuccess)
//	assert.LessOrEqual(t, seckillSuccess, initialStock) // Successful seckills should not exceed initial stock
//
//	orderCount, err := query.TbVoucherOrder.Where(
//		query.TbVoucherOrder.VoucherID.Eq(voucher.ID)).Count()
//	assert.NoError(t, err, "Failed to count voucher orders")
//	assert.LessOrEqual(t, orderCount, initialStock, "Order count should not exceed initial stock")
//
//	finalSeckill, err := query.TbSeckillVoucher.Where(
//		query.TbSeckillVoucher.VoucherID.Eq(voucher.ID)).First()
//	assert.NoError(t, err, "Failed to query seckill voucher")
//	assert.Equal(t, initialStock-seckillSuccess, finalSeckill.Stock, "Final stock should match expected value after seckills")
//}
