package sf

import (
	"fmt"
	"time"

	"github.com/sony/sonyflake"
)

func NewSonyflake() (*sonyflake.Sonyflake, error) {
	// Sonyflake's start time is configurable.
	// This is January 1, 2024 00:00:00 UTC
	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// In a real application, the MachineID should be configured
	// to be unique for each node. For example, it could be derived
	// from the instance's IP address or a pre-assigned node ID.
	st := sonyflake.Settings{
		StartTime: startTime,
		MachineID: func() (uint16, error) {
			return 1, nil // Hardcoded for simplicity
		},
	}

	sf := sonyflake.NewSonyflake(st)
	if sf == nil {
		return nil, fmt.Errorf("failed to create sonyflake instance")
	}

	return sf, nil
}
