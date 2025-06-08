package protocol

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/yimingwow/solroute/pkg/raydium"
	"github.com/yimingwow/solroute/pkg/sol"
)

type RaydiumCpmmProtocol struct {
	SolClient *sol.Client
}

func NewRaydiumCpmm(solClient *sol.Client) *RaydiumCpmmProtocol {
	return &RaydiumCpmmProtocol{
		SolClient: solClient,
	}
}

func (p *RaydiumCpmmProtocol) GetCPMMPoolByTokenPair(ctx context.Context, baseMint string, quoteMint string) ([]*raydium.CPMMPool, error) {
	res := make([]*raydium.CPMMPool, 0)

	// Fetch pools with baseMint as token0
	programAccounts, err := p.getCPMMPoolAccountsByTokenPair(ctx, baseMint, quoteMint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pools with base token %s: %w", baseMint, err)
	}
	for _, v := range programAccounts {
		data := v.Account.Data.GetBinary()
		var layout raydium.CPMMPool
		if err := layout.Decode(data); err != nil {
			continue
		}
		layout.PoolId = v.Pubkey
		res = append(res, &layout)
	}

	// Fetch pools with quoteMint as token0
	programAccounts, err = p.getCPMMPoolAccountsByTokenPair(ctx, quoteMint, baseMint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pools with base token %s: %w", quoteMint, err)
	}
	for _, v := range programAccounts {
		data := v.Account.Data.GetBinary()
		var layout raydium.CPMMPool
		if err := layout.Decode(data); err != nil {
			continue
		}
		layout.PoolId = v.Pubkey
		res = append(res, &layout)
	}
	return res, nil
}

func (p *RaydiumCpmmProtocol) getCPMMPoolAccountsByTokenPair(ctx context.Context, baseMint string, quoteMint string) (rpc.GetProgramAccountsResult, error) {
	baseKey, err := solana.PublicKeyFromBase58(baseMint)
	if err != nil {
		return nil, fmt.Errorf("invalid base mint address: %w", err)
	}
	quoteKey, err := solana.PublicKeyFromBase58(quoteMint)
	if err != nil {
		return nil, fmt.Errorf("invalid quote mint address: %w", err)
	}

	var layout raydium.CPMMPool
	filters := []rpc.RPCFilter{
		{
			DataSize: 637,
		},
		{
			Memcmp: &rpc.RPCFilterMemcmp{
				Offset: layout.Offset("Token0Mint"),
				Bytes:  baseKey.Bytes(),
			},
		},
		{
			Memcmp: &rpc.RPCFilterMemcmp{
				Offset: layout.Offset("Token1Mint"),
				Bytes:  quoteKey.Bytes(),
			},
		},
	}

	result, err := p.SolClient.RpcClient.GetProgramAccountsWithOpts(ctx, raydium.RAYDIUM_CPMM_PROGRAM_ID, &rpc.GetProgramAccountsOpts{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %w", err)
	}

	return result, nil
}

// GetCPMMPoolByPoolId 获取CPMM池子信息, poolRaydium_V4==CPMMPool
func (r *RaydiumCpmmProtocol) GetCPMMPoolByPoolId(ctx context.Context, poolID solana.PublicKey) (*raydium.CPMMPool, error) {
	account, err := r.SolClient.RpcClient.GetAccountInfo(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool account %s: %w", poolID.String(), err)
	}

	layout := &raydium.CPMMPool{}
	if err := layout.Decode(account.Value.Data.GetBinary()); err != nil {
		return nil, fmt.Errorf("failed to decode pool data for %s: %w", poolID.String(), err)
	}
	layout.PoolId = poolID

	return layout, nil
}
