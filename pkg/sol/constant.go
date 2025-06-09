package sol

import "github.com/gagliardetto/solana-go"

var (
	WSOL      = solana.MustPublicKeyFromBase58("So11111111111111111111111111111111111111112")
	NativeSOL = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")

	TokenAccountSize = uint64(165)
)
