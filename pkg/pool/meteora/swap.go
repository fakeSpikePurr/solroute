package meteora

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"

	"cosmossdk.io/math"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/solana-zh/solroute/pkg/sol"
)

// BuildSwapInstructions creates Solana instructions for performing a swap operation
func (pool *MeteoraDlmmPool) BuildSwapInstructions(
	ctx context.Context,
	solClient *sol.Client,
	user solana.PublicKey,
	inputMint string,
	inputAmount math.Int,
	minOut math.Int,
	userBaseAccount solana.PublicKey,
	userQuoteAccount solana.PublicKey,
) ([]solana.Instruction, error) {
	instructions := []solana.Instruction{}

	var userInTokenAccount solana.PublicKey
	var userOutTokenAccount solana.PublicKey
	log.Printf("inputMint: %v, pool.TokenXMint: %v,if:%v", inputMint, pool.TokenXMint.String(), inputMint == pool.TokenXMint.String())
	if inputMint == pool.TokenXMint.String() {
		userInTokenAccount = userBaseAccount
		userOutTokenAccount = userQuoteAccount
	} else {
		userInTokenAccount = userQuoteAccount
		userOutTokenAccount = userBaseAccount
	}

	instruction := SwapInstruction{
		AmountIn:         inputAmount.Uint64(),
		MinAmountOut:     minOut.Uint64(),
		AccountMetaSlice: make(solana.AccountMetaSlice, 16+len(pool.BinArrays)),
		RemainingAccountsInfo: RemainingAccountsInfo{
			Slices: []RemainingAccountsSlice{
				{
					AccountsType: AccountsTypeTransferHookX,
					Length:       0, // Set as needed
				},
				{
					AccountsType: AccountsTypeTransferHookY,
					Length:       0, // Set as needed
				},
			},
		},
	}
	instruction.BaseVariant = bin.BaseVariant{
		Impl: instruction,
	}

	// Ensure correct Token Program address is used
	instruction.AccountMetaSlice[0] = solana.NewAccountMeta(pool.PoolId, true, false)
	if pool.bitmapExtension != nil {
		instruction.AccountMetaSlice[1] = solana.NewAccountMeta(pool.BitmapExtensionKey, false, false)
	} else {
		instruction.AccountMetaSlice[1] = solana.NewAccountMeta(MeteoraProgramID, false, false)
	}
	instruction.AccountMetaSlice[2] = solana.NewAccountMeta(pool.reserveX, true, false)
	instruction.AccountMetaSlice[3] = solana.NewAccountMeta(pool.reserveY, true, false)
	instruction.AccountMetaSlice[4] = solana.NewAccountMeta(userInTokenAccount, true, false)
	instruction.AccountMetaSlice[5] = solana.NewAccountMeta(userOutTokenAccount, true, false)
	instruction.AccountMetaSlice[6] = solana.NewAccountMeta(pool.TokenXMint, false, false)
	instruction.AccountMetaSlice[7] = solana.NewAccountMeta(pool.TokenYMint, false, false)
	instruction.AccountMetaSlice[8] = solana.NewAccountMeta(pool.oracle, true, false)
	instruction.AccountMetaSlice[9] = solana.NewAccountMeta(MeteoraProgramID, false, false) // Host fee account - set to null in JS SDK but not in Rust SDK
	instruction.AccountMetaSlice[10] = solana.NewAccountMeta(user, true, true)
	tokenProgramID := solana.MustPublicKeyFromBase58("TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA")
	instruction.AccountMetaSlice[11] = solana.NewAccountMeta(tokenProgramID, false, false)
	instruction.AccountMetaSlice[12] = solana.NewAccountMeta(tokenProgramID, false, false)
	instruction.AccountMetaSlice[13] = solana.NewAccountMeta(MemoProgramID, false, false)
	instruction.AccountMetaSlice[14] = solana.NewAccountMeta(DeriveEventAuthorityPDA(), false, false)
	instruction.AccountMetaSlice[15] = solana.NewAccountMeta(MeteoraProgramID, true, false)

	index := 16
	for binArrayKey := range pool.BinArrays {
		instruction.AccountMetaSlice[index] = solana.NewAccountMeta(solana.MustPublicKeyFromBase58(binArrayKey), true, false)
		index++
	}

	instructions = append(instructions, &instruction)

	return instructions, nil
}

// AccountsType represents the type of accounts in the remaining accounts slice
type AccountsType uint8

const (
	AccountsTypeTransferHookX AccountsType = iota
	AccountsTypeTransferHookY
)

// RemainingAccountsSlice represents a slice of remaining accounts with type and length
type RemainingAccountsSlice struct {
	AccountsType AccountsType
	Length       uint8
}

// RemainingAccountsInfo contains information about remaining accounts slices
type RemainingAccountsInfo struct {
	Slices []RemainingAccountsSlice // Define based on actual SliceInfo structure if needed
}

// SwapInstruction represents a Meteora swap instruction
type SwapInstruction struct {
	bin.BaseVariant
	AmountIn                uint64                `bin:"amount_in"`
	MinAmountOut            uint64                `bin:"min_amount_out"`
	RemainingAccountsInfo   RemainingAccountsInfo `bin:"remaining_accounts_info"`
	solana.AccountMetaSlice `bin:"-" borsh_skip:"true"`
}

// ProgramID returns the Meteora program ID
func (instruction *SwapInstruction) ProgramID() solana.PublicKey {
	return MeteoraProgramID
}

// Accounts returns the account metadata for the instruction
func (instruction *SwapInstruction) Accounts() (out []*solana.AccountMeta) {
	return instruction.Impl.(solana.AccountsGettable).GetAccounts()
}

// Data serializes the instruction data for on-chain execution
func (instruction *SwapInstruction) Data() ([]byte, error) {
	// Commented out code for reference:
	// namespace := "global"
	// name := "swap2"
	// discriminator := sol.GetDiscriminator(namespace, name)

	buffer := new(bytes.Buffer)
	if _, err := buffer.Write(Swap2IxDiscm[:]); err != nil {
		return nil, fmt.Errorf("failed to write discriminator: %w", err)
	}

	if err := bin.NewBorshEncoder(buffer).WriteUint64(instruction.AmountIn, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode amount: %w", err)
	}

	// Write minimum amount out threshold
	if err := bin.NewBorshEncoder(buffer).WriteUint64(instruction.MinAmountOut, binary.LittleEndian); err != nil {
		return nil, fmt.Errorf("failed to encode minimum amount out threshold: %w", err)
	}

	if err := bin.NewBorshEncoder(buffer).Encode(instruction.RemainingAccountsInfo); err != nil {
		return nil, fmt.Errorf("failed to encode remaining accounts info: %w", err)
	}

	return buffer.Bytes(), nil
}
