package protocol

import (
	"context"
	"fmt"
	"log"

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
	programAccounts, err := p.getCPMMPoolAccountsByTokenPair(ctx, baseMint, quoteMint)
	if err != nil {
		log.Printf("GetPoolKeys programAccounts err: %v\n", err)
		return nil, err
	}
	res := make([]*raydium.CPMMPool, 0)
	for _, v := range programAccounts {
		data := v.Account.Data.GetBinary()
		var layout raydium.CPMMPool
		if err := layout.Decode(data); err != nil {
			log.Printf("Failed to decode pool %s: %v", v.Pubkey, err)
			continue
		}
		layout.PoolId = v.Pubkey
		res = append(res, &layout)
	}
	programAccounts, err = p.getCPMMPoolAccountsByTokenPair(ctx, quoteMint, baseMint)
	if err != nil {
		log.Printf("GetPoolKeys programAccounts err: %v\n", err)
		return nil, err
	}
	for _, v := range programAccounts {
		data := v.Account.Data.GetBinary()
		var layout raydium.CPMMPool
		if err := layout.Decode(data); err != nil {
			log.Printf("Failed to decode pool %s: %v", v.Pubkey, err)
			continue
		}
		layout.PoolId = v.Pubkey
		res = append(res, &layout)
	}
	return res, nil
}

func (p *RaydiumCpmmProtocol) getCPMMPoolAccountsByTokenPair(ctx context.Context, baseMint string, quoteMint string) (rpc.GetProgramAccountsResult, error) {
	baseKey := solana.MustPublicKeyFromBase58(baseMint)
	quoteKey := solana.MustPublicKeyFromBase58(quoteMint)

	var layout raydium.CPMMPool

	// 构建过滤条件
	filters := []rpc.RPCFilter{
		{
			DataSize: 637, // 使用实际大小
		},
		{
			// Token0Mint 在偏移量 8 处
			Memcmp: &rpc.RPCFilterMemcmp{
				Offset: layout.Offset("Token0Mint"),
				Bytes:  baseKey.Bytes(),
			},
		},
		{
			// Token1Mint 在偏移量 40 处 (8 + 32)
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
		return nil, fmt.Errorf("failed to get pools: %v", err)
	}

	return result, nil
}

// GetCPMMPoolByPoolId 获取CPMM池子信息, poolRaydium_V4==CPMMPool
func (r *RaydiumCpmmProtocol) GetCPMMPoolByPoolId(ctx context.Context, poolID solana.PublicKey) (*raydium.CPMMPool, error) {
	account, err := r.SolClient.RpcClient.GetAccountInfo(ctx, poolID)
	if err != nil {
		log.Printf("GetAMMPool pool.ID: %s, err: %v\n", poolID.String(), err)
		return nil, fmt.Errorf("failed to get pool account: %v", err)
	}
	layout := &raydium.CPMMPool{}
	err = layout.Decode(account.Value.Data.GetBinary())
	if err != nil {
		log.Printf("GetCPMMPoolByPoolId pool.ID: %s, err: %v\n", poolID.String(), err)
		return nil, fmt.Errorf("failed to decode pool info: %v", err)
	}
	layout.PoolId = poolID

	return layout, nil
}
