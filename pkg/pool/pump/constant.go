package pump

import (
	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
)

var (
	PumpSwapProgramID                    = solana.MustPublicKeyFromBase58("pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA")
	PumpGlobalConfig                     = solana.MustPublicKeyFromBase58("ADyA8hdefvWN2dbGGWFotbzWxrAvLW83WG6QCVXvJKqw")
	PumpProtocolFeeRecipient             = solana.MustPublicKeyFromBase58("62qc2CNXwrYqQScmEdiZFFAnJR262PxWEuNQtxfafNgV")
	PumpProtocolFeeRecipientTokenAccount = solana.MustPublicKeyFromBase58("94qWNrtmfn42h3ZjUZwWvK1MEo9uVmmrBPd2hpNjYDjb")
)

var (
	BaseDecimalInt = 1000000000                   // 1*10^9
	BaseDecimal    = math.NewIntWithDecimal(1, 9) // 1*10^9
)
