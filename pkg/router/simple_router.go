package router

import (
	"context"
	"fmt"
	"log"
	"sync"

	"cosmossdk.io/math"
	"github.com/yimingwow/solroute/pkg"
	"github.com/yimingwow/solroute/pkg/sol"
)

type SimpleRouter struct {
	Protocols []pkg.Protocol
	Pools     []pkg.Pool
}

func NewSimpleRouter(protocols ...pkg.Protocol) *SimpleRouter {
	return &SimpleRouter{
		Protocols: protocols,
		Pools:     []pkg.Pool{},
	}
}

func (r *SimpleRouter) QueryAllPools(ctx context.Context, baseMint, quoteMint string) error {
	var allPools []pkg.Pool

	// Loop through each protocol sequentially
	for _, proto := range r.Protocols {
		log.Printf("😈Fetching pools from protocol: %v", proto.ProtocolName())
		pools, err := proto.FetchPoolsByPair(ctx, baseMint, quoteMint)
		if err != nil {
			log.Printf("error fetching pools from protocol: %v", err)
			continue
		}
		allPools = append(allPools, pools...)
	}

	r.Pools = allPools
	return nil
}

func (r *SimpleRouter) GetBestPool(ctx context.Context, solClient *sol.Client, tokenIn string, amountIn math.Int) (pkg.Pool, math.Int, error) {
	type quoteResult struct {
		pool      pkg.Pool
		outAmount math.Int
		err       error
	}

	// Create a channel to collect results
	resultChan := make(chan quoteResult, len(r.Pools))
	var wg sync.WaitGroup

	// Launch goroutines for each pool
	for _, pool := range r.Pools {
		wg.Add(1)
		go func(p pkg.Pool) {
			defer wg.Done()
			outAmount, err := p.Quote(ctx, solClient, tokenIn, amountIn)
			resultChan <- quoteResult{
				pool:      p,
				outAmount: outAmount,
				err:       err,
			}
		}(pool)
	}

	// Close the channel when all goroutines are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results and find the best one
	var best pkg.Pool
	maxOut := math.NewInt(0)

	for result := range resultChan {
		if result.err != nil {
			log.Printf("error quoting pool %s: %v", result.pool.GetID(), result.err)
			continue
		}
		// if result.outAmount.GT(maxOut) {
		// 	maxOut = result.outAmount
		// 	best = result.pool
		// }
		if result.pool.GetID() == "8sLbNZoA1cfnvMJLPfp98ZLAnFSYCFApfJKMbiXNLwxj" {
			maxOut = result.outAmount
			best = result.pool
		}
	}

	if best == nil {
		return nil, math.ZeroInt(), fmt.Errorf("no route found")
	}
	return best, maxOut, nil
}
