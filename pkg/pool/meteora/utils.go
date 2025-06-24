package meteora

import (
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"lukechampine.com/uint128"
)

// MostSignificantBit finds the position of the most significant bit in a number
func MostSignificantBit(number *big.Int, bitLength int) int {
	highestIndex := bitLength - 1
	if number.Cmp(big.NewInt(0)) == 0 {
		return -1 // Return -1 to indicate null
	}

	for i := highestIndex; i >= 0; i-- {
		if number.Bit(i) != 0 {
			return highestIndex - i
		}
	}
	return -1 // Return -1 to indicate null
}

// LeastSignificantBit finds the position of the least significant bit in a number
func LeastSignificantBit(number *big.Int, bitLength int) int {
	if number.Cmp(big.NewInt(0)) == 0 {
		return -1 // Return -1 to indicate null
	}

	for i := 0; i < bitLength; i++ {
		if number.Bit(i) != 0 {
			return i
		}
	}
	return -1 // Return -1 to indicate null
}

// BitmapType represents the type of bitmap
type BitmapType int

const (
	U1024 BitmapType = iota
	Other
)

// BitmapDetail contains the bit and byte information for a bitmap type
type BitmapDetail struct {
	Bits  int
	Bytes int
}

// BitmapTypeDetail returns the detail information for a given bitmap type
func BitmapTypeDetail(bitmapType BitmapType) BitmapDetail {
	if bitmapType == U1024 {
		return BitmapDetail{
			Bits:  1024,
			Bytes: 1024 / 8,
		}
	} else {
		return BitmapDetail{
			Bits:  512,
			Bytes: 512 / 8,
		}
	}
}

const (
	// MaxBinPerArray is the maximum number of bins in a bin array
	MaxBinPerArray     = 70
	BinArraySeed       = "bin_array"
	BinArrayBitmapSeed = "bitmap"
)

// BinIDToBinArrayIndex converts a bin ID to its corresponding bin array index
func BinIDToBinArrayIndex(binID int32) int64 {
	// Calculate quotient and remainder
	quotient := binID / MaxBinPerArray
	remainder := binID % MaxBinPerArray

	// If binID is negative and there is a remainder, subtract 1 from the quotient
	if binID < 0 && remainder != 0 {
		quotient--
	}

	return int64(quotient)
}

// DeriveEventAuthorityPDA derives the event authority PDA
func DeriveEventAuthorityPDA() solana.PublicKey {
	seeds := [][]byte{[]byte("__event_authority")}
	pda, _, _ := solana.FindProgramAddress(seeds, MeteoraProgramID)
	return pda
}

// DeriveBinArrayPDA derives a bin array PDA for the given LB pair and bin array index
func DeriveBinArrayPDA(lbPair solana.PublicKey, binArrayIndex int64) (solana.PublicKey, uint8) {
	// Convert bin_array_index to little endian bytes
	binArrayIndexBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(binArrayIndexBytes, uint64(binArrayIndex))

	// Create the seeds slice
	seeds := [][]byte{
		[]byte(BinArraySeed),
		lbPair.Bytes(),
		binArrayIndexBytes,
	}

	// Find the PDA
	pda, bump, err := solana.FindProgramAddress(seeds, MeteoraProgramID)
	if err != nil {
		return solana.PublicKey{}, 0
	}

	return pda, bump
}

// DeriveBinArrayBitmapExtension derives the bin array bitmap extension PDA
func DeriveBinArrayBitmapExtension(lbPair solana.PublicKey) (solana.PublicKey, uint8) {
	pda, bump, err := solana.FindProgramAddress(
		[][]byte{
			[]byte(BinArrayBitmapSeed),
			lbPair.Bytes(),
		},
		MeteoraProgramID, // Replace with actual program ID
	)
	if err != nil {
		return solana.PublicKey{}, 0
	}

	return pda, bump
}

// BitmapRange returns the minimum and maximum bitmap index range
func BitmapRange() (int32, int32) {
	return -BinArrayBitmapSize, BinArrayBitmapSize - 1
}

// IsOverflowDefaultBinArrayBitmap checks if the given bin array index is out of the bitmap range
func IsOverflowDefaultBinArrayBitmap(binArrayIndex int32) bool {
	minBitmapID, maxBitmapID := BitmapRange()
	return binArrayIndex > maxBitmapID || binArrayIndex < minBitmapID
}

// FromLimbs converts a slice of uint64 limbs to a big.Int
func FromLimbs(limbs []uint64) *big.Int {
	// Create a new big integer
	result := new(big.Int)

	// If no limbs, return 0
	if len(limbs) == 0 {
		return result
	}

	// Process each limb from highest to lowest
	// Since big.Int expects big-endian bytes, we start from the highest position
	for i := len(limbs) - 1; i >= 0; i-- {
		// Shift result left by 64 bits
		result.Lsh(result, 64)
		// Add current limb
		result.Or(result, new(big.Int).SetUint64(limbs[i]))
	}

	return result
}

// GetBinArrayOffset calculates the offset for a bin array index
func GetBinArrayOffset(binArrayIndex int32) uint {
	return uint(binArrayIndex + BinArrayBitmapSize)
}

// SafeMulShrCast safely performs multiplication and right shift with casting
func SafeMulShrCast(x, y *big.Int, offset uint8, rounding Rounding) (*big.Int, error) {
	// Calculate mul_shr
	result, err := MulShr(x, y, offset, rounding)
	if err != nil {
		return nil, fmt.Errorf("mul shr calculation error: %w", err)
	}

	return result, nil
}

// SafeShlDivCast safely performs left shift and division with casting
func SafeShlDivCast(x, y *big.Int, offset uint8, rounding Rounding) (*big.Int, error) {
	result, err := ShlDiv(x, y, offset, rounding)
	if err != nil {
		return nil, fmt.Errorf("overflow in shl div: %w", err)
	}

	return result, nil
}

// SafeMulDivCast safely performs multiplication and division with casting
func SafeMulDivCast(x, y, denominator *big.Int, rounding Rounding) (*big.Int, error) {
	result := MulDiv(x, y, denominator, rounding)

	return result, nil
}

// Rounding represents the rounding mode for mathematical operations
type Rounding int

const (
	RoundingUp Rounding = iota
	RoundingDown
)

// MulShr calculates (x * y) >> offset
func MulShr(x, y *big.Int, offset uint8, rounding Rounding) (*big.Int, error) {
	one := big.NewInt(1)
	scale := new(big.Int).Lsh(one, uint(offset))
	denominator := scale
	if denominator.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("shift overflow")
	}
	return MulDiv(x, y, denominator, rounding), nil
}

// ShlDiv calculates (x << offset) / y
func ShlDiv(x, y *big.Int, offset uint8, rounding Rounding) (*big.Int, error) {
	one := big.NewInt(1)
	scale := new(big.Int).Lsh(one, uint(offset))
	if scale.Cmp(big.NewInt(0)) == 0 {
		return nil, fmt.Errorf("shift overflow")
	}
	return MulDiv(x, scale, y, rounding), nil
}

// MulDiv performs multiplication and division with rounding
func MulDiv(x, y, denominator *big.Int, rounding Rounding) *big.Int {
	// Convert to big.Int for calculation (equivalent to U256 in Rust)
	xBig := x
	yBig := y

	// Calculate product
	prod := new(big.Int).Mul(xBig, yBig)

	div, mod := new(big.Int).DivMod(prod, denominator, new(big.Int))

	if rounding == RoundingUp && mod.Sign() != 0 {
		return div.Add(div, big.NewInt(1))
	}

	return div
}

// GetPriceFromID calculates the price from active ID and bin step
func GetPriceFromID(activeID int32, binStep uint16) (uint128.Uint128, error) {
	// Convert binStep to uint128
	bps := uint128.From64(uint64(binStep))

	// Calculate bps << SCALE_OFFSET
	shiftedBps := bps.Lsh(uint(ScaleOffset))

	// Divide by BASIS_POINT_MAX
	if basisPointMax := uint128.From64(BasisPointMax); !basisPointMax.IsZero() {
		bps = shiftedBps.Div(basisPointMax)
	} else {
		return uint128.Zero, fmt.Errorf("division by zero")
	}

	// Calculate base = ONE + bps
	// One = 1 << SCALE_OFFSET, use direct shift operation
	base := One.Add(bps)

	// Calculate base^activeID
	result, err := Pow(base, activeID)
	if err != nil {
		return uint128.Zero, fmt.Errorf("power calculation error: %w", err)
	}

	return result, nil
}

// GetBinArrayLowerUpperBinID calculates the lower and upper bin IDs for a bin array index
func GetBinArrayLowerUpperBinID(index int32) (int32, int32, error) {
	// Calculate lower bound
	lowerBinID := index * int32(MaxBinPerArray)

	// Calculate upper bound
	temp := lowerBinID + int32(MaxBinPerArray)

	upperBinID := temp - 1

	return lowerBinID, upperBinID, nil
}

// Pow calculates base raised to the power of exponent
func Pow(base uint128.Uint128, power int32) (uint128.Uint128, error) {
	// Handle special cases
	if power == 0 {
		return One, nil
	}

	// Handle negative exponent
	isNegative := power < 0
	if isNegative {
		power = -power
	}

	// Calculate result
	result := One
	current := base
	exp := uint32(power)

	for exp > 0 {
		if exp&1 == 1 {
			// Check for multiplication overflow
			if result.Hi > 0 && current.Hi > 0 {
				return uint128.Zero, fmt.Errorf("multiplication overflow")
			}
			result = result.Mul(current)
		}
		exp >>= 1
		if exp > 0 {
			// Check for square overflow
			if current.Hi > 0 {
				return uint128.Zero, fmt.Errorf("square overflow")
			}
			current = current.Mul(current)
		}
	}

	// If negative exponent, need to calculate reciprocal
	if isNegative {
		// For negative exponent, we need to calculate reciprocal: 1/result
		// This requires precise division implementation
		return uint128.Zero, fmt.Errorf("negative power not implemented")
	}

	return result, nil
}

// GetBitmapOffset calculates the bitmap offset for a bin array index
func GetBitmapOffset(binArrayIndex int32) (int, error) {
	var offset int32
	if binArrayIndex > 0 {
		offset = binArrayIndex/BinArrayBitmapSize - 1
	} else {
		offset = -(binArrayIndex+1)/BinArrayBitmapSize - 1
	}

	return int(offset), nil
}

// BinArrayOffsetInBitmap calculates the offset of a bin array within the bitmap
func BinArrayOffsetInBitmap(binArrayIndex int32) (int, error) {
	if binArrayIndex > 0 {
		// For positive numbers, take modulo directly
		if remainder := binArrayIndex % BinArrayBitmapSize; remainder >= 0 {
			return int(remainder), nil
		} else {
			return 0, fmt.Errorf("overflow in positive modulo operation")
		}
	} else {
		// For negative numbers, first negate binArrayIndex + 1, then take modulo
		negIndex := -(binArrayIndex + 1)
		if remainder := negIndex % BinArrayBitmapSize; remainder >= 0 {
			return int(remainder), nil
		} else {
			return 0, fmt.Errorf("overflow in negative modulo operation")
		}
	}
}

// ArrayToBigInt converts an array of 8 uint64 values to a big.Int
func ArrayToBigInt(arr [8]uint64) *big.Int {
	result := new(big.Int)
	// From high to low, shift left by 64 bits each time and add new limb
	for i := 0; i < 8; i++ {
		temp := new(big.Int).SetUint64(arr[i])
		result.Lsh(result, 64)  // Shift left by 64 bits
		result.Or(result, temp) // OR operation, add new limb
	}
	return result
}

// CountLeadingZeros counts the number of leading zeros in a big.Int
func CountLeadingZeros(n *big.Int) int {
	if n.Sign() == 0 { // If 0, all bits are leading zeros
		return BinArrayBitmapSize
	}

	// BitLen() returns the minimum number of bits needed to store this number
	// Subtract actual needed bits from total bits to get leading zero count
	bits := n.BitLen()
	return BinArrayBitmapSize - bits
}

// ToBinArrayIndex converts offset and bin array offset to bin array index
func ToBinArrayIndex(offset, binArrayOffset int, isPositive bool) (*int32, error) {
	// Convert to int32
	offsetInt32 := int32(offset)
	binArrayOffsetInt32 := int32(binArrayOffset)

	if isPositive {
		// For positive case
		res := (offsetInt32+1)*BinArrayBitmapSize + binArrayOffsetInt32
		return &res, nil
	} else {
		// For negative case
		res := -((offsetInt32+1)*BinArrayBitmapSize + binArrayOffsetInt32) - 1
		return &res, nil
	}
}

// CountTrailingZeros counts the number of trailing zeros in a big.Int
func CountTrailingZeros(n *big.Int) int {
	if n.Sign() == 0 { // If 0, all bits are 0
		return BinArrayBitmapSize
	}

	// Count from lowest bit until first 1 is found
	count := 0
	temp := new(big.Int).Set(n) // Create a copy to avoid modifying original value

	// Check each bit until first 1 is found
	for temp.Bit(count) == 0 {
		count++
		if count >= BinArrayBitmapSize {
			break
		}
	}

	return count
}
