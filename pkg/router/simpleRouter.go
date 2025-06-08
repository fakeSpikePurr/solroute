package router

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/yimingwow/solroute/pkg"
)

type SimpleRouter struct {
	protocols []pkg.Protocol
}

func NewSimpleRouter(protocols ...pkg.Protocol) *SimpleRouter {
	return &SimpleRouter{protocols: protocols}
}

func (r *SimpleRouter) QueryAllPools(baseMint, quoteMint string) ([]pkg.Pool, error) {
	var allPools []pkg.Pool
	for _, proto := range r.protocols {
		pools, err := proto.FetchPoolsByPair(baseMint, quoteMint)
		if err != nil {
			continue
		}
		allPools = append(allPools, pools...)
	}
	return allPools, nil
}

func (r *SimpleRouter) GetBestPool(
	ctx context.Context,
	tokenIn, tokenOut string,
	amountIn math.Int,
) (pkg.Pool, math.Int, error) {
	var best pkg.Pool
	var maxOut math.Int

	for _, p := range r.protocols {
		pools, err := p.FetchPoolsByPair(tokenIn, tokenOut)
		if err != nil {
			continue
		}

		for _, pool := range pools {
			outAmount, err := pool.GetQuote(ctx, tokenIn, amountIn)
			if err != nil {
				continue
			}

			if outAmount.GT(maxOut) {
				maxOut = outAmount
				best = pool
			}
		}
	}

	if best == nil {
		return nil, math.ZeroInt(), fmt.Errorf("no route found")
	}

	return best, maxOut, nil
}
