package json_time

import (
	"encoding/json"
	"fmt"
	"time"
)

// CustomTime 自定义时间类型，支持JSON解析
type CustomTime time.Time

// UnmarshalJSON 实现JSON反序列化
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	var timeStr string
	if err := json.Unmarshal(data, &timeStr); err != nil {
		return err
	}

	layout := "2006-01-02 15:04:05"
	t, err := time.Parse(layout, timeStr)
	if err != nil {
		return fmt.Errorf("time format must be '2006-01-02 15:04:05', got: %s", timeStr)
	}

	*ct = CustomTime(t)
	return nil
}

// MarshalJSON 实现JSON序列化
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	layout := "2006-01-02 15:04:05"
	formattedTime := time.Time(ct).Format(layout)
	return json.Marshal(formattedTime)
}
