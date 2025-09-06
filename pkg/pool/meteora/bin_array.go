package meteora

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"lukechampine.com/uint128"
)

// BinArray represents an array of liquidity bins in the Meteora DLMM protocol
// Each bin array contains 70 bins for a specific price range
type BinArray struct {
	index   int64
	version uint8
	padding [7]uint8
	LbPair  solana.PublicKey
	bins    [70]Bin
}

// GetBinMut retrieves a mutable reference to a bin by its active ID
func (binArray *BinArray) GetBinMut(activeID int32) (*Bin, error) {
	// Get the bin index in the array
	index, err := binArray.GetBinIndexInArray(activeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bin index: %w", err)
	}

	// Check if index is within valid range
	if index >= len(binArray.bins) {
		return nil, fmt.Errorf("bin index out of range")
	}

	// Return pointer to the corresponding bin
	return &binArray.bins[index], nil
}

// GetBinIndexInArray calculates the index of a bin within the array based on its active ID
func (binArray *BinArray) GetBinIndexInArray(activeID int32) (int, error) {
	// Check if bin ID is within range
	isWithinRange, err := binArray.IsBinIDWithinRange(activeID)
	if err != nil {
		return 0, fmt.Errorf("failed to check bin range: %w", err)
	}
	if !isWithinRange {
		return 0, fmt.Errorf("bin id out of range")
	}

	// Get the lower bound ID
	lowerBinID, _, err := GetBinArrayLowerUpperBinID(int32(binArray.index))
	if err != nil {
		return 0, fmt.Errorf("failed to get bin array bounds: %w", err)
	}

	// Calculate index (check for subtraction overflow)
	index := activeID - lowerBinID
	if index < 0 || activeID < lowerBinID { // Check for subtraction overflow
		return 0, fmt.Errorf("index calculation overflow")
	}

	return int(index), nil
}

// IsBinIDWithinRange checks if the given active ID is within the range of this bin array
func (binArray *BinArray) IsBinIDWithinRange(activeID int32) (bool, error) {
	lowerBinID, upperBinID, err := GetBinArrayLowerUpperBinID(int32(binArray.index))
	if err != nil {
		return false, fmt.Errorf("failed to get bin array bounds: %w", err)
	}
	return activeID >= lowerBinID && activeID <= upperBinID, nil
}

// ParseBinArray deserializes binary data into a BinArray structure
func ParseBinArray(data []byte) (BinArray, error) {
	if len(data) < 16 {
		return BinArray{}, errors.New("data too short")
	}

	// Skip account discriminator (8 bytes)
	offset := 8

	// Read index (int64)
	index := int64(binary.LittleEndian.Uint64(data[offset : offset+8]))
	offset += 8

	// Read version (uint8)
	version := data[offset]
	offset++

	// Read padding (7 bytes)
	padding := [7]uint8{}
	copy(padding[:], data[offset:offset+7])
	offset += 7

	// Read lbPair (32 bytes)
	lbPair := solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	// Read bins (70 * Bin)
	bins := [70]Bin{}
	for i := 0; i < 70; i++ {
		// Read amountX (uint64)
		amountX := binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		// Read amountY (uint64)
		amountY := binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		// Read price (uint128)
		price := uint128.Uint128{
			Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
			Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		}
		offset += 16

		// Read liquiditySupply (uint128)
		liquiditySupply := uint128.Uint128{
			Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
			Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		}
		offset += 16

		// Read rewardPerTokenStored (2 * uint128)
		rewardPerTokenStored := [2]uint128.Uint128{}
		for j := 0; j < 2; j++ {
			rewardPerTokenStored[j] = uint128.Uint128{
				Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
				Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
			}
			offset += 16
		}

		// Read feeAmountXPerTokenStored (uint128)
		feeAmountXPerTokenStored := uint128.Uint128{
			Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
			Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		}
		offset += 16

		// Read feeAmountYPerTokenStored (uint128)
		feeAmountYPerTokenStored := uint128.Uint128{
			Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
			Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		}
		offset += 16

		// Read amountXIn (uint128)
		amountXIn := uint128.Uint128{
			Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
			Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		}
		offset += 16

		// Read amountYIn (uint128)
		amountYIn := uint128.Uint128{
			Lo: binary.LittleEndian.Uint64(data[offset : offset+8]),
			Hi: binary.LittleEndian.Uint64(data[offset+8 : offset+16]),
		}
		offset += 16

		bins[i] = Bin{
			amountX:                  amountX,
			amountY:                  amountY,
			price:                    price,
			liquiditySupply:          liquiditySupply,
			rewardPerTokenStored:     rewardPerTokenStored,
			feeAmountXPerTokenStored: feeAmountXPerTokenStored,
			feeAmountYPerTokenStored: feeAmountYPerTokenStored,
			amountXIn:                amountXIn,
			amountYIn:                amountYIn,
		}
	}

	return BinArray{
		index:   index,
		version: version,
		padding: padding,
		LbPair:  lbPair,
		bins:    bins,
	}, nil
}
