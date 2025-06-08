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
	"github.com/yimingwow/solroute/utils"
)

func (s *PumpAMMPool) BuildSwapInstructions(
	ctx context.Context,
	solClient *rpc.Client,
	user solana.PublicKey,
	inputMint string,
	inputAmount math.Int,
	minOut math.Int,
) ([]solana.Instruction, []solana.PrivateKey, error) {
	if inputMint == s.BaseMint.String() {
		return s.buyInAMMPool(ctx, user, s, inputAmount, minOut)
	} else {
		return s.sellInAMMPool(ctx, user, s, inputAmount, minOut)
	}
}

func (s *PumpAMMPool) buyInAMMPool(ctx context.Context, userAddr solana.PublicKey, pool *PumpAMMPool,
	maxInputAmountWithDecimals math.Int, outAmountWithDecimals math.Int) ([]solana.Instruction, []solana.PrivateKey, error) {
	// 初始化指令数组和签名者
	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{}

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

	return instrs, signers, nil
}

func (s *PumpAMMPool) sellInAMMPool(ctx context.Context, userAddr solana.PublicKey,
	pool *PumpAMMPool, baseAmountIn math.Int, minQuoteAmountOut math.Int) ([]solana.Instruction, []solana.PrivateKey, error) {
	instrs := []solana.Instruction{}
	signers := []solana.PrivateKey{}

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

	return instrs, signers, nil
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
