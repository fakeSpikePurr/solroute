package pump

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/yimingwow/solroute/pkg"
	"github.com/yimingwow/solroute/utils"
)

type PumpAMMPool struct {
	Discriminator         [8]uint8 `bin:"skip"`
	PoolBump              uint8    // 第9个字节，应该是252 (fc)
	Index                 uint16
	Creator               solana.PublicKey // 32字节
	BaseMint              solana.PublicKey // 32字节
	QuoteMint             solana.PublicKey // 32字节
	LpMint                solana.PublicKey // 32字节
	PoolBaseTokenAccount  solana.PublicKey // 32字节
	PoolQuoteTokenAccount solana.PublicKey // 32字节
	LpSupply              uint64           // 8字节
	CoinCreator           solana.PublicKey // 32字节

	PoolId           solana.PublicKey
	BaseAmount       math.Int
	QuoteAmount      math.Int
	UserBaseAccount  solana.PublicKey
	UserQuoteAccount solana.PublicKey
}

func (l *PumpAMMPool) Span() uint64 {
	return uint64(300)
}

func (l *PumpAMMPool) Offset(value string) uint64 {
	switch value {
	case "BaseMint":
		return 43
	case "QuoteMint":
		return 43 + 32
	default:
		return 0
	}
}

func (l *PumpAMMPool) Decode(data []byte) error {
	if len(data) < 211 {
		return fmt.Errorf("data too short: expected 211 bytes, got %d", len(data))
	}
	dec := bin.NewBinDecoder(data)
	return dec.Decode(l)
}

func ParsePoolData(data []byte) *PumpAMMPool {
	layout := &PumpAMMPool{}
	// 解析结构
	discriminator := [8]byte{}
	copy(discriminator[:], data[:8])
	layout.PoolBump = uint8(data[8])
	layout.Index = binary.LittleEndian.Uint16(data[9:11])

	offset := 11
	layout.Creator = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.BaseMint = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.QuoteMint = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.LpMint = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.PoolBaseTokenAccount = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.PoolQuoteTokenAccount = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.LpSupply = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	if len(data[offset:]) > 32 {
		layout.CoinCreator = solana.PublicKeyFromBytes(data[offset : offset+32])
	} else {
		layout.CoinCreator = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	}

	return layout
}

func (l *PumpAMMPool) GetID() string {
	return l.PoolId.String()
}

func (l *PumpAMMPool) GetTokens() (string, string) {
	return l.BaseMint.String(), l.QuoteMint.String()
}

func (l *PumpAMMPool) GetType() pkg.PoolType {
	return pkg.PoolTypePumpAMM
}

func (s *PumpAMMPool) BuildSwapInstructions(
	ctx context.Context,
	solClient *rpc.Client,
	user solana.PublicKey,
	inputMint string,
	inputAmount math.Int,
	minOut math.Int,
) ([]solana.Instruction, error) {
	if inputMint == s.BaseMint.String() {
		return s.buyInAMMPool(ctx, user, s, inputAmount, minOut)
	} else {
		return s.sellInAMMPool(ctx, user, s, inputAmount, minOut)
	}
}

func (s *PumpAMMPool) buyInAMMPool(ctx context.Context, userAddr solana.PublicKey, pool *PumpAMMPool,
	maxInputAmountWithDecimals math.Int, outAmountWithDecimals math.Int) ([]solana.Instruction, error) {
	// 初始化指令数组和签名者
	instrs := []solana.Instruction{}

	inst := BuySwapInstruction{
		BaseAmountOut:    outAmountWithDecimals.Uint64(),
		MAxQuoteAmountIn: maxInputAmountWithDecimals.Uint64(),
	}
	if pool.CoinCreator == solana.MustPublicKeyFromBase58("11111111111111111111111111111111") {
		inst.AccountMetaSlice = make(solana.AccountMetaSlice, 17)
	} else {
		inst.AccountMetaSlice = make(solana.AccountMetaSlice, 19)
	}

	inst.BaseVariant = bin.BaseVariant{
		Impl: inst,
	}
	// 确保使用正确的 Token Program 地址
	inst.AccountMetaSlice[0] = solana.NewAccountMeta(pool.PoolId, false, false)
	inst.AccountMetaSlice[1] = solana.NewAccountMeta(userAddr, true, true)
	inst.AccountMetaSlice[2] = solana.NewAccountMeta(PumpGlobalConfig, false, false)
	inst.AccountMetaSlice[3] = solana.NewAccountMeta(pool.BaseMint, false, false)
	inst.AccountMetaSlice[4] = solana.NewAccountMeta(pool.QuoteMint, false, false)
	inst.AccountMetaSlice[5] = solana.NewAccountMeta(pool.UserBaseAccount, true, false)
	inst.AccountMetaSlice[6] = solana.NewAccountMeta(pool.UserQuoteAccount, true, false)
	inst.AccountMetaSlice[7] = solana.NewAccountMeta(pool.PoolBaseTokenAccount, true, false)
	inst.AccountMetaSlice[8] = solana.NewAccountMeta(pool.PoolQuoteTokenAccount, true, false)
	inst.AccountMetaSlice[9] = solana.NewAccountMeta(PumpProtocolFeeRecipient, false, false)
	inst.AccountMetaSlice[10] = solana.NewAccountMeta(PumpProtocolFeeRecipientTokenAccount, true, false)
	tokenProgramID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	inst.AccountMetaSlice[11] = solana.NewAccountMeta(tokenProgramID, false, false)
	inst.AccountMetaSlice[12] = solana.NewAccountMeta(tokenProgramID, false, false)
	inst.AccountMetaSlice[13] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58("11111111111111111111111111111111"), false, false)
	inst.AccountMetaSlice[14] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"), false, false)
	inst.AccountMetaSlice[15] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58("GS4CU59F31iL7aR2Q8zVS8DRrcRnXX1yjQ66TqNVQnaR"), false, false)
	inst.AccountMetaSlice[16] = solana.NewAccountMeta(PumpSwapProgramID, false, false)
	if pool.CoinCreator != solana.MustPublicKeyFromBase58("11111111111111111111111111111111") {
		inst.AccountMetaSlice[17] = solana.NewAccountMeta(coinCreatorVaultATA(pool.CoinCreator), true, false)
		inst.AccountMetaSlice[18] = solana.NewAccountMeta(coinCreatorVaultAuthority(pool.CoinCreator), false, false)
	}
	instrs = append(instrs, &inst)

	return instrs, nil
}

func (s *PumpAMMPool) sellInAMMPool(ctx context.Context, userAddr solana.PublicKey,
	pool *PumpAMMPool, baseAmountIn math.Int, minQuoteAmountOut math.Int) ([]solana.Instruction, error) {
	instrs := []solana.Instruction{}

	inst := SellSwapInstruction{
		BaseAmountIn:      baseAmountIn.Uint64(),
		MinQuoteAmountOut: minQuoteAmountOut.Uint64(),
	}
	if pool.CoinCreator == solana.MustPublicKeyFromBase58("11111111111111111111111111111111") {
		inst.AccountMetaSlice = make(solana.AccountMetaSlice, 17)
	} else {
		inst.AccountMetaSlice = make(solana.AccountMetaSlice, 19)
	}
	inst.BaseVariant = bin.BaseVariant{
		Impl: inst,
	}
	// 确保使用正确的 Token Program 地址
	inst.AccountMetaSlice[0] = solana.NewAccountMeta(pool.PoolId, false, false)
	inst.AccountMetaSlice[1] = solana.NewAccountMeta(userAddr, true, true)
	inst.AccountMetaSlice[2] = solana.NewAccountMeta(PumpGlobalConfig, false, false)
	inst.AccountMetaSlice[3] = solana.NewAccountMeta(pool.BaseMint, false, false)
	inst.AccountMetaSlice[4] = solana.NewAccountMeta(pool.QuoteMint, false, false)
	inst.AccountMetaSlice[5] = solana.NewAccountMeta(pool.UserBaseAccount, true, false)
	inst.AccountMetaSlice[6] = solana.NewAccountMeta(pool.UserQuoteAccount, true, false)
	inst.AccountMetaSlice[7] = solana.NewAccountMeta(pool.PoolBaseTokenAccount, true, false)
	inst.AccountMetaSlice[8] = solana.NewAccountMeta(pool.PoolQuoteTokenAccount, true, false)
	inst.AccountMetaSlice[9] = solana.NewAccountMeta(PumpProtocolFeeRecipient, false, false)
	inst.AccountMetaSlice[10] = solana.NewAccountMeta(PumpProtocolFeeRecipientTokenAccount, true, false)
	tokenProgramID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	inst.AccountMetaSlice[11] = solana.NewAccountMeta(tokenProgramID, false, false)
	inst.AccountMetaSlice[12] = solana.NewAccountMeta(tokenProgramID, false, false)
	inst.AccountMetaSlice[13] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58("11111111111111111111111111111111"), false, false)
	inst.AccountMetaSlice[14] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58("ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL"), false, false)
	inst.AccountMetaSlice[15] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58("GS4CU59F31iL7aR2Q8zVS8DRrcRnXX1yjQ66TqNVQnaR"), false, false)
	inst.AccountMetaSlice[16] = solana.NewAccountMeta(PumpSwapProgramID, false, false)
	if pool.CoinCreator != solana.MustPublicKeyFromBase58("11111111111111111111111111111111") {
		inst.AccountMetaSlice[17] = solana.NewAccountMeta(coinCreatorVaultATA(pool.CoinCreator), false, false)
		inst.AccountMetaSlice[18] = solana.NewAccountMeta(coinCreatorVaultAuthority(pool.CoinCreator), false, false)
	}
	instrs = append(instrs, &inst)

	return instrs, nil
}

type BuySwapInstruction struct {
	bin.BaseVariant
	BaseAmountOut           uint64
	MAxQuoteAmountIn        uint64
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

func (inst *BuySwapInstruction) ProgramID() solana.PublicKey {
	return PumpSwapProgramID
}

func (inst *BuySwapInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.Impl.(solana.AccountsGettable).GetAccounts()
}

func (inst *BuySwapInstruction) Data() ([]byte, error) {

	// 手动构建指令数据
	buf := new(bytes.Buffer)

	// Write discriminator for swap instruction
	namespace := "global"
	name := "buy"
	discriminator := utils.GetDiscriminator(namespace, name)
	if _, err := buf.Write(discriminator); err != nil {
		return nil, fmt.Errorf("failed to write discriminator: %w", err)
	}

	// Write amount
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.BaseAmountOut, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode amount: %w", err)
	}

	// Write other amount threshold
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.MAxQuoteAmountIn, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode other amount threshold: %w", err)
	}

	return buf.Bytes(), nil
}

type SellSwapInstruction struct {
	bin.BaseVariant
	BaseAmountIn            uint64
	MinQuoteAmountOut       uint64
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

func (inst *SellSwapInstruction) ProgramID() solana.PublicKey {
	return PumpSwapProgramID
}

func (inst *SellSwapInstruction) Accounts() (out []*solana.AccountMeta) {
	return inst.Impl.(solana.AccountsGettable).GetAccounts()
}

func (inst *SellSwapInstruction) Data() ([]byte, error) {

	// 手动构建指令数据
	buf := new(bytes.Buffer)

	// Write discriminator for swap instruction
	namespace := "global"
	name := "sell"
	discriminator := utils.GetDiscriminator(namespace, name)
	if _, err := buf.Write(discriminator); err != nil {
		return nil, fmt.Errorf("failed to write discriminator: %w", err)
	}

	// Write amount
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.BaseAmountIn, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode amount: %w", err)
	}

	// Write other amount threshold
	if err := bin.NewBorshEncoder(buf).WriteUint64(inst.MinQuoteAmountOut, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode other amount threshold: %w", err)
	}

	return buf.Bytes(), nil
}

func (pool *PumpAMMPool) GetQuote(ctx context.Context, inputMint string, inputAmount math.Int) (math.Int, error) {

	feeRate := 1 - 0.00250
	feeMultiplier := math.NewInt(int64(feeRate * float64(BaseDecimalInt)))

	// 计算用base兑换quote的outamount
	// 计算 k = baseIn * quoteOut
	k := pool.BaseAmount.Mul(pool.QuoteAmount)

	if inputMint == pool.BaseMint.String() {
		// 计算 newBaseIn = baseIn + amountWithFee
		newBase := pool.BaseAmount.Add(inputAmount.Mul(feeMultiplier).Quo(BaseDecimal))
		// 计算 newQuoteOut = k / newBaseIn
		newQuote := k.Quo(newBase)
		priceBaseToQuote := pool.QuoteAmount.Sub(newQuote)
		return priceBaseToQuote, nil
	} else {
		// 计算 newQuoteIn = quoteIn + amountWithFee
		newQuote2 := pool.QuoteAmount.Add(inputAmount.Mul(feeMultiplier).Quo(BaseDecimal))
		// 计算 newBaseOut = k / newQuoteIn
		newBase2 := k.Quo(newQuote2)
		priceQuoteToBase := pool.BaseAmount.Sub(newBase2)
		return priceQuoteToBase, nil
	}
}
