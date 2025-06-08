package pump

import (
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/yimingwow/solroute/pkg"
)

type PumpAMMPool struct {
	Discriminator         [8]uint8 `bin:"skip"`
	PoolBump              uint8    // 第9个字节，应该是252 (fc)
	Index                 uint16
	Creator               solana.PublicKey // 32字节
	BaseMint              solana.PublicKey // 32字节
	QuoteMint             solana.PublicKey // 32字节
	LpMint                solana.PublicKey // 32字节
	PoolBaseTokenAccount  solana.PublicKey // 32字节
	PoolQuoteTokenAccount solana.PublicKey // 32字节
	LpSupply              uint64           // 8字节
	CoinCreator           solana.PublicKey // 32字节

	PoolId           solana.PublicKey
	BaseAmount       math.Int
	QuoteAmount      math.Int
	UserBaseAccount  solana.PublicKey
	UserQuoteAccount solana.PublicKey
}

func (l *PumpAMMPool) Span() uint64 {
	return uint64(300)
}

func (l *PumpAMMPool) Offset(value string) uint64 {
	switch value {
	case "BaseMint":
		return 43
	case "QuoteMint":
		return 43 + 32
	default:
		return 0
	}
}

func (l *PumpAMMPool) Decode(data []byte) error {
	if len(data) < 211 {
		return fmt.Errorf("data too short: expected 211 bytes, got %d", len(data))
	}
	dec := bin.NewBinDecoder(data)
	return dec.Decode(l)
}

func parsePoolData(data []byte) *PumpAMMPool {
	layout := &PumpAMMPool{}
	// 解析结构
	discriminator := [8]byte{}
	copy(discriminator[:], data[:8])
	layout.PoolBump = uint8(data[8])
	layout.Index = binary.LittleEndian.Uint16(data[9:11])

	offset := 11
	layout.Creator = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.BaseMint = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.QuoteMint = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.LpMint = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.PoolBaseTokenAccount = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.PoolQuoteTokenAccount = solana.PublicKeyFromBytes(data[offset : offset+32])
	offset += 32
	layout.LpSupply = binary.LittleEndian.Uint64(data[offset : offset+8])
	offset += 8
	if len(data[offset:]) > 32 {
		layout.CoinCreator = solana.PublicKeyFromBytes(data[offset : offset+32])
	} else {
		layout.CoinCreator = solana.MustPublicKeyFromBase58("11111111111111111111111111111111")
	}

	return layout
}

func (l *PumpAMMPool) GetID() string {
	return l.PoolId.String()
}

func (l *PumpAMMPool) GetTokens() (string, string) {
	return l.BaseMint.String(), l.QuoteMint.String()
}

func (l *PumpAMMPool) GetType() pkg.PoolType {
	return pkg.PoolTypePump
}
