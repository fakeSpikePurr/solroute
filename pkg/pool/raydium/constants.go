package raydium

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
)

// Program IDs
var (
	// Token Program IDs
	TOKEN_2022_PROGRAM_ID = solana.MustPublicKeyFromBase58("TokenzQdBNbLqP5VEhdkAS6EPFLC1PHnBqCXEpPxuEb")
	MEMO_PROGRAM_ID       = solana.MustPublicKeyFromBase58("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")

	// Raydium Program IDs
	RAYDIUM_AMM_PROGRAM_ID  = solana.MustPublicKeyFromBase58("675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8")
	RAYDIUM_CPMM_PROGRAM_ID = solana.MustPublicKeyFromBase58("CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C")
	RAYDIUM_CLMM_PROGRAM_ID = solana.MustPublicKeyFromBase58("CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK")
)

// Tick Array Configuration
const (
	TICK_ARRAY_SIZE                 = 60
	TickSize                        = 168
	TICK_ARRAY_BITMAP_SIZE          = 512
	MAX_TICK                        = 443636
	MIN_TICK                        = -443636
	EXTENSION_TICKARRAY_BITMAP_SIZE = 14
	U64Resolution                   = 64
)

// Price Constants
var (
	MIN_SQRT_PRICE_X64    = math.NewIntFromBigInt(big.NewInt(4295048016))
	MAX_SQRT_PRICE_X64, _ = math.NewIntFromString("79226673521066979257578248091")
	FEE_RATE_DENOMINATOR  = math.NewInt(int64(1000000))
)

// Liquidity Constants
var (
	LIQUIDITY_FEES_NUMERATOR   = math.NewInt(25)
	LIQUIDITY_FEES_DENOMINATOR = math.NewInt(10000)
)

// Seeds and Discriminators
var (
	AUTH_SEED                  = "vault_and_lp_mint_auth_seed"
	SwapBaseInputDiscriminator = []byte{143, 190, 90, 218, 196, 30, 51, 222}
)
