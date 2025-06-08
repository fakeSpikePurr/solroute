package pump

import (
	"context"

	"cosmossdk.io/math"
)

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
