package pump

import (
	"github.com/gagliardetto/solana-go"
	"github.com/yimingwow/solroute/pkg/sol"
)

// CoinCreatorVaultAuthority 计算 coin creator 的 vault authority PDA
func coinCreatorVaultAuthority(coinCreator solana.PublicKey) solana.PublicKey {
	seeds := [][]byte{
		[]byte("creator_vault"),
		coinCreator.Bytes(),
	}

	// 计算 PDA
	pda, _, err := solana.FindProgramAddress(seeds, PumpSwapProgramID)
	if err != nil {
		panic(err)
	}

	return pda
}

// CoinCreatorVaultATA 计算 coin creator 的 vault authority 的 Associated Token Account
func coinCreatorVaultATA(coinCreator solana.PublicKey) solana.PublicKey {
	creatorVaultAuthority := coinCreatorVaultAuthority(coinCreator)

	// 计算 Associated Token Account
	// 使用 associatedtokenaccount 包中的方法
	ata, _, err := solana.FindAssociatedTokenAddress(
		creatorVaultAuthority, // owner
		sol.WSOL,              // mint
	)
	if err != nil {
		panic(err)
	}

	return ata
}
