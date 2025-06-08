package raydium

import (
	"bytes"
	"fmt"

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
)

type AmmConfig struct {
	Bump            uint8
	Index           uint16
	Owner           solana.PublicKey
	ProtocolFeeRate uint32
	TradeFeeRate    uint32
	TickSpacing     uint16
	FundFeeRate     uint32
	PaddingU32      uint32
	FundOwner       solana.PublicKey
	Padding         [3]uint64
}

func (l *AmmConfig) Decode(data []byte) error {
	// Skip 8 bytes discriminator if present
	if len(data) > 8 {
		data = data[8:]
	}

	dec := bin.NewBinDecoder(data)
	return dec.Decode(l)
}

type SwapV4Instruction struct {
	Instruction      uint8
	AmountIn         uint64
	MinimumOutAmount uint64
}

func (inst *SwapV4Instruction) Data() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := bin.NewBorshEncoder(buf).Encode(inst); err != nil {
		return nil, fmt.Errorf("unable to encode instruction: %w", err)
	}
	return buf.Bytes(), nil
}
