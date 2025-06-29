package tests

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/hmmm42/city-picks/internal/service"
	"github.com/hmmm42/city-picks/pkg/code"
	"github.com/hmmm42/city-picks/pkg/json_time"
)

func TestAddSeckillVoucher(t *testing.T) {
	v := &service.VoucherDTO{
		ShopID:      1,
		Title:       "test_title" + time.Now().Format("20060102150405"), // 确保标题唯一
		SubTitle:    "test_subTitle",
		Rules:       "test_rules",
		PayValue:    100,
		ActualValue: 50,
		Type:        1, // 特价券
		Stock:       100,
		BeginTime:   json_time.CustomTime(time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)),
		EndTime:     json_time.CustomTime(time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)),
	}
	data, _ := json.Marshal(v)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post("http://localhost:14530/voucher/create", "application/json", bytes.NewBuffer(data))
	if err != nil {
		t.Errorf("failed to send request: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("failed to read response body: %v", err)
	}

	// Unmarshal the response body into the Response struct
	var apiResponse code.Response
	err = json.Unmarshal(bodyBytes, &apiResponse)
	if err != nil {
		t.Errorf("failed to unmarshal response body: %v. Raw response: %s", err, string(bodyBytes))
	}

	// Assertions
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status OK, got %d. Raw response: %s", resp.StatusCode, string(bodyBytes))
	}
	if apiResponse.Code != code.ErrSuccess {
		t.Errorf("expected success code %d, got %d. Message: %s. Raw response: %s", code.ErrSuccess, apiResponse.Code, apiResponse.Message, string(bodyBytes))
	}
}
