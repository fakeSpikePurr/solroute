// Package protocol provides implementations for different DeFi protocols
package protocol

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/solana-zh/solroute/pkg"
	"github.com/solana-zh/solroute/pkg/pool/meteora"
	"github.com/solana-zh/solroute/pkg/sol"
)

// MeteoraDlmmProtocol handles interactions with Meteora DLMM (Dynamic Liquidity Market Maker) pools
type MeteoraDlmmProtocol struct {
	SolClient *sol.Client
}

// NewMeteoraDlmm creates a new MeteoraDlmmProtocol instance
func NewMeteoraDlmm(solClient *sol.Client) *MeteoraDlmmProtocol {
	return &MeteoraDlmmProtocol{
		SolClient: solClient,
	}
}

func (protocol *MeteoraDlmmProtocol) ProtocolName() pkg.ProtocolName {
	return pkg.ProtocolNameMeteoraDlmm
}

// FetchPoolsByPair retrieves all Meteora DLMM pools for a given token pair
func (protocol *MeteoraDlmmProtocol) FetchPoolsByPair(ctx context.Context, baseMint string, quoteMint string) ([]pkg.Pool, error) {
	programAccounts := rpc.GetProgramAccountsResult{}

	// Fetch pools with baseMint as TokenX and quoteMint as TokenY
	baseQuotePools, err := protocol.getMeteoraDlmmPoolAccountsByTokenPair(ctx, baseMint, quoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pools with baseMint as TokenX: %w", err)
	}
	programAccounts = append(programAccounts, baseQuotePools...)

	pools := make([]pkg.Pool, 0, len(programAccounts))
	for _, account := range programAccounts {
		poolData := &meteora.MeteoraDlmmPool{}
		if err := poolData.Decode(account.Account.Data.GetBinary()); err != nil {
			// Skip pools that can't be decoded
			continue
		}

		poolData.PoolId = account.Pubkey
		if err := poolData.GetBinArrayForSwap(ctx, protocol.SolClient); err != nil {
			// Skip pools that can't get bin array
			continue
		}

		poolData.BitmapExtensionKey, _ = meteora.DeriveBinArrayBitmapExtension(poolData.PoolId)
		pools = append(pools, poolData)
	}
	return pools, nil
}

// getMeteoraDlmmPoolAccountsByTokenPair retrieves pool accounts for a specific token pair configuration
func (protocol *MeteoraDlmmProtocol) getMeteoraDlmmPoolAccountsByTokenPair(ctx context.Context, baseMint string, quoteMint string) (rpc.GetProgramAccountsResult, error) {
	var poolLayout meteora.MeteoraDlmmPool
	result, err := protocol.SolClient.GetProgramAccountsWithOpts(ctx, meteora.MeteoraProgramID, &rpc.GetProgramAccountsOpts{
		Filters: []rpc.RPCFilter{
			{
				DataSize: 904, // Meteora DLMM pool account size
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: poolLayout.Offset("TokenXMint"),
					Bytes:  solana.MustPublicKeyFromBase58(baseMint).Bytes(),
				},
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: poolLayout.Offset("TokenYMint"),
					Bytes:  solana.MustPublicKeyFromBase58(quoteMint).Bytes(),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get program accounts: %w", err)
	}
	return result, nil
}

// FetchPoolByID retrieves a specific Meteora DLMM pool by its ID
func (protocol *MeteoraDlmmProtocol) FetchPoolByID(ctx context.Context, poolID string) (pkg.Pool, error) {
	poolData := &meteora.MeteoraDlmmPool{}
	account, err := protocol.SolClient.GetAccountInfoWithOpts(ctx, solana.MustPublicKeyFromBase58(poolID))
	if err != nil {
		return nil, fmt.Errorf("failed to get pool account: %w", err)
	}

	if err := poolData.Decode(account.Value.Data.GetBinary()); err != nil {
		return nil, fmt.Errorf("failed to decode pool data: %w", err)
	}

	if err := poolData.GetBinArrayForSwap(ctx, protocol.SolClient); err != nil {
		return nil, fmt.Errorf("failed to get bin array for swap: %w", err)
	}

	bitmapExtensionKey, _ := meteora.DeriveBinArrayBitmapExtension(poolData.PoolId)
	poolData.BitmapExtensionKey = bitmapExtensionKey
	return poolData, nil
}
