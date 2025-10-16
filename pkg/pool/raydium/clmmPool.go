package raydium

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"

	cosmath "cosmossdk.io/math"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/solana-zh/solroute/pkg"
	"github.com/solana-zh/solroute/pkg/sol"
	"lukechampine.com/uint128"
)

type CLMMPool struct {
	// 8 bytes discriminator
	Discriminator [8]uint8 `bin:"skip"`
	// Core states
	Bump           uint8
	AmmConfig      solana.PublicKey
	Owner          solana.PublicKey
	TokenMint0     solana.PublicKey
	TokenMint1     solana.PublicKey
	TokenVault0    solana.PublicKey
	TokenVault1    solana.PublicKey
	ObservationKey solana.PublicKey
	MintDecimals0  uint8
	MintDecimals1  uint8
	TickSpacing    uint16
	// Liquidity states
	Liquidity                 uint128.Uint128
	SqrtPriceX64              uint128.Uint128
	TickCurrent               int32
	ObservationIndex          uint16
	ObservationUpdateDuration uint16
	FeeGrowthGlobal0X64       uint128.Uint128
	FeeGrowthGlobal1X64       uint128.Uint128
	ProtocolFeesToken0        uint64
	ProtocolFeesToken1        uint64
	SwapInAmountToken0        uint128.Uint128
	SwapOutAmountToken1       uint128.Uint128
	SwapInAmountToken1        uint128.Uint128
	SwapOutAmountToken0       uint128.Uint128
	Status                    uint8
	Padding                   [7]uint8
	// Reward states
	RewardInfos [3]RewardInfo
	// Tick array states
	TickArrayBitmap [16]uint64
	// Fee states
	TotalFeesToken0        uint64
	TotalFeesClaimedToken0 uint64
	TotalFeesToken1        uint64
	TotalFeesClaimedToken1 uint64
	FundFeesToken0         uint64
	FundFeesToken1         uint64
	// Other states
	OpenTime    uint64
	RecentEpoch uint64
	Padding1    [24]uint64
	Padding2    [32]uint64

	PoolId            solana.PublicKey
	FeeRate           uint32
	ExBitmapAddress   solana.PublicKey
	exTickArrayBitmap *TickArrayBitmapExtensionType
	TickArrayCache    map[string]TickArray
}

type RewardInfo struct {
	RewardState           uint8
	OpenTime              uint64
	EndTime               uint64
	LastUpdateTime        uint64
	EmissionsPerSecondX64 uint128.Uint128
	RewardTotalEmissioned uint64
	RewardClaimed         uint64
	TokenMint             solana.PublicKey
	TokenVault            solana.PublicKey
	Authority             solana.PublicKey
	RewardGrowthGlobalX64 uint128.Uint128
}

func (pool *CLMMPool) ProtocolName() pkg.ProtocolName {
	return pkg.ProtocolNameRaydiumClmm
}

func (pool *CLMMPool) GetProgramID() solana.PublicKey {
	return RAYDIUM_CLMM_PROGRAM_ID
}

func (l *CLMMPool) Decode(data []byte) error {
	// Skip 8 bytes discriminator if present
	if len(data) > 8 {
		data = data[8:]
	}

	offset := 0

	// Parse core states
	l.Bump = data[offset]
	offset += 1

	l.AmmConfig = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.Owner = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.TokenMint0 = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.TokenMint1 = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.TokenVault0 = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.TokenVault1 = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.ObservationKey = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32

	l.MintDecimals0 = data[offset]
	offset += 1

	l.MintDecimals1 = data[offset]
	offset += 1

	l.TickSpacing = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	// Parse liquidity states
	l.Liquidity = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.SqrtPriceX64 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.TickCurrent = int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
	offset += 4

	l.ObservationIndex = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	l.ObservationUpdateDuration = binary.LittleEndian.Uint16(data[offset : offset+2])
	offset += 2

	l.FeeGrowthGlobal0X64 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.FeeGrowthGlobal1X64 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.ProtocolFeesToken0 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.ProtocolFeesToken1 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.SwapInAmountToken0 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.SwapOutAmountToken1 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.SwapInAmountToken1 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.SwapOutAmountToken0 = uint128.FromBytes(data[offset : offset+16])
	offset += 16

	l.Status = data[offset]
	offset += 1

	// Skip padding
	offset += 7

	// Parse reward states
	for i := 0; i < 3; i++ {
		l.RewardInfos[i].RewardState = data[offset]
		offset += 1

		l.RewardInfos[i].OpenTime = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		l.RewardInfos[i].EndTime = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		l.RewardInfos[i].LastUpdateTime = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		l.RewardInfos[i].EmissionsPerSecondX64 = uint128.FromBytes(data[offset : offset+16])
		offset += 16

		l.RewardInfos[i].RewardTotalEmissioned = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		l.RewardInfos[i].RewardClaimed = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8

		l.RewardInfos[i].TokenMint = solana.PublicKeyFromBytes(data[offset : offset+32])
		offset += 32

		l.RewardInfos[i].TokenVault = solana.PublicKeyFromBytes(data[offset : offset+32])
		offset += 32

		l.RewardInfos[i].Authority = solana.PublicKeyFromBytes(data[offset : offset+32])
		offset += 32

		l.RewardInfos[i].RewardGrowthGlobalX64 = uint128.FromBytes(data[offset : offset+16])
		offset += 16
	}

	// Parse tick array bitmap
	for i := 0; i < 16; i++ {
		l.TickArrayBitmap[i] = binary.LittleEndian.Uint64(data[offset : offset+8])
		offset += 8
	}

	// Parse fee states
	l.TotalFeesToken0 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.TotalFeesClaimedToken0 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.TotalFeesToken1 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.TotalFeesClaimedToken1 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.FundFeesToken0 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.FundFeesToken1 = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Parse other states
	l.OpenTime = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	l.RecentEpoch = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8

	// Skip padding1
	offset += 24 * 8

	// Skip padding2
	offset += 32 * 8
	return nil
}

func (l *CLMMPool) Span() uint64 {
	return uint64(1544)
}

func (l *CLMMPool) Offset(field string) uint64 {
	// Add 8 bytes for discriminator
	baseOffset := uint64(8)

	switch field {
	case "TokenMint0":
		return baseOffset + 1 + 32 + 32 // bump + ammConfig + owner
	case "TokenMint1":
		return baseOffset + 1 + 32 + 32 + 32 // bump + ammConfig + owner + tokenMint0
	}
	return 0
}

func (l *CLMMPool) CurrentPrice() float64 {
	sqrtPrice, _ := l.SqrtPriceX64.Big().Float64()
	sqrtPrice = sqrtPrice / math.Pow(2, 64)
	price := sqrtPrice * sqrtPrice
	return price
}

func (p *CLMMPool) BuildSwapInstructions(
	ctx context.Context,
	solClient *sol.Client,
	userAddr solana.PublicKey,
	inputMint string,
	amountIn cosmath.Int,
	minOutAmountWithDecimals cosmath.Int,
	userBaseAccount solana.PublicKey,
	userQuoteAccount solana.PublicKey,
) ([]solana.Instruction, error) {

	instrs := []solana.Instruction{}

	var inputValueMint solana.PublicKey
	var outputValueMint solana.PublicKey
	if inputMint == p.TokenMint0.String() {
		log.Printf("inputMint: %v, p.TokenMint0: %v", inputMint, p.TokenMint0.String())
		inputValueMint = p.TokenMint0
		outputValueMint = p.TokenMint1
	} else {
		log.Printf("inputMint: %v, p.TokenMint1: %v", inputMint, p.TokenMint1.String())
		inputValueMint = p.TokenMint1
		outputValueMint = p.TokenMint0
	}

	inst := RayCLMMSwapInstruction{
		Amount:               amountIn.Uint64(),
		OtherAmountThreshold: minOutAmountWithDecimals.Uint64(),
		SqrtPriceLimitX64:    uint128.Zero,
		IsBaseInput:          inputValueMint == p.TokenMint0,
		AccountMetaSlice:     make(solana.AccountMetaSlice, 16),
	}
	inst.BaseVariant = bin.BaseVariant{
		Impl: inst,
	}

	// Set up account metas in the correct order according to SDK
	inst.AccountMetaSlice[0] = solana.NewAccountMeta(userAddr, false, true)
	inst.AccountMetaSlice[1] = solana.NewAccountMeta(p.AmmConfig, false, false)
	inst.AccountMetaSlice[2] = solana.NewAccountMeta(p.PoolId, true, false)

	if inputMint == p.TokenMint0.String() {
		inst.AccountMetaSlice[3] = solana.NewAccountMeta(userBaseAccount, true, false)
		inst.AccountMetaSlice[4] = solana.NewAccountMeta(userQuoteAccount, true, false)
		inst.AccountMetaSlice[5] = solana.NewAccountMeta(p.TokenVault0, true, false)
		inst.AccountMetaSlice[6] = solana.NewAccountMeta(p.TokenVault1, true, false)
	} else {
		inst.AccountMetaSlice[3] = solana.NewAccountMeta(userQuoteAccount, true, false)
		inst.AccountMetaSlice[4] = solana.NewAccountMeta(userBaseAccount, true, false)
		inst.AccountMetaSlice[5] = solana.NewAccountMeta(p.TokenVault1, true, false)
		inst.AccountMetaSlice[6] = solana.NewAccountMeta(p.TokenVault0, true, false)
	}
	inst.AccountMetaSlice[7] = solana.NewAccountMeta(p.ObservationKey, true, false)
	inst.AccountMetaSlice[8] = solana.NewAccountMeta(solana.TokenProgramID, false, false)
	inst.AccountMetaSlice[9] = solana.NewAccountMeta(TOKEN_2022_PROGRAM_ID, false, false)
	inst.AccountMetaSlice[10] = solana.NewAccountMeta(MEMO_PROGRAM_ID, false, false)
	inst.AccountMetaSlice[11] = solana.NewAccountMeta(inputValueMint, false, false)
	inst.AccountMetaSlice[12] = solana.NewAccountMeta(outputValueMint, false, false)

	// Add bitmap extension as remaining account if it exists
	exBitmapAddress, _, err := GetPdaExBitmapAccount(RAYDIUM_CLMM_PROGRAM_ID, p.PoolId)
	if err != nil {
		log.Printf("get pda address error: %v", err)
		return nil, fmt.Errorf("get pda address error: %v", err)
	}
	inst.AccountMetaSlice[13] = solana.NewAccountMeta(exBitmapAddress, true, false) // exTickArrayBitmap (is_writable = true, is_signer = false)

	// Add tick arrays as remaining accounts
	remainingAccounts, err := p.GetRemainAccounts(ctx, solClient, inputValueMint.String())
	if err != nil {
		log.Printf("GetRemainAccounts error: %v", err)
		return nil, err
	}
	inst.AccountMetaSlice[14] = solana.NewAccountMeta(remainingAccounts[0], true, false)
	inst.AccountMetaSlice[15] = solana.NewAccountMeta(remainingAccounts[1], true, false)

	instrs = append(instrs, &inst)

	return instrs, nil
}

// RayCLMMSwapInstruction represents a swap instruction for the Raydium CLMM pool
type RayCLMMSwapInstruction struct {
	bin.BaseVariant
	Amount                  uint64
	OtherAmountThreshold    uint64
	SqrtPriceLimitX64       uint128.Uint128
	IsBaseInput             bool
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

// ProgramID returns the program ID for the Raydium CLMM program
func (inst *RayCLMMSwapInstruction) ProgramID() solana.PublicKey {
	return RAYDIUM_CLMM_PROGRAM_ID
}

// Accounts returns the account metas for the instruction
func (inst *RayCLMMSwapInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.AccountMetaSlice
}

// Data serializes the instruction data
func (inst *RayCLMMSwapInstruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)

	// Write discriminator for swap instruction
	discriminator := []byte{43, 4, 237, 11, 26, 201, 30, 98} // anchorDataBuf.swap
	if _, err := buf.Write(discriminator); err != nil {
		return nil, fmt.Errorf("failed to write discriminator: %w", err)
	}

	// Write amount
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.Amount, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode amount: %w", err)
	}

	// Write other amount threshold
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.OtherAmountThreshold, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode other amount threshold: %w", err)
	}

	// Write sqrt price limit x64
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.SqrtPriceLimitX64.Hi, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode sqrt price limit hi: %w", err)
	}
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.SqrtPriceLimitX64.Lo, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode sqrt price limit lo: %w", err)
	}

	// Write is base input
	if err := bin.NewBorshEncoder(buf).WriteBool(inst.IsBaseInput); err != nil {
		return nil, fmt.Errorf("failed to encode is base input: %w", err)
	}

	return buf.Bytes(), nil
}

// GetID returns the pool ID
func (pool *CLMMPool) GetID() string {
	return pool.PoolId.String()
}

// GetTokens returns the base and quote token mints
func (pool *CLMMPool) GetTokens() (baseMint, quoteMint string) {
	return pool.TokenMint0.String(), pool.TokenMint1.String()
}

func (pool *CLMMPool) Quote(ctx context.Context, solClient *sol.Client, inputMint string, inputAmount cosmath.Int) (cosmath.Int, error) {
	// update pool state first
	results, err := solClient.GetMultipleAccountsWithOpts(ctx, []solana.PublicKey{pool.ExBitmapAddress})
	if err != nil {
		return cosmath.Int{}, fmt.Errorf("batch request failed: %v", err)
	}
	for _, result := range results.Value {
		pool.ParseExBitmapInfo(result.Data.GetBinary())
	}

	tickArrayAddresses, err := pool.GetTickArrayAddresses()
	if err != nil {
		return cosmath.Int{}, fmt.Errorf("get tick array address error: %v", err)
	}
	results, err = solClient.GetMultipleAccountsWithOpts(ctx, tickArrayAddresses)
	if err != nil {
		log.Printf("batch request failed: %v", err)
		return cosmath.Int{}, fmt.Errorf("batch request failed: %v", err)
	}
	for _, result := range results.Value {
		tickArray := &TickArray{}
		err := tickArray.Decode(result.Data.GetBinary())
		if err != nil {
			return cosmath.Int{}, fmt.Errorf("failed to decode tick array: %w", err)
		}
		if pool.TickArrayCache == nil {
			pool.TickArrayCache = make(map[string]TickArray)
		}
		pool.TickArrayCache[strconv.FormatInt(int64(tickArray.StartTickIndex), 10)] = *tickArray
	}

	if inputMint == pool.TokenMint0.String() {
		priceBaseToQuote, err := pool.ComputeAmountOutFormat(pool.TokenMint0.String(), inputAmount)
		if err != nil {
			return cosmath.Int{}, err
		}
		return priceBaseToQuote.Neg(), nil
	} else {
		priceQuoteToBase, err := pool.ComputeAmountOutFormat(pool.TokenMint1.String(), inputAmount)
		if err != nil {
			return cosmath.Int{}, err
		}
		return priceQuoteToBase.Neg(), nil
	}
}

// ComputeAmountOutFormat calculates the expected output amount for a given input amount
func (pool *CLMMPool) ComputeAmountOutFormat(inputTokenMint string, inputAmount cosmath.Int) (cosmath.Int, error) {
	zeroForOne := inputTokenMint == pool.TokenMint0.String()

	firstTickArrayStartIndex, _, err := pool.getFirstInitializedTickArray(zeroForOne, pool.exTickArrayBitmap)
	if err != nil {
		return cosmath.Int{}, fmt.Errorf("failed to get first initialized tick array: %w", err)
	}

	expectedAmountOut, err := pool.swapCompute(
		int64(pool.TickCurrent),
		zeroForOne,
		inputAmount,
		cosmath.NewIntFromUint64(uint64(pool.FeeRate)),
		firstTickArrayStartIndex,
		pool.exTickArrayBitmap,
	)
	if err != nil {
		return cosmath.Int{}, fmt.Errorf("failed to compute swap amount: %w", err)
	}

	return expectedAmountOut, nil
}

// swapCompute performs the core swap calculation logic
func (pool *CLMMPool) swapCompute(
	currentTick int64,
	zeroForOne bool,
	amountSpecified cosmath.Int,
	fee cosmath.Int,
	lastSavedTickArrayStartIndex int64,
	exTickArrayBitmap *TickArrayBitmapExtensionType,
) (cosmath.Int, error) {
	if amountSpecified.IsZero() {
		return cosmath.Int{}, errors.New("input amount cannot be zero")
	}

	baseInput := amountSpecified.IsPositive()
	sqrtPriceLimitX64 := cosmath.NewInt(0)

	// Initialize calculation variables
	amountSpecifiedRemaining := amountSpecified
	amountCalculated := cosmath.NewInt(0)
	amountIn := cosmath.NewInt(0)
	amountOut := cosmath.NewInt(0)
	feeAmount := cosmath.NewInt(0)
	sqrtPriceX64 := cosmath.NewIntFromBigInt(pool.SqrtPriceX64.Big())
	tick := int64(0)

	// Calculate initial tick
	if currentTick > lastSavedTickArrayStartIndex {
		if lastSavedTickArrayStartIndex+getTickCount(int64(pool.TickSpacing))-1 < currentTick {
			tick = lastSavedTickArrayStartIndex + getTickCount(int64(pool.TickSpacing)) - 1
		} else {
			tick = currentTick
		}
	} else {
		tick = lastSavedTickArrayStartIndex
	}

	// Initialize accounts and liquidity
	accounts := make([]*solana.PublicKey, 0)
	liquidity := cosmath.NewIntFromBigInt(pool.Liquidity.Big())
	tickAarrayStartIndex := lastSavedTickArrayStartIndex
	tickArrayCurrent := pool.TickArrayCache[strconv.FormatInt(lastSavedTickArrayStartIndex, 10)]

	// Set price limits based on direction
	if baseInput {
		sqrtPriceLimitX64 = MIN_SQRT_PRICE_X64.Add(cosmath.NewInt(1))
	} else {
		sqrtPriceLimitX64 = MAX_SQRT_PRICE_X64.Sub(cosmath.NewInt(1))
	}
	t := !zeroForOne && int64(tickArrayCurrent.StartTickIndex) == tick

	// Main swap calculation loop
	loop := 0
	for {
		if amountSpecifiedRemaining.IsZero() || sqrtPriceX64.Equal(sqrtPriceLimitX64) {
			break
		}

		sqrtPriceStartX64 := sqrtPriceX64
		tickState := getNextInitTick(&tickArrayCurrent, tick, int64(pool.TickSpacing), zeroForOne, t)

		nextInitTick := tickState
		tickArrayAddress := &solana.PublicKey{}

		// Handle liquidity crossing
		if nextInitTick == nil || nextInitTick.LiquidityGross.Big().Cmp(big.NewInt(0)) <= 0 {
			isExist, nextInitTickArrayIndex, err := nextInitializedTickArrayStartIndexUtils(
				exTickArrayBitmap,
				tick,
				int64(pool.TickSpacing),
				pool.TickArrayBitmap,
				zeroForOne,
			)
			if err != nil {
				return cosmath.Int{}, fmt.Errorf("failed to get next initialized tick array: %w", err)
			}
			if !isExist {
				return cosmath.Int{}, errors.New("insufficient liquidity")
			}

			tickAarrayStartIndex := nextInitTickArrayIndex
			expectedNextTickArrayAddress := getPdaTickArrayAddress(RAYDIUM_CLMM_PROGRAM_ID, pool.PoolId, tickAarrayStartIndex)

			tickArrayAddress = &expectedNextTickArrayAddress
			tickArrayCurrent = pool.TickArrayCache[strconv.FormatInt(tickAarrayStartIndex, 10)]
			nextInitTick, err = firstInitializedTick(&tickArrayCurrent, zeroForOne)
			if err != nil {
				return cosmath.Int{}, fmt.Errorf("failed to get first initialized tick: %w", err)
			}
		}

		// Calculate next tick and price
		tickNext := int64(nextInitTick.Tick)
		initialized := nextInitTick.LiquidityGross.Big().Cmp(big.NewInt(0)) > 0
		if lastSavedTickArrayStartIndex != tickAarrayStartIndex && tickArrayAddress != nil {
			accounts = append(accounts, tickArrayAddress)
			lastSavedTickArrayStartIndex = tickAarrayStartIndex
		}

		// Clamp tick to valid range
		if tickNext < MIN_TICK {
			tickNext = MIN_TICK
		} else if tickNext > MAX_TICK {
			tickNext = MAX_TICK
		}

		sqrtPriceNextX64, err := getSqrtPriceX64FromTick(int64(tickNext))
		if err != nil {
			return cosmath.Int{}, fmt.Errorf("failed to get sqrt price from tick: %w", err)
		}

		// Calculate target price
		targetPrice := cosmath.NewInt(0)
		if (zeroForOne && sqrtPriceNextX64.LT(sqrtPriceLimitX64)) ||
			(!zeroForOne && sqrtPriceNextX64.GT(sqrtPriceLimitX64)) {
			targetPrice = sqrtPriceLimitX64
		} else {
			targetPrice = sqrtPriceNextX64
		}

		// Calculate swap step
		sqrtPriceX64, amountIn, amountOut, feeAmount = swapStepCompute(
			sqrtPriceX64.BigInt(),
			targetPrice.BigInt(),
			liquidity.BigInt(),
			amountSpecifiedRemaining.BigInt(),
			uint32(fee.Int64()),
			zeroForOne,
		)

		// Update amounts
		if baseInput {
			amountSpecifiedRemaining = amountSpecifiedRemaining.Sub(amountIn.Add(feeAmount))
			amountCalculated = amountCalculated.Sub(amountOut)
		} else {
			amountSpecifiedRemaining = amountSpecifiedRemaining.Add(amountOut)
			amountCalculated = amountCalculated.Add(amountIn.Add(feeAmount))
		}

		// Update liquidity and tick
		if sqrtPriceX64.Equal(sqrtPriceNextX64) {
			if initialized {
				liquidityNet := nextInitTick.LiquidityNet
				if zeroForOne {
					liquidityNet = -liquidityNet
				}
				liquidity = liquidity.Add(cosmath.NewInt(liquidityNet))
			}
			t = tickNext != tick && !zeroForOne && int64(tickArrayCurrent.StartTickIndex) == tickNext
			if zeroForOne {
				tick = tickNext - 1
			} else {
				tick = tickNext
			}
		} else if sqrtPriceX64 != sqrtPriceStartX64 {
			_T, err := getTickFromSqrtPriceX64(sqrtPriceX64)
			if err != nil {
				return cosmath.Int{}, fmt.Errorf("failed to get tick from sqrt price: %w", err)
			}
			t = _T != tick && !zeroForOne && int64(tickArrayCurrent.StartTickIndex) == _T
			tick = _T
		}

		// Safety check for infinite loops
		loop++
		if loop > 100 {
			return cosmath.Int{}, errors.New("swap computation exceeded maximum iterations")
		}
	}

	return amountCalculated, nil
}

// GetRemainAccounts returns the remaining accounts needed for the swap
func (pool *CLMMPool) GetRemainAccounts(
	ctx context.Context,
	client *sol.Client,
	inputTokenMint string,
) ([]solana.PublicKey, error) {
	// Determine swap direction
	zeroForOne := inputTokenMint == pool.TokenMint0.String()

	// Get first initialized tick array
	_, firstTickArray, err := pool.getFirstInitializedTickArray(zeroForOne, pool.exTickArrayBitmap)
	if err != nil {
		return nil, fmt.Errorf("failed to get first tick array: %w", err)
	}

	allNeededAccounts := make([]solana.PublicKey, 0)
	allNeededAccounts = append(allNeededAccounts, firstTickArray)

	// Get next tick array
	tickAarrayStartIndex, _ := nextInitializedTickArray(
		int64(pool.TickCurrent),
		int64(pool.TickSpacing),
		zeroForOne,
		pool.TickArrayBitmap,
		pool.exTickArrayBitmap,
	)

	exTickArrayBitmapAddress := getPdaTickArrayAddress(RAYDIUM_CLMM_PROGRAM_ID, pool.PoolId, tickAarrayStartIndex)
	allNeededAccounts = append(allNeededAccounts, exTickArrayBitmapAddress)
	if exTickArrayBitmapAddress.String() == firstTickArray.String() {
		return nil, errors.New("exTickArrayBitmapAddress is the same as firstTickArray")
	}
	return allNeededAccounts, nil
}
