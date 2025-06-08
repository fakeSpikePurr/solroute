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
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/yimingwow/solroute/pkg"
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
	UserBaseAccount   solana.PublicKey
	UserQuoteAccount  solana.PublicKey
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
	// 转换为 float64
	sqrtPrice, _ := l.SqrtPriceX64.Big().Float64()
	// Q64.64 格式转换
	sqrtPrice = sqrtPrice / math.Pow(2, 64)
	// 计算实际价格
	price := sqrtPrice * sqrtPrice
	return price
}

func (s *CLMMPool) BuildSwapInstructions(
	ctx context.Context,
	solClient *rpc.Client,
	userAddr solana.PublicKey,
	pool *CLMMPool,
	amountIn cosmath.Int,
	minOutAmountWithDecimals cosmath.Int,
	inputMint string,
) ([]solana.Instruction, error) {

	// 初始化指令数组和签名者
	instrs := []solana.Instruction{}

	var inputValueMint solana.PublicKey
	var outputValueMint solana.PublicKey
	var inputValue solana.PublicKey
	var outputValue solana.PublicKey
	if inputMint == pool.TokenMint0.String() {
		inputValueMint = pool.TokenMint0
		outputValueMint = pool.TokenMint1
		inputValue = pool.TokenVault0
		outputValue = pool.TokenVault1
	} else {
		inputValueMint = pool.TokenMint1
		outputValueMint = pool.TokenMint0
		inputValue = pool.TokenVault1
		outputValue = pool.TokenVault0
	}

	// Create toAccount if needed
	var fromAccount solana.PublicKey
	var toAccount solana.PublicKey
	if inputValueMint.String() == pool.TokenMint0.String() {
		fromAccount = pool.UserBaseAccount
		toAccount = pool.UserQuoteAccount
	} else {
		fromAccount = pool.UserQuoteAccount
		toAccount = pool.UserBaseAccount
	}

	inst := RayCLMMSwapInstruction{
		Amount:               amountIn.Uint64(),
		OtherAmountThreshold: minOutAmountWithDecimals.Uint64(),
		SqrtPriceLimitX64:    uint128.Zero,
		IsBaseInput:          inputValueMint == pool.TokenMint0,
		AccountMetaSlice:     make(solana.AccountMetaSlice, 0),
	}
	inst.BaseVariant = bin.BaseVariant{
		Impl: inst,
	}

	// Set up account metas in the correct order according to SDK
	inst.AccountMetaSlice = append(inst.AccountMetaSlice,
		solana.NewAccountMeta(userAddr, false, true),               // payer (is_signer = true, is_writable = false)
		solana.NewAccountMeta(pool.AmmConfig, false, false),        // ammConfigId
		solana.NewAccountMeta(pool.PoolId, true, false),            // poolId
		solana.NewAccountMeta(fromAccount, true, false),            // inputTokenAccount (is_writable = true, is_signer = false)
		solana.NewAccountMeta(toAccount, true, false),              // outputTokenAccount (is_writable = true, is_signer = false)
		solana.NewAccountMeta(inputValue, true, false),             // inputVault
		solana.NewAccountMeta(outputValue, true, false),            // outputVault
		solana.NewAccountMeta(pool.ObservationKey, true, false),    // observationId
		solana.NewAccountMeta(solana.TokenProgramID, false, false), // TOKEN_PROGRAM_ID
		solana.NewAccountMeta(TOKEN_2022_PROGRAM_ID, false, false), // TOKEN_2022_PROGRAM_ID
		solana.NewAccountMeta(MEMO_PROGRAM_ID, false, false),       // MEMO_PROGRAM_ID
		solana.NewAccountMeta(inputValueMint, false, false),        // inputMint
		solana.NewAccountMeta(outputValueMint, false, false),       // inputMint
	)

	// Add bitmap extension as remaining account if it exists
	exBitmapAddress, _, err := GetPdaExBitmapAccount(RAYDIUM_CLMM_PROGRAM_ID, pool.PoolId)
	if err != nil {
		log.Printf("get pda address error: %v", err)
		return nil, fmt.Errorf("get pda address error: %v", err)
	}
	inst.AccountMetaSlice = append(inst.AccountMetaSlice, solana.NewAccountMeta(exBitmapAddress, true, false)) // exTickArrayBitmap (is_writable = true, is_signer = false)

	// Add tick arrays as remaining accounts
	remainingAccounts, err := pool.GetRemainAccounts(ctx, solClient, inputValueMint.String())
	if err != nil {
		log.Printf("GetRemainAccounts error: %v", err)
		return nil, err
	}

	for _, tickArray := range remainingAccounts {
		inst.AccountMetaSlice = append(inst.AccountMetaSlice, solana.NewAccountMeta(tickArray, true, false)) // tickArrays (is_writable = true, is_signer = false)
	}
	instrs = append(instrs, &inst)

	return instrs, nil
}

type RayCLMMSwapInstruction struct {
	bin.BaseVariant
	Amount                  uint64
	OtherAmountThreshold    uint64
	SqrtPriceLimitX64       uint128.Uint128
	IsBaseInput             bool
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

func (inst *RayCLMMSwapInstruction) ProgramID() solana.PublicKey {
	return RAYDIUM_CLMM_PROGRAM_ID
}

func (inst *RayCLMMSwapInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.AccountMetaSlice
}

func (inst *RayCLMMSwapInstruction) Data() ([]byte, error) {
	// 手动构建指令数据
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
func (p *CLMMPool) GetID() string {
	return p.PoolId.String()
}

// GetTokens returns the base and quote token mints
func (p *CLMMPool) GetTokens() (baseMint, quoteMint string) {
	return p.TokenMint0.String(), p.TokenMint1.String()
}

// GetType returns the pool type
func (p *CLMMPool) GetType() pkg.PoolType {
	return pkg.PoolTypeRaydiumCLMM
}

// GetQuote returns the quote for a given input amount
func (p *CLMMPool) GetQuote(ctx context.Context, inputMint string, inputAmount cosmath.Int) (cosmath.Int, error) {
	// TODO: Implement quote calculation based on pool state
	// This would involve calculating the expected output amount based on the pool's reserves
	// and the input amount, taking into account fees and slippage
	return cosmath.ZeroInt(), nil
}

func (p *CLMMPool) ComputeAmountOutFormat(inputTokenMint string, inputAmount cosmath.Int) (cosmath.Int, error) {

	zeroForOne := inputTokenMint == p.TokenMint0.String()

	firstTickArrayStartIndex, _, err := p.getFirstInitializedTickArray(zeroForOne, p.exTickArrayBitmap)
	if err != nil {
		return cosmath.Int{}, err
	}
	expectedAmountOut, err := p.swapCompute(
		int64(p.TickCurrent),
		zeroForOne,
		inputAmount,
		cosmath.NewIntFromUint64(uint64(p.FeeRate)),
		firstTickArrayStartIndex,
		p.exTickArrayBitmap,
	)
	if err != nil {
		return cosmath.Int{}, err
	}

	return expectedAmountOut, nil
}

func (p *CLMMPool) swapCompute(
	currentTick int64,
	zeroForOne bool,
	amountSpecified cosmath.Int,
	fee cosmath.Int,
	lastSavedTickArrayStartIndex int64,
	exTickArrayBitmap *TickArrayBitmapExtensionType,
) (cosmath.Int, error) {
	if amountSpecified.IsZero() {
		return cosmath.Int{}, errors.New("input amount is zero")
	}

	baseInput := false
	if amountSpecified.IsPositive() {
		baseInput = true
	}

	sqrtPriceLimitX64 := cosmath.NewInt(0)

	amountSpecifiedRemaining := amountSpecified
	amountCalculated := cosmath.NewInt(0)
	amountIn := cosmath.NewInt(0)
	amountOut := cosmath.NewInt(0)
	feeAmount := cosmath.NewInt(0)
	sqrtPriceX64 := cosmath.NewIntFromBigInt(p.SqrtPriceX64.Big())
	tick := int64(0)

	if currentTick > lastSavedTickArrayStartIndex {
		if lastSavedTickArrayStartIndex+getTickCount(int64(p.TickSpacing))-1 < currentTick {
			tick = lastSavedTickArrayStartIndex + getTickCount(int64(p.TickSpacing)) - 1
		} else {
			tick = currentTick
		}
	} else {
		tick = lastSavedTickArrayStartIndex
	}

	accounts := make([]*solana.PublicKey, 0)
	liquidity := cosmath.NewIntFromBigInt(p.Liquidity.Big())
	tickAarrayStartIndex := lastSavedTickArrayStartIndex
	tickArrayCurrent := p.TickArrayCache[strconv.FormatInt(lastSavedTickArrayStartIndex, 10)]

	if baseInput {
		sqrtPriceLimitX64 = MIN_SQRT_PRICE_X64.Add(cosmath.NewInt(1))
	} else {
		sqrtPriceLimitX64 = MAX_SQRT_PRICE_X64.Sub(cosmath.NewInt(1))
	}
	t := !zeroForOne && int64(tickArrayCurrent.StartTickIndex) == tick

	loop := 0
	for {
		if amountSpecifiedRemaining.IsZero() || sqrtPriceX64.Equal(sqrtPriceLimitX64) {
			break
		}

		sqrtPriceStartX64 := sqrtPriceX64
		tickState := getNextInitTick(&tickArrayCurrent, tick, int64(p.TickSpacing), zeroForOne, t)

		nextInitTick := tickState
		tickArrayAddress := &solana.PublicKey{}

		if nextInitTick == nil || nextInitTick.LiquidityGross.Big().Cmp(big.NewInt(0)) <= 0 {
			isExist, nextInitTickArrayIndex, err := nextInitializedTickArrayStartIndexUtils(exTickArrayBitmap,
				tick, int64(p.TickSpacing), p.TickArrayBitmap, zeroForOne)
			if err != nil {
				return cosmath.Int{}, err
			}
			if !isExist {
				return cosmath.Int{}, errors.New("liquidity insufficient")
			}

			tickAarrayStartIndex := nextInitTickArrayIndex
			expectedNextTickArrayAddress := getPdaTickArrayAddress(RAYDIUM_CLMM_PROGRAM_ID, p.PoolId, tickAarrayStartIndex)

			tickArrayAddress = &expectedNextTickArrayAddress
			tickArrayCurrent = p.TickArrayCache[strconv.FormatInt(tickAarrayStartIndex, 10)]
			nextInitTick, err = firstInitializedTick(&tickArrayCurrent, zeroForOne)
			if err != nil {
				return cosmath.Int{}, err
			}
		}

		tickNext := int64(nextInitTick.Tick)
		initialized := nextInitTick.LiquidityGross.Big().Cmp(big.NewInt(0)) > 0
		if lastSavedTickArrayStartIndex != tickAarrayStartIndex && tickArrayAddress != nil {
			accounts = append(accounts, tickArrayAddress)
			lastSavedTickArrayStartIndex = tickAarrayStartIndex
		}

		if tickNext < MIN_TICK {
			tickNext = MIN_TICK
		} else if tickNext > MAX_TICK {
			tickNext = MAX_TICK
		}

		sqrtPriceNextX64, err := getSqrtPriceX64FromTick(int64(tickNext))
		if err != nil {
			return cosmath.Int{}, err
		}

		targetPrice := cosmath.NewInt(0)
		if (zeroForOne && sqrtPriceNextX64.LT(sqrtPriceLimitX64)) ||
			(!zeroForOne && sqrtPriceNextX64.GT(sqrtPriceLimitX64)) {
			targetPrice = sqrtPriceLimitX64
		} else {
			targetPrice = sqrtPriceNextX64
		}

		sqrtPriceX64, amountIn, amountOut, feeAmount = swapStepCompute(
			sqrtPriceX64.BigInt(),
			targetPrice.BigInt(),
			liquidity.BigInt(),
			amountSpecifiedRemaining.BigInt(),
			uint32(fee.Int64()),
			zeroForOne,
		)

		if baseInput {
			amountSpecifiedRemaining = amountSpecifiedRemaining.Sub(amountIn.Add(feeAmount))
			amountCalculated = amountCalculated.Sub(amountOut)
		} else {
			amountSpecifiedRemaining = amountSpecifiedRemaining.Add(amountOut)
			amountCalculated = amountCalculated.Add(amountIn.Add(feeAmount))
		}

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
				return cosmath.Int{}, err
			}
			t = _T != tick && !zeroForOne && int64(tickArrayCurrent.StartTickIndex) == _T
			tick = _T
		}
		loop++
		if loop > 100 {
			panic("1")
		}
	}
	return amountCalculated, nil

}

// GetOutputAmountAndRemainAccounts
func (p CLMMPool) GetRemainAccounts(ctx context.Context, client *rpc.Client, inputTokenMint string) ([]solana.PublicKey, error) {

	// 1. 判断交易方向
	zeroForOne := inputTokenMint == p.TokenMint0.String()

	// 3. 获取第一个初始化的 tick array
	_, firstTickArray, err := p.getFirstInitializedTickArray(zeroForOne, p.exTickArrayBitmap)
	if err != nil {
		return nil, fmt.Errorf("get first tick array error: %v", err)
	}
	allNeededAccounts := make([]solana.PublicKey, 0)
	allNeededAccounts = append(allNeededAccounts, firstTickArray)

	tickAarrayStartIndex, _ := nextInitializedTickArray(
		int64(p.TickCurrent),
		int64(p.TickSpacing),
		zeroForOne,
		p.TickArrayBitmap,
		p.exTickArrayBitmap,
	)

	exTickArrayBitmapAddress := getPdaTickArrayAddress(RAYDIUM_CLMM_PROGRAM_ID, p.PoolId, tickAarrayStartIndex)
	allNeededAccounts = append(allNeededAccounts, exTickArrayBitmapAddress)

	return allNeededAccounts, nil
}

func (p *CLMMPool) GetPrice(amountIn cosmath.Int, inToken string) (cosmath.Int, error) {
	if inToken == p.TokenMint0.String() {
		priceBaseToQuote, err := p.ComputeAmountOutFormat(p.TokenMint0.String(), amountIn)
		if err != nil {
			return cosmath.Int{}, err
		}
		return priceBaseToQuote.Neg(), nil
	} else {
		priceQuoteToBase, err := p.ComputeAmountOutFormat(p.TokenMint1.String(), amountIn)
		if err != nil {
			return cosmath.Int{}, err
		}
		return priceQuoteToBase.Neg(), nil
	}
}
