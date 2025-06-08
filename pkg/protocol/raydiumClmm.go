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

type RaydiumClmmProtocol struct {
	SolClient *sol.Client
}

func NewRaydiumClmm(solClient *sol.Client) *RaydiumClmmProtocol {
	return &RaydiumClmmProtocol{
		SolClient: solClient,
	}
}

func (p *RaydiumClmmProtocol) GetCLMMPoolByPair(ctx context.Context, baseMint string, quoteMint string) ([]*raydium.CLMMPool, error) {
	accounts := make([]*rpc.KeyedAccount, 0)
	programAccounts, err := p.getCLMMPoolAccountsByTokenPair(ctx, baseMint, quoteMint)
	if err != nil {
		log.Printf("getCLMMPoolAccountsByTokenPair err: %v\n", err)
		return nil, err
	}
	accounts = append(accounts, programAccounts...)
	programAccounts, err = p.getCLMMPoolAccountsByTokenPair(ctx, quoteMint, baseMint)
	if err != nil {
		log.Printf("getCLMMPoolAccountsByTokenPair err: %v\n", err)
		return nil, err
	}
	accounts = append(accounts, programAccounts...)

	res := make([]*raydium.CLMMPool, 0)
	for _, v := range accounts {
		data := v.Account.Data.GetBinary()
		layout := &raydium.CLMMPool{}
		err := layout.Decode(data)
		if err != nil {
			log.Printf("decode CLMMPool err: %v\n", err)
			continue
		}
		layout.PoolId = v.Pubkey

		ammConfigData, err := p.SolClient.RpcClient.GetAccountInfo(ctx, layout.AmmConfig)
		if err != nil {
			log.Printf("Failed to get amm config: %v", err)
			continue
		}
		feeRate, err := parseAmmConfig(ammConfigData.Value.Data.GetBinary())
		if err != nil {
			log.Printf("parseAmmConfig: %v", err)
			continue
		}
		layout.FeeRate = feeRate
		exBitmapAddress, _, err := raydium.GetPdaExBitmapAccount(raydium.RAYDIUM_CLMM_PROGRAM_ID, layout.PoolId)
		if err != nil {
			log.Printf("get pda address error: %v", err)
			continue
		}
		layout.ExBitmapAddress = exBitmapAddress

		res = append(res, layout)
	}
	return res, nil
}

func (p *RaydiumClmmProtocol) getCLMMPoolAccountsByTokenPair(ctx context.Context, baseMint string, quoteMint string) (rpc.GetProgramAccountsResult, error) {

	baseKey := solana.MustPublicKeyFromBase58(baseMint)
	quoteKey := solana.MustPublicKeyFromBase58(quoteMint)

	// Now try with filters
	var knownPoolLayout raydium.CLMMPool
	result, err := p.SolClient.RpcClient.GetProgramAccountsWithOpts(ctx, raydium.RAYDIUM_CLMM_PROGRAM_ID, &rpc.GetProgramAccountsOpts{
		Filters: []rpc.RPCFilter{
			{
				DataSize: uint64(knownPoolLayout.Span()), // Use actual size from known pool
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: knownPoolLayout.Offset("TokenMint0"),
					Bytes:  baseKey.Bytes(),
				},
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: knownPoolLayout.Offset("TokenMint1"),
					Bytes:  quoteKey.Bytes(),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pools: %v", err)
	}

	return result, nil
}

// GetCLMMPoolByPoolId 获取并解析池子信息
func (r *RaydiumClmmProtocol) GetCLMMPoolByPoolId(ctx context.Context, poolId solana.PublicKey) (*raydium.CLMMPool, error) {
	account, err := r.SolClient.RpcClient.GetAccountInfo(ctx, poolId)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool account: %v", err)
	}

	data := account.Value.Data.GetBinary()
	layout := &raydium.CLMMPool{}
	err = layout.Decode(data)
	if err != nil {
		log.Printf("GetAMMPool pool.ID: %s, err: %v\n", poolId.String(), err)
		return nil, fmt.Errorf("failed to decode pool info: %v", err)
	}
	return layout, nil
}

func parseAmmConfig(data []byte) (uint32, error) {
	// 解析池数据获取价格
	var ammConfig raydium.AmmConfig
	err := ammConfig.Decode(data)
	if err != nil {
		return 0, fmt.Errorf("failed to decode amm config: %w", err)
	}
	return ammConfig.TradeFeeRate, nil
}
