package json_time_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hmmm42/city-picks/pkg/json_time"
	"github.com/stretchr/testify/assert"
)

func TestCustomTime_UnmarshalJSON(t *testing.T) {
	// Test case 1: Valid time string
	jsonStr1 := `"2023-10-26 10:30:00"`
	expectedTime1, _ := time.Parse("2006-01-02 15:04:05", "2023-10-26 10:30:00")
	var ct1 json_time.CustomTime
	err := json.Unmarshal([]byte(jsonStr1), &ct1)
	assert.NoError(t, err)
	assert.Equal(t, expectedTime1, time.Time(ct1))

	// Test case 2: Invalid time string format
	jsonStr2 := `"2023/10/26 10:30:00"`
	var ct2 json_time.CustomTime
	err = json.Unmarshal([]byte(jsonStr2), &ct2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time format must be '2006-01-02 15:04:05'")

	// Test case 3: Non-string input
	jsonStr3 := `12345`
	var ct3 json_time.CustomTime
	err = json.Unmarshal([]byte(jsonStr3), &ct3)
	assert.Error(t, err)

	// Test case 4: Empty string
	jsonStr4 := `""`
	var ct4 json_time.CustomTime
	err = json.Unmarshal([]byte(jsonStr4), &ct4)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time format must be '2006-01-02 15:04:05'")
}

func TestCustomTime_MarshalJSON(t *testing.T) {
	// Test case 1: Valid time
	time1, _ := time.Parse("2006-01-02 15:04:05", "2023-10-26 10:30:00")
	ct1 := json_time.CustomTime(time1)
	marshaled1, err := json.Marshal(ct1)
	assert.NoError(t, err)
	assert.Equal(t, `"2023-10-26 10:30:00"`, string(marshaled1))

	// Test case 2: Zero time
	ct2 := json_time.CustomTime(time.Time{})
	marshaled2, err := json.Marshal(ct2)
	assert.NoError(t, err)
	// Note: time.Time{} formats to "0001-01-01 00:00:00"
	assert.Equal(t, `"0001-01-01 00:00:00"`, string(marshaled2))
}
