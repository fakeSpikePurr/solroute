package meteora

import (
	"fmt"
	"math/big"

	"lukechampine.com/uint128"
)

// Bin represents a liquidity bin in the Meteora DLMM protocol
// Each bin contains liquidity for a specific price range
type Bin struct {
	amountX                  uint64
	amountY                  uint64
	price                    uint128.Uint128
	liquiditySupply          uint128.Uint128
	rewardPerTokenStored     [2]uint128.Uint128
	feeAmountXPerTokenStored uint128.Uint128
	feeAmountYPerTokenStored uint128.Uint128
	amountXIn                uint128.Uint128
	amountYIn                uint128.Uint128
}

// IsEmpty checks if the bin is empty for the specified token
func (bin *Bin) IsEmpty(isX bool) bool {
	if isX {
		return bin.amountX == 0
	}
	return bin.amountY == 0
}

// GetMaxAmountOut returns the maximum amount that can be swapped out for the given direction
func (bin *Bin) GetMaxAmountOut(swapForY bool) uint64 {
	if swapForY {
		return bin.amountY
	}
	return bin.amountX
}

// GetAmountOut calculates the output amount for a given input amount and price
// Uses rounding down for both swap directions
func (bin *Bin) GetAmountOut(amountIn uint64, price uint128.Uint128, swapForY bool) (*big.Int, error) {
	if swapForY {
		// Calculate: price * amountIn >> SCALE_OFFSET (rounding down)
		return SafeMulShrCast(
			price.Big(),
			big.NewInt(int64(amountIn)),
			ScaleOffset,
			RoundingDown,
		)
	}

	// Calculate: (amountIn << SCALE_OFFSET) / price (rounding down)
	return SafeShlDivCast(
		big.NewInt(int64(amountIn)),
		price.Big(),
		ScaleOffset,
		RoundingDown,
	)
}

// GetMaxAmountIn calculates the maximum input amount that can be swapped for the given price
// Uses rounding up for both swap directions
func (bin *Bin) GetMaxAmountIn(price uint128.Uint128, swapForY bool) (*big.Int, error) {
	if swapForY {
		// Calculate: amountY << SCALE_OFFSET / price (rounding up)
		return SafeShlDivCast(
			big.NewInt(int64(bin.amountY)),
			price.Big(),
			ScaleOffset,
			RoundingUp,
		)
	}

	// Calculate: amountX * price >> SCALE_OFFSET (rounding up)
	return SafeMulShrCast(
		big.NewInt(int64(bin.amountX)),
		price.Big(),
		ScaleOffset,
		RoundingUp,
	)
}

// GetOrStoreBinPrice retrieves the bin price, computing it from ID if not already stored
func (bin *Bin) GetOrStoreBinPrice(id int32, binStep uint16) (uint128.Uint128, error) {
	if bin.price.IsZero() {
		price, err := GetPriceFromID(id, binStep)
		if err != nil {
			return uint128.Zero, fmt.Errorf("failed to get price from id: %w", err)
		}
		bin.price = price
	}

	return bin.price, nil
}
