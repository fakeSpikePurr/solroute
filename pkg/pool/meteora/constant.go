package meteora

import (
	"math/big"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"lukechampine.com/uint128"
)

// Array and bitmap size constants
const (
	TickArraySize                = 60
	TickSize                     = 168
	TickArrayBitmapSize          = 512
	ExtensionTickArrayBitmapSize = 14
	BinArrayBitmapSize           = 512
	ExtensionBinArrayBitmapSize  = 12
)

// Tick and bin ID range constants
const (
	MaxTick  = 443636
	MinTick  = -443636
	MaxBinID = 443636
	MinBinID = -443636
)

// Resolution and precision constants
const (
	U64Resolution = 64
	ScaleOffset   = 64
	FeePrecision  = 1_000_000_000
	MaxFeeRate    = 100_000_000
)

// Basis point constants
const (
	BasisPointMax = 10000
)

// Program IDs and system constants
var (
	// MeteoraProgramID is the main Meteora DLMM program ID
	MeteoraProgramID = solana.MustPublicKeyFromBase58("LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo")

	// MemoProgramID is the Solana memo program ID
	MemoProgramID = solana.MustPublicKeyFromBase58("MemoSq4gqABAXKb96qnH8TysNcWxMyWCqXgDLGmfcHr")

	// MinSqrtPriceX64 represents the minimum square root price in X64 format
	MinSqrtPriceX64 = math.NewIntFromBigInt(big.NewInt(4295048016))

	// One represents 1.0 in the scaled format (1 << ScaleOffset)
	One = uint128.From64(1).Lsh(uint(ScaleOffset))

	// Swap2IxDiscm is the instruction discriminator for swap2 instruction
	Swap2IxDiscm = [8]byte{65, 75, 63, 76, 235, 91, 91, 136}
)

// PairStatus represents the status of a trading pair
type PairStatus uint8

const (
	PairStatusEnabled  PairStatus = iota // Pair is active and can be used for trading
	PairStatusDisabled                   // Pair is disabled and cannot be used for trading
)

// PairType represents the type of trading pair
type PairType uint8

const (
	PairTypePermissionless             PairType = iota // Anyone can create liquidity
	PairTypePermission                                 // Only authorized users can create liquidity
	PairTypeCustomizablePermissionless                 // Permissionless with customizable parameters
	PairTypePermissionlessV2                           // Updated permissionless pair type
)

// ActivationType represents how a pair becomes active
type ActivationType uint8

const (
	ActivationTypeSlot      ActivationType = iota // Activated at a specific slot
	ActivationTypeTimestamp                       // Activated at a specific timestamp
)
