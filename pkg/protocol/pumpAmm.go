package protocol

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/yimingwow/solroute/pkg"
	"github.com/yimingwow/solroute/pkg/pump"
	"github.com/yimingwow/solroute/pkg/sol"
)

type PumpAmmProtocol struct {
	SolClient *sol.Client
}

func NewPumpAmm(solClient *sol.Client) *PumpAmmProtocol {
	return &PumpAmmProtocol{
		SolClient: solClient,
	}
}

func (p *PumpAmmProtocol) FetchPoolsByPair(ctx context.Context, baseMint string, quoteMint string) ([]pkg.Pool, error) {
	programAccounts := rpc.GetProgramAccountsResult{}
	data, err := p.getPumpAMMPoolAccountsByTokenPair(ctx, baseMint, quoteMint)
	if err != nil {
		log.Printf("GetPoolKeys programAccounts err: %v\n", err)
		return nil, err
	}
	programAccounts = append(programAccounts, data...)
	data, err = p.getPumpAMMPoolAccountsByTokenPair(ctx, quoteMint, baseMint)
	if err != nil {
		log.Printf("GetPoolKeys programAccounts err: %v\n", err)
		return nil, err
	}
	programAccounts = append(programAccounts, data...)

	res := make([]pkg.Pool, 0)
	for _, v := range programAccounts {
		layout := pump.ParsePoolData(v.Account.Data.GetBinary())
		if layout == nil {
			continue
		}
		layout.PoolId = v.Pubkey
		res = append(res, layout)
	}
	return res, nil
}

func (p *PumpAmmProtocol) getPumpAMMPoolAccountsByTokenPair(ctx context.Context, baseMint string, quoteMint string) (rpc.GetProgramAccountsResult, error) {
	var layout pump.PumpAMMPool
	return p.SolClient.RpcClient.GetProgramAccountsWithOpts(ctx, pump.PumpSwapProgramID, &rpc.GetProgramAccountsOpts{
		Filters: []rpc.RPCFilter{
			{
				DataSize: layout.Span(),
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: layout.Offset("BaseMint"),
					Bytes:  solana.MustPublicKeyFromBase58(baseMint).Bytes(),
				},
			},
			{
				Memcmp: &rpc.RPCFilterMemcmp{
					Offset: layout.Offset("QuoteMint"),
					Bytes:  solana.MustPublicKeyFromBase58(quoteMint).Bytes(),
				},
			},
		},
	})
}

func (p *PumpAmmProtocol) FetchPoolByID(ctx context.Context, poolId string) (pkg.Pool, error) {
	account, err := p.SolClient.RpcClient.GetAccountInfo(ctx, solana.MustPublicKeyFromBase58(poolId))
	if err != nil {
		log.Printf("GetAMMPool pool.ID: %s, err: %v\n", poolId, err)
		return nil, fmt.Errorf("failed to get pool account: %v", err)
	}

	layout := pump.ParsePoolData(account.Value.Data.GetBinary())
	if layout == nil {
		return nil, fmt.Errorf("failed to parse pool data")
	}
	layout.PoolId = solana.MustPublicKeyFromBase58(poolId)
	return layout, nil
}
