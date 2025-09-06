package raydium

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	cosmath "cosmossdk.io/math"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"lukechampine.com/uint128"
)

type TickArrayBitmapExtensionType struct {
	PoolId                  solana.PublicKey
	PositiveTickArrayBitmap [][]uint64
	NegativeTickArrayBitmap [][]uint64
}

type TickArray struct {
	_                    [8]byte          `bin:"skip"`         // padding
	PoolId               solana.PublicKey `bin:"fixed"`        // 32 bytes
	StartTickIndex       int32            `bin:"le"`           // 4 bytes
	Ticks                []TickState      `bin:"array,len=60"` // TICK_ARRAY_SIZE=60
	InitializedTickCount uint8            // 1 byte
	_                    [115]byte        `bin:"skip"` // padding
}

type TickState struct {
	Tick                    int32              `bin:"le"`   // 4 bytes
	LiquidityNet            int64              `bin:"le"`   // 8 bytes
	_                       [8]byte            `bin:"skip"` // skip high 8 bytes
	LiquidityGross          uint128.Uint128    `bin:"le"`   // 16 bytes
	FeeGrowthOutsideX64A    uint128.Uint128    `bin:"le"`   // 16 bytes
	FeeGrowthOutsideX64B    uint128.Uint128    `bin:"le"`   // 16 bytes
	RewardGrowthsOutsideX64 [3]uint128.Uint128 `bin:"le"`   // 48 bytes
	_                       [52]byte           `bin:"skip"` // padding
}

// Decode decodes the tick array data
func (t *TickArray) Decode(data []byte) error {
	decoder := bin.NewBinDecoder(data)

	// Decode initial padding
	var padding [8]byte
	if err := decoder.Decode(&padding); err != nil {
		return fmt.Errorf("failed to decode initial padding: %w", err)
	}

	// Decode PoolId
	if err := decoder.Decode(&t.PoolId); err != nil {
		return fmt.Errorf("failed to decode poolId: %w", err)
	}

	// Decode StartTickIndex
	if err := decoder.Decode(&t.StartTickIndex); err != nil {
		return fmt.Errorf("failed to decode startTickIndex: %w", err)
	}

	// Calculate first tick position
	tickStartPos := 8 + 32 + 4 // padding + poolId + startTickIndex

	t.Ticks = make([]TickState, TICK_ARRAY_SIZE)
	for i := 0; i < TICK_ARRAY_SIZE; i++ {
		tick := int32(binary.LittleEndian.Uint32(data[tickStartPos:]))
		tickStartPos += 4

		liquidityNet := int64(binary.LittleEndian.Uint64(data[tickStartPos:]))
		tickStartPos += 16 // Skip 16 bytes

		liquidityGross := parseUint128LE(data[tickStartPos:])
		tickStartPos += 16

		feeGrowthOutsideX64A := parseUint128LE(data[tickStartPos:])
		tickStartPos += 16

		feeGrowthOutsideX64B := parseUint128LE(data[tickStartPos:])
		tickStartPos += 16

		var rewards [3]uint128.Uint128
		for j := 0; j < 3; j++ {
			rewards[j] = parseUint128LE(data[tickStartPos:])
			tickStartPos += 16
		}

		tickStartPos += 52

		t.Ticks[i] = TickState{
			Tick:                    tick,
			LiquidityNet:            liquidityNet,
			LiquidityGross:          liquidityGross,
			FeeGrowthOutsideX64A:    feeGrowthOutsideX64A,
			FeeGrowthOutsideX64B:    feeGrowthOutsideX64B,
			RewardGrowthsOutsideX64: rewards,
		}
	}

	// Decode InitializedTickCount
	t.InitializedTickCount = data[tickStartPos]

	return nil
}

// GetTickArrayAddresses returns the addresses of tick arrays
func (p *CLMMPool) GetTickArrayAddresses() ([]solana.PublicKey, error) {
	startIndexArray := p.getInitializedTickArrayInRange(10) // Get 10 tick arrays
	tickArrayAddresses := make([]solana.PublicKey, 0, len(startIndexArray))
	for _, itemIndex := range startIndexArray {
		tickArrayAddress := getPdaTickArrayAddress(RAYDIUM_CLMM_PROGRAM_ID, p.PoolId, itemIndex)
		tickArrayAddresses = append(tickArrayAddresses, tickArrayAddress)
	}
	return tickArrayAddresses, nil
}

// FetchPoolTickArrays fetches tick arrays for the pool
func (p *CLMMPool) FetchPoolTickArrays(ctx context.Context, client *rpc.Client) error {
	tickArrayAddresses, err := p.GetTickArrayAddresses()
	if err != nil {
		return fmt.Errorf("get tick array address error: %v", err)
	}
	accounts, err := client.GetMultipleAccounts(ctx, tickArrayAddresses...)
	if err != nil {
		return fmt.Errorf("get accounts error: %v", err)
	}

	p.TickArrayCache = make(map[string]TickArray)
	for _, account := range accounts.Value {
		if account == nil {
			continue
		}
		tickArray := &TickArray{}
		err := tickArray.Decode(account.Data.GetBinary())
		if err != nil {
			return fmt.Errorf("failed to decode tick array: %w", err)
		}
		p.TickArrayCache[strconv.FormatInt(int64(tickArray.StartTickIndex), 10)] = *tickArray
	}

	return nil
}

// ParseExBitmapInfo parses the extended bitmap information
func (p *CLMMPool) ParseExBitmapInfo(data []byte) {
	var bitmap TickArrayBitmapExtensionType

	// Skip 8-byte discriminator
	data = data[8:]

	// Parse poolId (32 bytes)
	copy(bitmap.PoolId[:], data[:32])
	data = data[32:]

	// Parse positiveTickArrayBitmap
	positiveBitmaps := make([][]uint64, EXTENSION_TICKARRAY_BITMAP_SIZE)
	for i := 0; i < EXTENSION_TICKARRAY_BITMAP_SIZE; i++ {
		arr := make([]uint64, 8)
		for j := 0; j < 8; j++ {
			arr[j] = binary.LittleEndian.Uint64(data[j*8 : (j+1)*8])
		}
		positiveBitmaps[i] = arr
		data = data[64:] // Skip 8 uint64s (64 bytes)
	}
	bitmap.PositiveTickArrayBitmap = positiveBitmaps

	// Parse negativeTickArrayBitmap
	negativeBitmaps := make([][]uint64, EXTENSION_TICKARRAY_BITMAP_SIZE)
	for i := 0; i < EXTENSION_TICKARRAY_BITMAP_SIZE; i++ {
		arr := make([]uint64, 8)
		for j := 0; j < 8; j++ {
			arr[j] = binary.LittleEndian.Uint64(data[j*8 : (j+1)*8])
		}
		negativeBitmaps[i] = arr
		data = data[64:] // Skip 8 uint64s (64 bytes)
	}
	bitmap.NegativeTickArrayBitmap = negativeBitmaps

	p.exTickArrayBitmap = &bitmap
}

// getInitializedTickArrayInRange returns initialized tick arrays in range
func (p *CLMMPool) getInitializedTickArrayInRange(count int64) []int64 {
	tickArrayBitmap := p.TickArrayBitmap
	exBitmapInfo := p.exTickArrayBitmap
	tickSpacing := int64(p.TickSpacing)

	tickArrayStartIndex := getTickArrayStartIndexByTick(int64(p.TickCurrent), int64(p.TickSpacing))
	tickArrayOffset := math.Floor(float64(tickArrayStartIndex) / (float64(tickSpacing) * float64(TICK_ARRAY_SIZE)))

	result := make([]int64, 0, count)
	r := SearchLowBitFromStart(tickArrayBitmap, exBitmapInfo, int64(tickArrayOffset-1), count, tickSpacing)
	result = append(result, r...)
	r = SearchHighBitFromStart(tickArrayBitmap, exBitmapInfo, int64(tickArrayOffset-1), count, tickSpacing)
	result = append(result, r...)

	return result
}

// nextInitializedTickArray finds the next initialized tick array
func nextInitializedTickArray(
	tick int64,
	tickSpacing int64,
	zeroForOne bool,
	tickArrayBitmap [16]uint64,
	tickarrayBitmapExtension *TickArrayBitmapExtensionType,
) (int64, bool) {
	currentOffset := math.Floor(float64(tick) / float64(getTickCount(tickSpacing)))
	var result []int64
	if zeroForOne {
		result = SearchLowBitFromStart(tickArrayBitmap, tickarrayBitmapExtension, int64(currentOffset-1), 1, tickSpacing)
	} else {
		result = SearchHighBitFromStart(tickArrayBitmap, tickarrayBitmapExtension, int64(currentOffset+1), 1, tickSpacing)
	}
	if len(result) > 0 {
		return result[0], true
	}
	return 0, false
}

// checkTickArrayIsInitialized checks if a tick array is initialized
func checkTickArrayIsInitialized(tickArrayBitmap [16]uint64, tick int64, tickSpacing int64) bool {
	multiplier := tickSpacing * TICK_ARRAY_SIZE
	compressed := tick/multiplier + 512
	bitPos := int(math.Abs(float64(compressed)))

	wordPos := bitPos / 64
	if wordPos >= len(tickArrayBitmap) {
		return false
	}

	bitPosInWord := uint(bitPos % 64)
	return (tickArrayBitmap[wordPos] & (1 << bitPosInWord)) != 0
}

// getTickArrayBitIndex gets the bit index of a tick array
func getTickArrayBitIndex(tickIndex int64, tickSpacing int64) int64 {
	ticksInArray := getTickCount(tickSpacing)
	startIndex := float64(tickIndex) / float64(ticksInArray)

	if tickIndex < 0 && tickIndex%ticksInArray != 0 {
		return int64(math.Ceil(startIndex) - 1)
	}
	return int64(math.Floor(startIndex))
}

// getTickCount returns the number of ticks in an array
func getTickCount(tickSpacing int64) int64 {
	return tickSpacing * TICK_ARRAY_SIZE
}

func SearchLowBitFromStart(
	tickArrayBitmap [16]uint64,
	exTickArrayBitmap *TickArrayBitmapExtensionType,
	currentTickArrayBitStartIndex int64,
	expectedCount int64,
	tickSpacing int64) []int64 {

	var tickArrayBitmaps []*big.Int

	for i := len(exTickArrayBitmap.NegativeTickArrayBitmap) - 1; i >= 0; i-- {
		tickArrayBitmaps = append(tickArrayBitmaps, MergeTickArrayBitmap(exTickArrayBitmap.NegativeTickArrayBitmap[i]))
	}
	tickArrayBitmaps = append(tickArrayBitmaps, MergeTickArrayBitmap(tickArrayBitmap[0:8]))
	tickArrayBitmaps = append(tickArrayBitmaps, MergeTickArrayBitmap(tickArrayBitmap[8:16]))
	for _, bitmap := range exTickArrayBitmap.PositiveTickArrayBitmap {
		tickArrayBitmaps = append(tickArrayBitmaps, MergeTickArrayBitmap(bitmap))
	}

	result := make([]int64, 0)
	for currentTickArrayBitStartIndex >= -7680 {
		arrayIndex := (currentTickArrayBitStartIndex + 7680) / 512
		searchIndex := (currentTickArrayBitStartIndex + 7680) % 512

		if tickArrayBitmaps[arrayIndex].Bit(int(searchIndex)) == 1 {
			result = append(result, (currentTickArrayBitStartIndex))
		}

		currentTickArrayBitStartIndex--
		if len(result) == int(expectedCount) {
			break
		}
	}

	tickCount := getTickCount(int64(tickSpacing))
	finalResult := make([]int64, len(result))
	for i, val := range result {
		finalResult[i] = int64(val) * tickCount
	}

	return finalResult
}

// SearchHighBitFromStart searches for high bits from start
func SearchHighBitFromStart(tickArrayBitmap [16]uint64,
	exTickArrayBitmap *TickArrayBitmapExtensionType,
	currentTickArrayBitStartIndex int64,
	expectedCount int64,
	tickSpacing int64) []int64 {

	var tickArrayBitmaps []*big.Int

	for i := len(exTickArrayBitmap.NegativeTickArrayBitmap) - 1; i >= 0; i-- {
		tickArrayBitmaps = append(tickArrayBitmaps, MergeTickArrayBitmap(exTickArrayBitmap.NegativeTickArrayBitmap[i]))
	}

	firstPart := MergeTickArrayBitmap(tickArrayBitmap[0:8])
	secondPart := MergeTickArrayBitmap(tickArrayBitmap[8:16])
	tickArrayBitmaps = append(tickArrayBitmaps, firstPart, secondPart)

	for _, bitmap := range exTickArrayBitmap.PositiveTickArrayBitmap {
		tickArrayBitmaps = append(tickArrayBitmaps, MergeTickArrayBitmap(bitmap))
	}

	result := make([]int64, 0)
	for currentTickArrayBitStartIndex < 7680 {
		arrayIndex := (currentTickArrayBitStartIndex + 7680) / 512
		searchIndex := (currentTickArrayBitStartIndex + 7680) % 512

		if tickArrayBitmaps[arrayIndex].Bit(int(searchIndex)) == 1 {
			result = append(result, currentTickArrayBitStartIndex)
		}

		currentTickArrayBitStartIndex++
		if len(result) == int(expectedCount) {
			break
		}
	}

	tickCount := getTickCount(tickSpacing)
	finalResult := make([]int64, len(result))
	for i, val := range result {
		finalResult[i] = val * tickCount
	}

	return finalResult
}

// TickArrayOffsetInBitmap calculates the offset of a tick array in bitmap
func TickArrayOffsetInBitmap(tickArrayStartIndex int64, tickSpacing int64) int64 {
	maxTick := MaxTickInTickarrayBitmap(tickSpacing)
	m := abs(tickArrayStartIndex) % maxTick
	tickArrayOffsetInBitmap := m / getTickCount(int64(tickSpacing))

	if tickArrayStartIndex < 0 && m != 0 {
		tickArrayOffsetInBitmap = TICK_ARRAY_BITMAP_SIZE - tickArrayOffsetInBitmap
	}

	return tickArrayOffsetInBitmap
}

// MaxTickInTickarrayBitmap returns the maximum tick in tick array bitmap
func MaxTickInTickarrayBitmap(tickSpacing int64) int64 {
	return TICK_ARRAY_BITMAP_SIZE * getTickCount(tickSpacing)
}

// abs returns the absolute value of an integer
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// GetBitmapTickBoundary returns the tick boundary of a bitmap
func GetBitmapTickBoundary(tickarrayStartIndex int64, tickSpacing int64) (int64, int64) {
	ticksInOneBitmap := MaxTickInTickarrayBitmap(tickSpacing)
	m := abs(tickarrayStartIndex) / ticksInOneBitmap

	if tickarrayStartIndex < 0 && abs(tickarrayStartIndex)%ticksInOneBitmap != 0 {
		m += 1
	}

	minValue := ticksInOneBitmap * m

	if tickarrayStartIndex < 0 {
		return -minValue, -minValue + ticksInOneBitmap
	}

	return minValue, minValue + ticksInOneBitmap
}

// GetBitmap returns the bitmap for a given tick index
func GetBitmap(tickIndex int64, tickSpacing int64,
	tickArrayBitmapExtension *TickArrayBitmapExtensionType) (int64, []uint64, error) {
	offset, err := GetBitmapOffset(tickIndex, tickSpacing)
	if err != nil {
		return -1, nil, err
	}

	if tickIndex < 0 {
		return offset, tickArrayBitmapExtension.NegativeTickArrayBitmap[offset], nil
	}
	return offset, tickArrayBitmapExtension.PositiveTickArrayBitmap[offset], nil
}

// GetBitmapOffset returns the bitmap offset for a given tick index
func GetBitmapOffset(tickIndex int64, tickSpacing int64) (int64, error) {
	if !checkIsValidStartIndex(tickIndex, tickSpacing) {
		return 0, fmt.Errorf("no enough initialized tickArray")
	}

	if err := checkExtensionBoundary(tickIndex, tickSpacing); err != nil {
		return 0, err
	}

	ticksInOneBitmap := MaxTickInTickarrayBitmap(tickSpacing)
	offset := abs(tickIndex)/ticksInOneBitmap - 1

	if tickIndex < 0 && abs(tickIndex)%ticksInOneBitmap == 0 {
		offset--
	}

	return offset, nil
}

// checkExtensionBoundary checks if a tick index is within extension boundary
func checkExtensionBoundary(tickIndex int64, tickSpacing int64) error {
	NegativeTickBoundary, PositiveTickBoundary, err := ExtensionTickBoundary(tickSpacing)
	if err != nil {
		return err
	}

	if tickIndex >= NegativeTickBoundary && tickIndex < PositiveTickBoundary {
		return fmt.Errorf("checkExtensionBoundary -> InvalidTickArrayBoundary")
	}

	return nil
}

// ExtensionTickBoundary returns the tick boundaries for extension
func ExtensionTickBoundary(tickSpacing int64) (int64, int64, error) {
	positiveTickBoundary := MaxTickInTickarrayBitmap(tickSpacing)
	negativeTickBoundary := -positiveTickBoundary

	if MAX_TICK <= positiveTickBoundary {
		return 0, 0, fmt.Errorf("extensionTickBoundary check error: %d, %d", MAX_TICK, positiveTickBoundary)
	}
	if negativeTickBoundary <= MIN_TICK {
		return 0, 0, fmt.Errorf("extensionTickBoundary check error: %d, %d", negativeTickBoundary, MIN_TICK)
	}

	return positiveTickBoundary, negativeTickBoundary, nil
}

func nextInitializedTickArrayStartIndexUtils(exTickArrayBitmap *TickArrayBitmapExtensionType, tickCurrent, tickSpacing int64, tickArrayBitmap [16]uint64,
	zeroForOne bool) (bool, int64, error) {
	lastTickArrayStartIndex := GetArrayStartIndex(tickCurrent, tickSpacing)

	// eslint-disable-next-line no-constant-condition
	for {
		startIsInit, startIndex := nextInitializedTickArrayStartIndex(
			MergeTickArrayBitmap(tickArrayBitmap[:]),
			int64(lastTickArrayStartIndex),
			int64(tickSpacing),
			zeroForOne,
		)
		if startIsInit {
			return true, startIndex, nil
		}
		lastTickArrayStartIndex = startIndex
		isInit, tickIndex, err := nextInitializedTickArrayFromOneBitmap(
			lastTickArrayStartIndex,
			int64(tickSpacing),
			zeroForOne,
			exTickArrayBitmap,
		)
		if err != nil {
			return false, 0, err
		}
		if isInit {
			return true, tickIndex, nil
		}
		lastTickArrayStartIndex = tickIndex

		if lastTickArrayStartIndex < MIN_TICK || lastTickArrayStartIndex > MAX_TICK {
			return false, 0, errors.New("error: out of range")
		}
	}
}

func nextInitializedTickArrayFromOneBitmap(lastTickArrayStartIndex, tickSpacing int64,
	zeroForOne bool, tickArrayBitmapExtension *TickArrayBitmapExtensionType) (bool, int64, error) {
	multiplier := getTickCount(tickSpacing)
	nextTickArrayStartIndex := int64(0)
	if zeroForOne {
		nextTickArrayStartIndex = lastTickArrayStartIndex - multiplier
	} else {
		nextTickArrayStartIndex = lastTickArrayStartIndex + multiplier
	}

	_, tickarrayBitmap, err := GetBitmap(nextTickArrayStartIndex, tickSpacing, tickArrayBitmapExtension)
	if err != nil {
		return false, 0, err
	}

	// nextInitializedTickArrayInBitmap
	bitmapMinTickBoundary, bitmapMaxTickBoundary := GetBitmapTickBoundary(nextTickArrayStartIndex, tickSpacing)

	tickArrayOffsetInBitmap := TickArrayOffsetInBitmap(nextTickArrayStartIndex, tickSpacing)
	if zeroForOne {
		mergedBitmap := MergeTickArrayBitmap(tickarrayBitmap)
		offsetBitMap := new(big.Int).Set(mergedBitmap)
		offsetBitMap.Lsh(offsetBitMap, uint(TICK_ARRAY_BITMAP_SIZE-1-tickArrayOffsetInBitmap))

		if IsZero(512, offsetBitMap) || LeadingZeros(512, offsetBitMap) == nil {
			return false, bitmapMinTickBoundary, nil
		}
		if !IsZero(512, offsetBitMap) && LeadingZeros(512, offsetBitMap) != nil {
			nextBit := *LeadingZeros(512, offsetBitMap)
			nextArrayStartIndex := nextTickArrayStartIndex - int64(nextBit)*getTickCount(tickSpacing)
			return true, nextArrayStartIndex, nil
		}
	} else {
		mergedBitmap := MergeTickArrayBitmap(tickarrayBitmap)
		offsetBitMap := new(big.Int).Set(mergedBitmap)
		offsetBitMap.Rsh(offsetBitMap, uint(tickArrayOffsetInBitmap))

		if !IsZero(512, offsetBitMap) && TrailingZeros(512, offsetBitMap) != nil {
			nextBit := *TrailingZeros(512, offsetBitMap)
			nextArrayStartIndex := nextTickArrayStartIndex + int64(nextBit)*getTickCount(tickSpacing)
			return true, nextArrayStartIndex, nil
		}
		if IsZero(512, offsetBitMap) || TrailingZeros(512, offsetBitMap) == nil {
			return false, bitmapMaxTickBoundary - getTickCount(tickSpacing), nil

		}
	}
	return false, 0, errors.New("nextInitializedTickArrayFromOneBitmap error: out of range")
}

// nextInitializedTickArrayStartIndex 获取下一个初始化的 tick array 起始索引
func nextInitializedTickArrayStartIndex(bitMap *big.Int,
	lastTickArrayStartIndex int64, tickSpacing int64, zeroForOne bool) (bool, int64) {

	if !checkIsValidStartIndex(lastTickArrayStartIndex, tickSpacing) {
		panic("invalid start index")
	}

	tickBoundary := maxTickInTickarrayBitmap(tickSpacing)
	nextTickArrayStartIndex := int64(0)
	if zeroForOne {
		nextTickArrayStartIndex = lastTickArrayStartIndex - getTickCount(tickSpacing)
	} else {
		nextTickArrayStartIndex = lastTickArrayStartIndex + getTickCount(tickSpacing)
	}

	if nextTickArrayStartIndex < -tickBoundary || nextTickArrayStartIndex >= tickBoundary {
		return false, lastTickArrayStartIndex
	}

	multiplier := int64(tickSpacing) * TICK_ARRAY_SIZE
	compressed := float64(nextTickArrayStartIndex)/float64(multiplier) + 512
	if nextTickArrayStartIndex < 0 && nextTickArrayStartIndex%multiplier != 0 {
		compressed--
	}

	bitPos := int(math.Abs(compressed))
	// mergedBitmap := mergeBitmap([16]uint64{})

	if zeroForOne {
		// 向下搜索
		offsetBitMap := new(big.Int).Set(bitMap)
		offsetBitMap.Lsh(offsetBitMap, uint(1024-bitPos-1))
		nextBit := MostSignificantBit(1024, offsetBitMap)
		if nextBit != nil {
			nextArrayStartIndex := int64(bitPos-*nextBit-512) * multiplier
			return true, nextArrayStartIndex
		} else {
			return false, -tickBoundary
		}
	} else {
		// 向上搜索
		offsetBitMap := new(big.Int).Set(bitMap)
		offsetBitMap.Rsh(offsetBitMap, uint(bitPos))
		nextBit := LeastSignificantBit(1024, offsetBitMap)
		if nextBit != nil {
			nextArrayStartIndex := int64(bitPos+*nextBit-512) * multiplier
			return true, nextArrayStartIndex
		}
		return false, tickBoundary - getTickCount(int64(tickSpacing))
	}
}

// isOverflowDefaultTickarrayBitmap 检查是否超出默认 bitmap 范围
func isOverflowDefaultTickarrayBitmap(tickSpacing int64, tickarrayStartIndexs []int64) bool {
	tickRange := tickRange(tickSpacing)
	maxTickBoundary := tickRange.maxTickBoundary
	minTickBoundary := tickRange.minTickBoundary

	for _, tickIndex := range tickarrayStartIndexs {
		tickarrayStartIndex := getTickArrayStartIndexByTick(tickIndex, tickSpacing)
		if tickarrayStartIndex >= maxTickBoundary || tickarrayStartIndex < minTickBoundary {
			return true
		}
	}
	return false
}

// tickRange 获取 tick 范围
func tickRange(tickSpacing int64) struct {
	maxTickBoundary int64
	minTickBoundary int64
} {
	maxTickBoundary := maxTickInTickarrayBitmap(tickSpacing)
	minTickBoundary := -maxTickBoundary

	if maxTickBoundary > MAX_TICK {
		maxTickBoundary = getTickArrayStartIndex(MAX_TICK, tickSpacing) + getTickCount(tickSpacing)
	}
	if minTickBoundary < MIN_TICK {
		minTickBoundary = getTickArrayStartIndex(MIN_TICK, tickSpacing)
	}
	return struct {
		maxTickBoundary int64
		minTickBoundary int64
	}{
		maxTickBoundary: maxTickBoundary,
		minTickBoundary: minTickBoundary,
	}
}

// checkTickArrayIsInitExtension 检查扩展 bitmap 中的 tick array 是否已初始化
func checkTickArrayIsInit(tick int64, tickSpacing int64, exBitmapInfo *TickArrayBitmapExtensionType) bool {
	arrayIndex := getTickArrayBitIndex(tick, tickSpacing)
	isPositive := arrayIndex >= 0

	var targetBitmap [][]uint64
	if isPositive {
		targetBitmap = exBitmapInfo.PositiveTickArrayBitmap
		arrayIndex = int64(math.Abs(float64(arrayIndex)))
	} else {
		targetBitmap = exBitmapInfo.NegativeTickArrayBitmap
		arrayIndex = int64(math.Abs(float64(arrayIndex))) - 1
	}

	wordPos := arrayIndex / 64
	bitPos := arrayIndex % 64

	if int(wordPos) >= len(targetBitmap) {
		return false
	}

	word := targetBitmap[wordPos]
	if len(word) == 0 {
		return false
	}

	// 检查对应位是否为 1
	return (word[0] & (1 << uint(bitPos))) != 0
}

// checkIsValidStartIndex 检查起始索引是否有效
func checkIsValidStartIndex(startIndex int64, tickSpacing int64) bool {
	return startIndex%getTickCount(tickSpacing) == 0
}

// maxTickInTickarrayBitmap 获取 bitmap 中最大的 tick
func maxTickInTickarrayBitmap(tickSpacing int64) int64 {
	return TICK_ARRAY_BITMAP_SIZE * getTickCount(tickSpacing)
}

// getTickArrayStartIndex 获取 tick array 的起始索引
func getTickArrayStartIndex(tick int64, tickSpacing int64) int64 {
	return tick - tick%getTickCount(tickSpacing)
}

func MostSignificantBit(bitNum int, data *big.Int) *int {
	// 检查是否为零
	if IsZero(bitNum, data) {
		return nil
	}
	// 返回前导零的数量
	return LeadingZeros(bitNum, data)
}

// TrailingZeros 计算尾随零的数量
func TrailingZeros(bitNum int, data *big.Int) *int {
	var count int
	// 从最低位向高位遍历
	for j := 0; j < bitNum; j++ {
		if data.Bit(j) == 0 {
			count++
		} else {
			break
		}
	}
	return &count
}

func GetArrayStartIndex(tickIndex int64, tickSpacing int64) int64 {
	ticksInArray := getTickCount(int64(tickSpacing))
	start := math.Floor(float64(tickIndex) / float64(ticksInArray)) // Go 的整数除法会自动向下取整，相当于 Math.floor
	return int64(start * float64(ticksInArray))
}

func MergeTickArrayBitmap(bns []uint64) *big.Int {
	result := new(big.Int)

	// 遍历数组
	for i, bn := range bns {
		// Convert uint64 to big.Int
		bnBig := new(big.Int).SetUint64(bn)
		// Create a temporary big.Int for the shift operation
		shifted := new(big.Int).Lsh(bnBig, uint(64*i))
		result.Add(result, shifted)
	}

	return result
}

// firstInitializedTick 查找第一个初始化的价格刻度
func firstInitializedTick(tickArrayCurrent *TickArray, zeroForOne bool) (*TickState, error) {
	if tickArrayCurrent == nil || len(tickArrayCurrent.Ticks) == 0 {
		return nil, fmt.Errorf("invalid tick array")
	}

	if zeroForOne {
		// 从后向前遍历
		for i := TICK_ARRAY_SIZE - 1; i >= 0; i-- {
			if i >= len(tickArrayCurrent.Ticks) {
				continue
			}
			// 检查 liquidityGross 是否大于 0
			if tickArrayCurrent.Ticks[i].LiquidityGross.Big().Cmp(big.NewInt(0)) > 0 {
				return &tickArrayCurrent.Ticks[i], nil
			}
		}
	} else {
		// 从前向后遍历
		for i := 0; i < TICK_ARRAY_SIZE && i < len(tickArrayCurrent.Ticks); i++ {
			// 检查 liquidityGross 是否大于 0
			if tickArrayCurrent.Ticks[i].LiquidityGross.Big().Cmp(big.NewInt(0)) > 0 {
				return &tickArrayCurrent.Ticks[i], nil
			}
		}
	}

	return nil, fmt.Errorf("firstInitializedTick check error: no initialized tick found")
}

func getNextInitTick(
	tickArrayCurrent *TickArray,
	currentTickIndex int64,
	tickSpacing int64,
	zeroForOne bool,
	t bool,
) *TickState {

	currentTickArrayStartIndex := GetArrayStartIndex(currentTickIndex, tickSpacing)
	if currentTickArrayStartIndex != int64(tickArrayCurrent.StartTickIndex) {
		return nil
	}
	offsetInArray := (currentTickIndex - int64(tickArrayCurrent.StartTickIndex)) / tickSpacing
	if zeroForOne {
		for offsetInArray >= 0 {
			if tickArrayCurrent.Ticks[offsetInArray].LiquidityGross.Big().Cmp(big.NewInt(0)) > 0 {
				return &tickArrayCurrent.Ticks[offsetInArray]
			}
			offsetInArray = offsetInArray - 1
		}
	} else {
		if !t {
			offsetInArray = offsetInArray + 1
		}
		for offsetInArray < TICK_ARRAY_SIZE {
			if tickArrayCurrent.Ticks[offsetInArray].LiquidityGross.Big().Cmp(big.NewInt(0)) > 0 {
				return &tickArrayCurrent.Ticks[offsetInArray]
			}
			offsetInArray = offsetInArray + 1
		}
	}
	return nil
}

// getFirstInitializedTickArray 获取第一个初始化的 tick array
func (poolInfo *CLMMPool) getFirstInitializedTickArray(zeroForOne bool, exTickArrayBitmap *TickArrayBitmapExtensionType) (int64, solana.PublicKey, error) {

	// 1. 计算当前 tick 所在的 tick array 起始索引
	startIndex := getTickArrayStartIndexByTick(int64(poolInfo.TickCurrent), int64(poolInfo.TickSpacing))

	// 2. 检查该 tick array 是否已初始化
	isInitialized := false
	if isOverflowDefaultTickarrayBitmap(int64(poolInfo.TickSpacing), []int64{int64(poolInfo.TickCurrent)}) {
		isInitialized = checkTickArrayIsInit(
			GetArrayStartIndex(int64(poolInfo.TickCurrent), int64(poolInfo.TickSpacing)),
			int64(poolInfo.TickSpacing),
			exTickArrayBitmap)
	} else {
		isInitialized = checkTickArrayIsInitialized(poolInfo.TickArrayBitmap, int64(poolInfo.TickCurrent), int64(poolInfo.TickSpacing))
	}

	if isInitialized {
		// 3. 如果已初始化，获取其 PDA 地址
		address := getPdaTickArrayAddress(
			RAYDIUM_CLMM_PROGRAM_ID,
			poolInfo.PoolId,
			startIndex,
		)
		return startIndex, address, nil
	}

	// 4. 如果未初始化，获取下一个初始化的 tick array
	isExist, nextStartIndex, err := nextInitializedTickArrayStartIndexUtils(
		exTickArrayBitmap,
		int64(poolInfo.TickCurrent),
		int64(poolInfo.TickSpacing),
		poolInfo.TickArrayBitmap,
		zeroForOne,
	)
	if err != nil {
		return 0, solana.PublicKey{}, err
	}
	if isExist {
		address := getPdaTickArrayAddress(
			RAYDIUM_CLMM_PROGRAM_ID,
			poolInfo.PoolId,
			nextStartIndex,
		)
		return nextStartIndex, address, err
	}
	return startIndex, solana.PublicKey{}, nil
}

// getPdaTickArrayAddress 获取 tick array 的 PDA 地址
func getPdaTickArrayAddress(programId solana.PublicKey, poolId solana.PublicKey, startIndex int64) solana.PublicKey {
	startIndexBytes := i32ToBytes(startIndex)
	seeds := [][]byte{
		[]byte("tick_array"), poolId.Bytes(), startIndexBytes,
	}
	pk, _, _ := solana.FindProgramAddress(seeds, programId)
	return pk
}

func GetPdaExBitmapAccount(programId solana.PublicKey, id solana.PublicKey) (solana.PublicKey, uint8, error) {
	seeds := [][]byte{
		[]byte("pool_tick_array_bitmap_extension"),
		id.Bytes(),
	}
	return solana.FindProgramAddress(seeds, programId)
}

func getTickArrayStartIndexByTick(tickIndex int64, tickSpacing int64) int64 {
	return getTickArrayBitIndex(tickIndex, tickSpacing) * getTickCount(tickSpacing)
}

func i32ToBytes(num int64) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(num))
	return b
}

// 添加一个辅助函数来正确解析 uint128
func parseUint128LE(data []byte) uint128.Uint128 {
	lo := binary.LittleEndian.Uint64(data[:8])
	hi := binary.LittleEndian.Uint64(data[8:])
	return uint128.New(lo, hi)
}

// leastSignificantBit 获取最低有效位
// LeastSignificantBit 找到最低位的1的位置
func LeastSignificantBit(bitNum int, data *big.Int) *int {
	// 检查是否为零
	if IsZero(bitNum, data) {
		return nil
	}
	// 返回尾随零的数量
	return TrailingZeros(bitNum, data)
}

// LeadingZeros 计算前导零的数量
func LeadingZeros(bitNum int, data *big.Int) *int {
	var count int
	// 从最高位向低位遍历
	for j := bitNum - 1; j >= 0; j-- {
		if data.Bit(j) == 0 {
			count++
		} else {
			break
		}
	}
	return &count
}

// IsZero 检查指定位数范围内的值是否为零
func IsZero(bitNum int, data *big.Int) bool {
	// 创建一个掩码，长度为bitNum
	mask := new(big.Int).Lsh(big.NewInt(1), uint(bitNum))
	mask.Sub(mask, big.NewInt(1))

	// 将data与掩码进行与操作
	result := new(big.Int).And(data, mask)

	// 检查结果是否为零
	return result.Sign() == 0
}

const (
	MinTick = -443636 // This should match your MIN_TICK constant
	MaxTick = 443636  // This should match your MAX_TICK constant
)

var (
	MaxUint128    = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
	MaxUint128Int = cosmath.NewIntFromBigInt(MaxUint128)
)

func mulRightShift(val, mulBy cosmath.Int) cosmath.Int {
	// 先乘法
	result := val.Mul(mulBy)

	// 然后右移 64 位
	// 2^64 = 18446744073709551616
	pow64Big, ok := cosmath.NewIntFromString("18446744073709551616")
	if !ok {
		panic("failed to create pow64Big")
	}

	// 除以 2^64 相当于右移 64 位
	return result.Quo(pow64Big)
}

// getSqrtPriceX64FromTick calculates the sqrt price from a tick value
func getSqrtPriceX64FromTick(tick int64) (cosmath.Int, error) {
	if tick < MinTick || tick > MaxTick {
		return cosmath.Int{}, errors.New("tick must be in MIN_TICK and MAX_TICK")
	}

	tickAbs := tick
	if tick < 0 {
		tickAbs = -tick
	}

	ratio := cosmath.Int{}
	if (tickAbs & 0x1) != 0 {
		ratio, _ = cosmath.NewIntFromString("18445821805675395072")
	} else {
		ratio, _ = cosmath.NewIntFromString("18446744073709551616")
	}

	if (tickAbs & 0x2) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18444899583751176192")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x4) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18443055278223355904")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x8) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18439367220385607680")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x10) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18431993317065453568")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x20) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18417254355718170624")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x40) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18387811781193609216")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x80) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18329067761203558400")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x100) != 0 {
		mulBy, _ := cosmath.NewIntFromString("18212142134806163456")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x200) != 0 {
		mulBy, _ := cosmath.NewIntFromString("17980523815641700352")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x400) != 0 {
		mulBy, _ := cosmath.NewIntFromString("17526086738831433728")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x800) != 0 {
		mulBy, _ := cosmath.NewIntFromString("16651378430235570176")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x1000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("15030750278694412288")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x2000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("12247334978884435968")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x4000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("8131365268886854656")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x8000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("3584323654725218816")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x10000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("696457651848324352")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x20000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("26294789957507116")
		ratio = mulRightShift(ratio, mulBy)
	}
	if (tickAbs & 0x40000) != 0 {
		mulBy, _ := cosmath.NewIntFromString("37481735321082")
		ratio = mulRightShift(ratio, mulBy)
	}

	if tick > 0 {
		ratio = MaxUint128Int.Quo(ratio)
	}

	return ratio, nil
}

// Constants
var (
	MaxSqrtPriceX64, _        = cosmath.NewIntFromString("79226673515401279992447579055")
	MinSqrtPriceX64, _        = cosmath.NewIntFromString("4295048016")
	BitPrecision              = 14
	LogB2X32, _               = cosmath.NewIntFromString("59543866431248")
	LogBPErrMarginLowerX64, _ = cosmath.NewIntFromString("184467440737095516")
	LogBPErrMarginUpperX64, _ = cosmath.NewIntFromString("15793534762490258745")
)

// signedLeftShift performs a left shift operation on a big.Int with sign handling
func signedLeftShift(n *big.Int, shiftBy int, bitWidth int) *big.Int {
	result := new(big.Int).Lsh(n, uint(shiftBy))
	mask := new(big.Int).Lsh(big.NewInt(1), uint(bitWidth))
	mask.Sub(mask, big.NewInt(1))
	return new(big.Int).And(result, mask)
}

// signedRightShift performs a right shift operation on a big.Int with sign handling
func signedRightShift(n *big.Int, shiftBy int, bitWidth int) *big.Int {
	return new(big.Int).Rsh(n, uint(shiftBy))
}

func getTickFromSqrtPriceX64(sqrtPriceX64 cosmath.Int) (int64, error) {
	if sqrtPriceX64.GT(MaxSqrtPriceX64) || sqrtPriceX64.LT(MinSqrtPriceX64) {
		return 0, errors.New("provided sqrtPrice is not within the supported sqrtPrice range")
	}

	// Calculate MSB (most significant bit)
	msb := sqrtPriceX64.BigInt().BitLen() - 1
	adjustedMsb := big.NewInt(int64(msb - 64))
	log2pIntegerX32 := signedLeftShift(adjustedMsb, 32, 128)

	// Initialize variables for the loop
	bit, _ := new(big.Int).SetString("8000000000000000", 16)
	precision := 0
	log2pFractionX64 := big.NewInt(0)

	// Calculate initial r value
	var r *big.Int
	if msb >= 64 {
		r = new(big.Int).Rsh(sqrtPriceX64.BigInt(), uint(msb-63))
	} else {
		r = new(big.Int).Lsh(sqrtPriceX64.BigInt(), uint(63-msb))
	}

	zero := big.NewInt(0)
	for bit.Cmp(zero) > 0 && precision < BitPrecision {
		r = new(big.Int).Mul(r, r)
		rMoreThanTwo := new(big.Int).Rsh(r, 127)
		r = new(big.Int).Rsh(r, uint(63+rMoreThanTwo.Int64()))
		log2pFractionX64 = new(big.Int).Add(log2pFractionX64, new(big.Int).Mul(bit, rMoreThanTwo))
		bit = new(big.Int).Rsh(bit, 1)
		precision++
	}

	log2pFractionX32 := new(big.Int).Rsh(log2pFractionX64, 32)
	log2pX32 := new(big.Int).Add(log2pIntegerX32, log2pFractionX32)
	logbpX64 := new(big.Int).Mul(log2pX32, LogB2X32.BigInt())

	tickLow := new(big.Int).Sub(logbpX64, LogBPErrMarginLowerX64.BigInt())
	tickLow = signedRightShift(tickLow, 64, 128)

	tickHigh := new(big.Int).Add(logbpX64, LogBPErrMarginUpperX64.BigInt())
	tickHigh = signedRightShift(tickHigh, 64, 128)

	if tickLow.Cmp(tickHigh) == 0 {
		return tickLow.Int64(), nil
	}

	// Get sqrt price for high tick and compare
	derivedTickHighSqrtPriceX64, err := getSqrtPriceX64FromTick(tickHigh.Int64())
	if err != nil {
		return 0, err
	}

	if derivedTickHighSqrtPriceX64.LTE(sqrtPriceX64) {
		return tickHigh.Int64(), nil
	}
	return tickLow.Int64(), nil
}

// mergeBitmap 合并 bitmap
func mergeBitmap(bns [16]uint64) uint64 {
	var result uint64
	for i := 0; i < len(bns); i++ {
		result |= bns[i] << uint(i*64)
	}
	return result
}

type SwapStep struct {
	SqrtPriceX64Next *big.Int
	AmountIn         *big.Int
	AmountOut        *big.Int
	FeeAmount        *big.Int
}

// swapStepCompute calculates the next sqrt price, amounts in/out and fee amount for a single swap step
func swapStepCompute(
	sqrtPriceX64Current *big.Int,
	sqrtPriceX64Target *big.Int,
	liquidity *big.Int,
	amountRemaining *big.Int,
	feeRate uint32,
	zeroForOne bool,
) (cosmath.Int, cosmath.Int, cosmath.Int, cosmath.Int) {

	swapStep := &SwapStep{
		SqrtPriceX64Next: new(big.Int),
		AmountIn:         new(big.Int),
		AmountOut:        new(big.Int),
		FeeAmount:        new(big.Int),
	}

	zero := new(big.Int)
	baseInput := amountRemaining.Cmp(zero) >= 0

	if baseInput {
		feeRateBig := cosmath.NewInt(int64(feeRate))
		tmp := FEE_RATE_DENOMINATOR.Sub(feeRateBig)
		amountRemainingSubtractFee := mulDivFloor(cosmath.NewIntFromBigInt(amountRemaining), tmp, FEE_RATE_DENOMINATOR)
		if zeroForOne {
			swapStep.AmountIn = getTokenAmountAFromLiquidity(sqrtPriceX64Target, sqrtPriceX64Current, liquidity, true)
		} else {
			swapStep.AmountIn = getTokenAmountBFromLiquidity(sqrtPriceX64Current, sqrtPriceX64Target, liquidity, true)
		}

		if amountRemainingSubtractFee.GTE(cosmath.NewIntFromBigInt(swapStep.AmountIn)) {
			swapStep.SqrtPriceX64Next.Set(sqrtPriceX64Target)
		} else {
			swapStep.SqrtPriceX64Next = getNextSqrtPriceX64FromInput(
				sqrtPriceX64Current,
				liquidity,
				amountRemainingSubtractFee.BigInt(),
				zeroForOne,
			)
		}
	} else {
		if zeroForOne {
			swapStep.AmountOut = getTokenAmountBFromLiquidity(sqrtPriceX64Target, sqrtPriceX64Current, liquidity, false)
		} else {
			swapStep.AmountOut = getTokenAmountAFromLiquidity(sqrtPriceX64Current, sqrtPriceX64Target, liquidity, false)
		}

		negativeOne := new(big.Int).SetInt64(-1)
		amountRemainingNeg := new(big.Int).Mul(amountRemaining, negativeOne)

		if amountRemainingNeg.Cmp(swapStep.AmountOut) >= 0 {
			swapStep.SqrtPriceX64Next.Set(sqrtPriceX64Target)
		} else {
			swapStep.SqrtPriceX64Next = getNextSqrtPriceX64FromOutput(
				sqrtPriceX64Current,
				liquidity,
				amountRemainingNeg,
				zeroForOne,
			)
		}
	}

	reachTargetPrice := swapStep.SqrtPriceX64Next.Cmp(sqrtPriceX64Target) == 0

	if zeroForOne {
		if !(reachTargetPrice && baseInput) {
			swapStep.AmountIn = getTokenAmountAFromLiquidity(
				swapStep.SqrtPriceX64Next,
				sqrtPriceX64Current,
				liquidity,
				true,
			)
		}

		if !(reachTargetPrice && !baseInput) {
			swapStep.AmountOut = getTokenAmountBFromLiquidity(
				swapStep.SqrtPriceX64Next,
				sqrtPriceX64Current,
				liquidity,
				false,
			)
		}
	} else {
		if reachTargetPrice && baseInput {
			// Keep existing amountIn
		} else {
			swapStep.AmountIn = getTokenAmountBFromLiquidity(
				sqrtPriceX64Current,
				swapStep.SqrtPriceX64Next,
				liquidity,
				true,
			)
		}

		if reachTargetPrice && !baseInput {
			// Keep existing amountOut
		} else {
			swapStep.AmountOut = getTokenAmountAFromLiquidity(
				sqrtPriceX64Current,
				swapStep.SqrtPriceX64Next,
				liquidity,
				false,
			)
		}
	}

	if !baseInput {
		negativeOne := new(big.Int).SetInt64(-1)
		amountRemainingNeg := new(big.Int).Mul(amountRemaining, negativeOne)
		if swapStep.AmountOut.Cmp(amountRemainingNeg) > 0 {
			swapStep.AmountOut.Set(amountRemainingNeg)
		}
	}

	if baseInput && swapStep.SqrtPriceX64Next.Cmp(sqrtPriceX64Target) != 0 {
		swapStep.FeeAmount = new(big.Int).Sub(amountRemaining, swapStep.AmountIn)
	} else {
		feeRateBig := cosmath.NewInt(int64(feeRate))
		feeRateSubtracted := FEE_RATE_DENOMINATOR.Sub(feeRateBig)
		swapStep.FeeAmount = mulDivCeil(cosmath.NewIntFromBigInt(swapStep.AmountIn), feeRateBig, feeRateSubtracted).BigInt()
	}

	return cosmath.NewIntFromBigInt(swapStep.SqrtPriceX64Next), cosmath.NewIntFromBigInt(swapStep.AmountIn),
		cosmath.NewIntFromBigInt(swapStep.AmountOut), cosmath.NewIntFromBigInt(swapStep.FeeAmount)
}

// Helper function for ceiling division
func mulDivCeil(a, b, denominator cosmath.Int) cosmath.Int {
	// 检查除数是否为0
	if denominator.IsZero() {
		return cosmath.Int{}
	}

	// 计算 a * b
	numerator := a.Mul(b).Add(denominator.Sub(cosmath.OneInt()))
	// 计算最终结果 numerator / denominator
	return numerator.Quo(denominator)
}

// getTokenAmountAFromLiquidity calculates token amount A from liquidity
func getTokenAmountAFromLiquidity(
	sqrtPriceX64A *big.Int,
	sqrtPriceX64B *big.Int,
	liquidity *big.Int,
	roundUp bool,
) *big.Int {
	// Create copies to avoid modifying the original values
	priceA := new(big.Int).Set(sqrtPriceX64A)
	priceB := new(big.Int).Set(sqrtPriceX64B)

	// Swap if priceA > priceB
	if priceA.Cmp(priceB) > 0 {
		priceA, priceB = priceB, priceA
	}

	// Check if priceA > 0
	if priceA.Cmp(big.NewInt(0)) <= 0 {
		panic("sqrtPriceX64A must be greater than 0")
	}

	// Calculate numerator1 = liquidity << U64Resolution
	numerator1 := new(big.Int).Lsh(liquidity, U64Resolution)

	// Calculate numerator2 = priceB - priceA
	numerator2 := new(big.Int).Sub(priceB, priceA)

	if roundUp {
		// First calculate mulDivCeil(numerator1, numerator2, priceB)
		temp := mulDivCeil(cosmath.NewIntFromBigInt(numerator1), cosmath.NewIntFromBigInt(numerator2), cosmath.NewIntFromBigInt(priceB))
		// Then calculate mulDivCeil(temp, 1, priceA)
		return mulDivCeil(temp, cosmath.NewIntFromBigInt(big.NewInt(1)), cosmath.NewIntFromBigInt(priceA)).BigInt()
	} else {
		// Calculate mulDivFloor(numerator1, numerator2, priceB)
		temp := mulDivFloor(cosmath.NewIntFromBigInt(numerator1), cosmath.NewIntFromBigInt(numerator2), cosmath.NewIntFromBigInt(priceB))
		// Then divide by priceA
		return temp.Quo(cosmath.NewIntFromBigInt(priceA)).BigInt()
	}
}

// getTokenAmountBFromLiquidity calculates token amount B from liquidity
func getTokenAmountBFromLiquidity(
	sqrtPriceX64A *big.Int,
	sqrtPriceX64B *big.Int,
	liquidity *big.Int,
	roundUp bool,
) *big.Int {
	// Create copies to avoid modifying the original values
	priceA := new(big.Int).Set(sqrtPriceX64A)
	priceB := new(big.Int).Set(sqrtPriceX64B)

	// Swap if priceA > priceB
	if priceA.Cmp(priceB) > 0 {
		priceA, priceB = priceB, priceA
	}

	// Check if priceA > 0
	if priceA.Cmp(big.NewInt(0)) <= 0 {
		panic("sqrtPriceX64A must be greater than 0")
	}

	// Calculate price difference
	priceDiff := new(big.Int).Sub(priceB, priceA)

	if roundUp {
		return mulDivCeil(cosmath.NewIntFromBigInt(liquidity), cosmath.NewIntFromBigInt(priceDiff), cosmath.NewIntFromBigInt(new(big.Int).Lsh(big.NewInt(1), U64Resolution))).BigInt()
	} else {
		return mulDivFloor(cosmath.NewIntFromBigInt(liquidity), cosmath.NewIntFromBigInt(priceDiff), cosmath.NewIntFromBigInt(new(big.Int).Lsh(big.NewInt(1), U64Resolution))).BigInt()
	}
}

// mulDivFloor performs multiplication and division with floor rounding
func mulDivFloor(a, b, denominator cosmath.Int) cosmath.Int {
	if denominator.IsZero() {
		panic("division by zero")
	}

	numerator := a.Mul(b)
	return numerator.Quo(denominator)
}

func getNextSqrtPriceX64FromInput(
	sqrtPriceX64Current *big.Int,
	liquidity *big.Int,
	amount *big.Int,
	zeroForOne bool,
) *big.Int {

	if sqrtPriceX64Current.Cmp(big.NewInt(0)) <= 0 {
		panic("sqrtPriceX64Current must be greater than 0")
	}
	if liquidity.Cmp(big.NewInt(0)) <= 0 {
		panic("liquidity must be greater than 0")
	}

	if amount.Cmp(big.NewInt(0)) == 0 {
		return sqrtPriceX64Current
	}

	if zeroForOne {
		return getNextSqrtPriceFromTokenAmountARoundingUp(sqrtPriceX64Current, liquidity, amount, true)
	} else {
		return getNextSqrtPriceFromTokenAmountBRoundingDown(sqrtPriceX64Current, liquidity, amount, true)
	}
}

// getNextSqrtPriceX64FromOutput calculates the next sqrt price from output amount
func getNextSqrtPriceX64FromOutput(
	sqrtPriceX64Current *big.Int,
	liquidity *big.Int,
	amount *big.Int,
	zeroForOne bool,
) *big.Int {
	if sqrtPriceX64Current.Cmp(big.NewInt(0)) <= 0 {
		panic("sqrtPriceX64Current must be greater than 0")
	}
	if liquidity.Cmp(big.NewInt(0)) <= 0 {
		panic("liquidity must be greater than 0")
	}

	if zeroForOne {
		return getNextSqrtPriceFromTokenAmountBRoundingDown(sqrtPriceX64Current, liquidity, amount, false)
	} else {
		return getNextSqrtPriceFromTokenAmountARoundingUp(sqrtPriceX64Current, liquidity, amount, false)
	}
}

func getNextSqrtPriceFromTokenAmountARoundingUp(
	sqrtPriceX64 *big.Int,
	liquidity *big.Int,
	amount *big.Int,
	add bool,
) *big.Int {

	if amount.Cmp(big.NewInt(0)) == 0 {
		return sqrtPriceX64
	}

	liquidityLeftShift := new(big.Int).Lsh(liquidity, U64Resolution)

	if add {
		numerator1 := liquidityLeftShift
		denominator := new(big.Int).Add(liquidityLeftShift, new(big.Int).Mul(amount, sqrtPriceX64))
		if denominator.Cmp(numerator1) >= 0 {
			return mulDivCeil(cosmath.NewIntFromBigInt(numerator1), cosmath.NewIntFromBigInt(sqrtPriceX64), cosmath.NewIntFromBigInt(denominator)).BigInt()
		}

		temp := new(big.Int).Div(numerator1, sqrtPriceX64)
		temp.Add(temp, amount)
		return mulDivRoundingUp(numerator1, big.NewInt(1), temp)
	} else {
		amountMulSqrtPrice := new(big.Int).Mul(amount, sqrtPriceX64)
		if liquidityLeftShift.Cmp(amountMulSqrtPrice) <= 0 {
			panic("getNextSqrtPriceFromTokenAmountARoundingUp: liquidityLeftShift must be greater than amountMulSqrtPrice")
		}
		denominator := new(big.Int).Sub(liquidityLeftShift, amountMulSqrtPrice)
		return mulDivCeil(cosmath.NewIntFromBigInt(liquidityLeftShift), cosmath.NewIntFromBigInt(sqrtPriceX64), cosmath.NewIntFromBigInt(denominator)).BigInt()
	}
}

// getNextSqrtPriceFromTokenAmountBRoundingDown calculates next sqrt price from token B amount
func getNextSqrtPriceFromTokenAmountBRoundingDown(
	sqrtPriceX64 *big.Int,
	liquidity *big.Int,
	amount *big.Int,
	add bool,
) *big.Int {
	deltaY := new(big.Int).Lsh(amount, U64Resolution)

	if add {
		return new(big.Int).Add(sqrtPriceX64, new(big.Int).Div(deltaY, liquidity))
	} else {
		amountDivLiquidity := mulDivRoundingUp(deltaY, big.NewInt(1), liquidity)
		if sqrtPriceX64.Cmp(amountDivLiquidity) <= 0 {
			panic("getNextSqrtPriceFromTokenAmountBRoundingDown: sqrtPriceX64 must be greater than amountDivLiquidity")
		}
		return new(big.Int).Sub(sqrtPriceX64, amountDivLiquidity)
	}
}

// mulDivRoundingUp performs multiplication and division with ceiling rounding
func mulDivRoundingUp(a, b, denominator *big.Int) *big.Int {
	numerator := new(big.Int).Mul(a, b)
	result := new(big.Int).Div(numerator, denominator)
	if !new(big.Int).Mod(numerator, denominator).IsInt64() {
		result.Add(result, big.NewInt(1))
	}
	return result
}
