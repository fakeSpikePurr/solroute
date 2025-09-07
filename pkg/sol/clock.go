package sol

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
)

const (
	// ClockAccountDataSize represents the expected size of the clock account data in bytes
	ClockAccountDataSize = 40
)

// Clock represents the Solana network's clock information
type Clock struct {
	Slot                uint64
	EpochStartTime      uint64
	Epoch               uint64
	LeaderScheduleEpoch uint64
	UnixTimestamp       uint64
}

// GetClock retrieves the current clock information from the Solana network
func (c *Client) GetClock(ctx context.Context) (*Clock, error) {
	// Fetch the clock account
	resp, err := c.GetAccountInfoWithOpts(ctx, solana.SysVarClockPubkey)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch clock account: %w", err)
	}

	if resp.Value == nil {
		return nil, errors.New("clock account not found in the network")
	}

	// Parse account data
	data := resp.Value.Data.GetBinary()
	if len(data) != ClockAccountDataSize {
		return nil, fmt.Errorf("invalid clock account data length: expected %d bytes, got %d", ClockAccountDataSize, len(data))
	}

	// Parse individual fields
	clock := &Clock{
		Slot:                binary.LittleEndian.Uint64(data[0:8]),
		EpochStartTime:      binary.LittleEndian.Uint64(data[8:16]),
		Epoch:               binary.LittleEndian.Uint64(data[16:24]),
		LeaderScheduleEpoch: binary.LittleEndian.Uint64(data[24:32]),
		UnixTimestamp:       binary.LittleEndian.Uint64(data[32:40]),
	}

	return clock, nil
}
