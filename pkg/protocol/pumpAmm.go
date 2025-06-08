package protocol

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/yimingwow/solroute/pkg"
)

func (p *pkg.PumpAMMPool) GetPumpAmmPoolByTokenPair(ctx context.Context, baseMint string, quoteMint string) ([]pkg.Pool, error) {
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
		layout := parsePoolData(v.Account.Data.GetBinary())
		if layout == nil {
			log.Printf("parsePoolData err: %v\n", err)
			continue
		}
		layout.PoolId = v.Pubkey
		res = append(res, layout)
	}
	return res, nil
}

func (p *pkg.PumpAMMPool) getPumpAMMPoolAccountsByTokenPair(ctx context.Context, baseMint string, quoteMint string) (rpc.GetProgramAccountsResult, error) {
	_, err := p.GetPumpAmmPoolByPoolId(ctx, solana.MustPublicKeyFromBase58("EiDvJRs9nzkoiXDzE6gFsPAntEFvXmqzV7bT2QKbeog3"))
	if err != nil {
		log.Printf("GetPumpAmmPoolByPoolId err: %v\n", err)
		return nil, err
	}

	var layout PumpAMMPool
	return p.SolClient.RpcClient.GetProgramAccountsWithOpts(ctx, PumpSwapProgramID, &rpc.GetProgramAccountsOpts{
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

func (p *PumpAMMPool) GetPumpAmmPoolByPoolId(ctx context.Context, poolId solana.PublicKey) (*PumpAMMPool, error) {

	account, err := p.SolClient.RpcClient.GetAccountInfo(ctx, poolId)
	if err != nil {
		log.Printf("GetAMMPool pool.ID: %s, err: %v\n", poolId.String(), err)
		return nil, fmt.Errorf("failed to get pool account: %v", err)
	}

	layout := parsePoolData(account.Value.Data.GetBinary())
	if layout == nil {
		log.Printf("parsePoolData err: %v\n", err)
		return nil, err
	}
	layout.PoolId = poolId
	return layout, nil
}
