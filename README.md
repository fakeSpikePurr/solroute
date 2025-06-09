# SolRoute SDK

SolRoute is a Go SDK that serves as the fundamental infrastructure for building DEX routing services on Solana. It provides essential building blocks for implementing cross-DEX routing solutions, enabling efficient token swaps across multiple liquidity pools.

## Supported Protocols

| Protocol | Program ID | Description |
|----------|------------|-------------|
| Raydium CPMM V4 | `675kPX9MHTjS2zt1qfr1NYHuzeLXfQM9H24wFSUt1Mp8` | Standard Raydium CPMM V4 pools |
| Raydium CPMM | `CPMMoo8L3F4NbTegBCKVNunggL7H1ZpdTHKxQB5qKP1C` | Raydium CPMM pools optimized for straightforward pool |
| Raydium CLMM | `CAMMCzo5YL8w4VFF8KVHrK22GGUsp5VTaW7grrKgrWqK` | Raydium Concentrated Liquidity pools |
| PumpSwap AMM | `pAMMBay6oceH9fJKBRHGP5D4bD4sWpmSwMn52FMfXEA` | PumpSwap AMM pools |

## Features

- **Pool Management**
  - Protocol-specific pool implementations
  - Pool information retrieval (ID, token pairs, reserves)
  - Quote calculation and swap instruction generation

- **Protocol Integration**
  - Unified interface for different DEX protocols
  - Pool discovery by token pair or pool ID
  - Protocol-specific implementations for Raydium and PumpSwap

- **Routing Engine**
  - Best price discovery across multiple pools
  - Optimal route selection for token swaps
  - Cross-protocol liquidity aggregation

## Installation

```bash
go get github.com/yimingWOW/solroute
```

## Quick Start

```go
// Initialize Solana client
solClient, err := sol.NewClient(ctx, "YOUR_RPC_ENDPOINT")
if err != nil {
    log.Fatal(err)
}
defer solClient.Close()

// Create router with supported protocols
router := router.NewSimpleRouter(
    protocol.NewPumpAmm(solClient),
    protocol.NewRaydiumAmm(solClient),
    protocol.NewRaydiumClmm(solClient),
    protocol.NewRaydiumCpmm(solClient),
)

// Query available pools for a token pair
pools, err := router.QueryAllPools(ctx, "TOKEN0_MINT", "TOKEN1_MINT")
if err != nil {
    log.Fatal(err)
}

// Find best pool for the swap
amountIn := math.NewInt(1000000000) // 1 token (9 decimals)
bestPool, amountOut, err := router.GetBestPool(ctx, solClient.RpcClient, 
    "TOKEN0_MINT", "TOKEN1_MINT", amountIn)
if err != nil {
    log.Fatal(err)
}

// Calculate minimum output amount with slippage
slippageBps := 100 // 1% slippage
minAmountOut := amountOut.Mul(math.NewInt(10000 - slippageBps)).Quo(math.NewInt(10000))

// Build and send swap transaction
instructions, err := bestPool.BuildSwapInstructions(ctx, solClient.RpcClient,
    userPublicKey, "TOKEN0_MINT", amountIn, minAmountOut)
if err != nil {
    log.Fatal(err)
}

// Send transaction
sig, err := solClient.SendTx(ctx, blockhash, signers, instructions, true)
```

## Project Structure

```
solroute/
├── pkg/
│   ├── api/         # Core interfaces (Pool and Protocol)
│   ├── pool/        # Protocol-specific pool implementations
│   ├── protocol/    # DEX protocol implementations
│   ├── router/      # Routing engine for best price discovery
│   └── sol/         # Solana client and utilities
```

### Core Components

- **Pool Interface**: Defines the standard interface for all pool implementations
  - `GetID()`: Retrieve pool identifier
  - `GetTokens()`: Get base and quote token mints
  - `GetQuote()`: Calculate swap output amount
  - `BuildSwapInstructions()`: Generate swap transaction instructions

- **Protocol Interface**: Defines the standard interface for DEX protocols
  - `FetchPoolsByPair()`: Find pools for a token pair
  - `FetchPoolByID()`: Retrieve specific pool by ID

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
