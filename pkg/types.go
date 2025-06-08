package pkg

import (
	"time"

	"cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
)

type PoolType string

const (
	PoolTypeRaydiumAMM  PoolType = "RaydiumAMM"
	PoolTypeRaydiumCPMM PoolType = "RaydiumCPMM"
	PoolTypeRaydiumCLMM PoolType = "RaydiumCLMM"
	PoolTypePumpAMM     PoolType = "PumpAMM"
)

type PoolInfo struct {
	Protocol string `json:"protocol"` // "Raydium", "Orca", "Meteora"

	Type             string           `json:"type"`      // "Standard", "Concentrated"
	ProgramID        solana.PublicKey `json:"programId"` // 程序ID
	ID               solana.PublicKey `json:"id"`        // 池地址
	BaseMint         solana.PublicKey `json:"baseMint"`  // A token信息
	QuoteMint        solana.PublicKey `json:"quoteMint"` // B token信息
	BaseVault        solana.PublicKey `json:"baseVault"`
	QuoteVault       solana.PublicKey `json:"quoteVault"`
	PriceBaseToQuote math.Int         `json:"priceBaseToQuote"` // 当前价格
	PriceQuoteToBase math.Int         `json:"priceQuoteToBase"` // 当前价格
	BaseAmount       math.Int         `json:"baseAmount"`       // A token数量
	QuoteAmount      math.Int         `json:"quoteAmount"`      // B token数量
	FeeRate          math.Int         `json:"feeRate"`          // 手续费率
	BaseDecimals     uint64           `json:"baseDecimals"`
	QuoteDecimals    uint64           `json:"quoteDecimals"`

	// for raydium
	PoolType         []string         `json:"pooltype"` // 池类型列表
	Version          uint64           `json:"version"`
	Authority        solana.PublicKey `json:"authority"`
	OpenOrders       solana.PublicKey `json:"openOrders"`
	TargetOrders     solana.PublicKey `json:"targetOrders"`
	WithdrawQueue    solana.PublicKey `json:"withdrawQueue"`
	MarketProgramId  solana.PublicKey `json:"marketProgramId"`
	MarketId         solana.PublicKey `json:"marketId"`
	MarketAuthority  solana.PublicKey `json:"marketAuthority"`
	MarketBaseVault  solana.PublicKey `json:"marketBaseVault"`
	MarketQuoteVault solana.PublicKey `json:"marketQuoteVault"`
	MarketBids       solana.PublicKey `json:"marketBids"`
	MarketAsks       solana.PublicKey `json:"marketAsks"`
	MarketEventQueue solana.PublicKey `json:"marketEventQueue"`

	// for Raydium clmm
	AmmConfig          solana.PublicKey `json:"ammConfig"`
	ObservationKey     solana.PublicKey `json:"observationKey"`
	TickSpacing        int32            `json:"tickSpacing"`
	TickCurrent        int32            `json:"tickCurrent"`
	TickArrayBitmap    [16]uint64       `json:"tickArrayBitmap"`
	SqrtPriceX64       math.Int         `json:"sqrtPriceX64"`
	Liquidity          math.Int         `json:"liquidity"`
	SwapFeeNumerator   uint64           `json:"swapFeeNumerator"`
	SwapFeeDenominator uint64           `json:"swapFeeDenominator"`

	// for meteora clmm
	Oracle            solana.PublicKey `json:"oracle"`
	Status            uint8            `json:"status"`
	PairType          uint8            `json:"pairType"`
	ActivationType    uint8            `json:"activationType"`
	ActivationPoint   uint64           `json:"activationPoint"`
	ActiveBinArrayKey solana.PublicKey `json:"activeBinArrayKey"`

	UpdateTime           time.Time `json:"updateTime"`           // 更新时间
	PayToken0Amount      math.Int  `json:"payToken0Amount"`      // 需要支付的token0数量
	ObatinedToken1Amount math.Int  `json:"obatinedToken1Amount"` // 卖出token0 应该得到的token1数量
	PayToken1Amount      math.Int  `json:"payToken1Amount"`      // 需要支付的token1数量
	ObatinedToken0Amount math.Int  `json:"obatinedToken0Amount"` // 卖出token1 应该得到的token0数量
}
