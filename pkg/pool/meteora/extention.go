package meteora

import (
	"fmt"
	"math/big"
)

// BinArrayBitmapExtension represents an extension of the bin array bitmap
// that handles both positive and negative bin array indices
type BinArrayBitmapExtension struct {
	PositiveBinArrayBitmap [][8]uint64 // Corresponds to positive_bin_array_bitmap in Rust
	NegativeBinArrayBitmap [][8]uint64 // Corresponds to negative_bin_array_bitmap in Rust
}

// BitmapRange returns the minimum and maximum bitmap indices
func (extension *BinArrayBitmapExtension) BitmapRange() (int32, int32) {
	return -BinArrayBitmapSize * (int32(ExtensionBinArrayBitmapSize) + 1),
		BinArrayBitmapSize*(int32(ExtensionBinArrayBitmapSize)+1) - 1
}

// NextBinArrayIndexWithLiquidity finds the next bin array index with liquidity
// based on the swap direction and starting index
func (extension *BinArrayBitmapExtension) NextBinArrayIndexWithLiquidity(swapForY bool, startIndex int32) (int32, bool, error) {
	minBitmapID, maxBitmapID := extension.BitmapRange()

	if startIndex > 0 {
		if swapForY {
			value, err := extension.IterBitmap(startIndex, BinArrayBitmapSize)
			if err != nil {
				return 0, false, err
			}
			if value != nil {
				return *value, true, nil
			}
			return BinArrayBitmapSize - 1, false, nil
		} else {
			value, err := extension.IterBitmap(startIndex, maxBitmapID)
			if err != nil {
				return 0, false, err
			}
			if value != nil {
				return *value, true, nil
			}
			return 0, false, fmt.Errorf("cannot find non zero liquidity bin array id")
		}
	} else {
		if swapForY {
			value, err := extension.IterBitmap(startIndex, minBitmapID)
			if err != nil {
				return 0, false, err
			}
			if value != nil {
				return *value, true, nil
			}
			return 0, false, fmt.Errorf("cannot find non zero liquidity bin array id")
		} else {
			value, err := extension.IterBitmap(startIndex, -BinArrayBitmapSize-1)
			if err != nil {
				return 0, false, err
			}
			if value != nil {
				return *value, true, nil
			}
			return -BinArrayBitmapSize, false, nil
		}
	}
}

// IterBitmap iterates through the bitmap from startIndex to endIndex
// and returns the first bin array index with liquidity
func (extension *BinArrayBitmapExtension) IterBitmap(startIndex, endIndex int32) (*int32, error) {
	// If start index equals end index, check that specific bit
	if startIndex == endIndex {
		hasBit, err := extension.Bit(startIndex)
		if err != nil {
			return nil, err
		}
		if hasBit {
			return &startIndex, nil
		}
		return nil, nil
	}

	offset, err := GetBitmapOffset(startIndex)
	if err != nil {
		return nil, err
	}

	binArrayOffset, err := BinArrayOffsetInBitmap(startIndex)
	if err != nil {
		return nil, err
	}

	if startIndex < 0 {
		// Handle negative range
		if startIndex < endIndex {
			// Forward iteration
			for i := offset; i >= 0; i-- {
				// Convert [8]uint64 to big.Int
				binArrayBitmap := ArrayToBigInt(extension.NegativeBinArrayBitmap[i])

				if i == offset {
					// Left shift operation
					shift := big.NewInt(int64(BinArrayBitmapSize - binArrayOffset - 1))
					binArrayBitmap.Lsh(binArrayBitmap, uint(shift.Int64()))

					if binArrayBitmap.Sign() == 0 {
						continue
					}

					// Calculate leading zeros count
					leadingZeros := CountLeadingZeros(binArrayBitmap)
					binArrayOffsetInBitmap := binArrayOffset - leadingZeros

					return ToBinArrayIndex(i, binArrayOffsetInBitmap, false)
				}

				if binArrayBitmap.Sign() == 0 {
					continue
				}

				leadingZeros := CountLeadingZeros(binArrayBitmap)
				binArrayOffsetInBitmap := BinArrayBitmapSize - leadingZeros - 1

				return ToBinArrayIndex(i, binArrayOffsetInBitmap, false)
			}
		} else {
			// Backward iteration
			for i := offset; i < ExtensionBinArrayBitmapSize; i++ {
				binArrayBitmap := ArrayToBigInt(extension.NegativeBinArrayBitmap[i])

				if i == offset {
					// Right shift operation
					binArrayBitmap.Rsh(binArrayBitmap, uint(binArrayOffset))

					if binArrayBitmap.Sign() == 0 {
						continue
					}

					trailingZeros := CountTrailingZeros(binArrayBitmap)
					binArrayOffsetInBitmap := binArrayOffset + trailingZeros

					return ToBinArrayIndex(i, binArrayOffsetInBitmap, false)
				}

				if binArrayBitmap.Sign() == 0 {
					continue
				}

				binArrayOffsetInBitmap := CountTrailingZeros(binArrayBitmap)
				return ToBinArrayIndex(i, binArrayOffsetInBitmap, false)
			}
		}
	} else {
		// Handle positive range
		if startIndex < endIndex {
			// Forward iteration
			for i := offset; i < ExtensionBinArrayBitmapSize; i++ {
				binArrayBitmap := ArrayToBigInt(extension.PositiveBinArrayBitmap[i])

				if i == offset {
					binArrayBitmap.Rsh(binArrayBitmap, uint(binArrayOffset))

					if binArrayBitmap.Sign() == 0 {
						continue
					}

					trailingZeros := CountTrailingZeros(binArrayBitmap)
					binArrayOffsetInBitmap := binArrayOffset + trailingZeros

					return ToBinArrayIndex(i, binArrayOffsetInBitmap, true)
				}

				if binArrayBitmap.Sign() == 0 {
					continue
				}

				binArrayOffsetInBitmap := CountTrailingZeros(binArrayBitmap)
				return ToBinArrayIndex(i, binArrayOffsetInBitmap, true)
			}
		} else {
			// Backward iteration
			for i := offset; i >= 0; i-- {
				binArrayBitmap := ArrayToBigInt(extension.PositiveBinArrayBitmap[i])

				if i == offset {
					shift := big.NewInt(int64(BinArrayBitmapSize - binArrayOffset - 1))
					binArrayBitmap.Lsh(binArrayBitmap, uint(shift.Int64()))

					if binArrayBitmap.Sign() == 0 {
						continue
					}

					leadingZeros := CountLeadingZeros(binArrayBitmap)
					binArrayOffsetInBitmap := binArrayOffset - leadingZeros

					return ToBinArrayIndex(i, binArrayOffsetInBitmap, true)
				}

				if binArrayBitmap.Sign() == 0 {
					continue
				}

				leadingZeros := CountLeadingZeros(binArrayBitmap)
				binArrayOffsetInBitmap := BinArrayBitmapSize - leadingZeros - 1

				return ToBinArrayIndex(i, binArrayOffsetInBitmap, true)
			}
		}
	}

	return nil, nil
}

// Bit checks if a specific bit is set in the bitmap at the given bin array index
func (extension *BinArrayBitmapExtension) Bit(binArrayIndex int32) (bool, error) {
	// Get bitmap data
	bitmap, _, err := extension.GetBitmap(binArrayIndex)
	if err != nil {
		return false, err
	}

	// Get offset within the bitmap
	binArrayOffset, err := BinArrayOffsetInBitmap(binArrayIndex)
	if err != nil {
		return false, err
	}

	// Convert [8]uint64 to big.Int
	bigInt := ArrayToBigInt(bitmap)

	// Check the bit value at the specified position
	return bigInt.Bit(binArrayOffset) == 1, nil
}

// GetBitmap retrieves the bitmap data for a given bin array index
func (extension *BinArrayBitmapExtension) GetBitmap(binArrayIndex int32) ([8]uint64, int, error) {
	offset, err := GetBitmapOffset(binArrayIndex)
	if err != nil {
		return [8]uint64{}, 0, err
	}

	if binArrayIndex < 0 {
		return extension.NegativeBinArrayBitmap[offset], offset, nil
	}
	return extension.PositiveBinArrayBitmap[offset], offset, nil
}
