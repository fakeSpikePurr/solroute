package meteora

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	cosmosmath "cosmossdk.io/math"
	"github.com/gagliardetto/solana-go"
	"github.com/yimingwow/solroute/pkg/sol"
	"lukechampine.com/uint128"
)

// Quote calculates the output amount for a given input amount and token
func (pool *MeteoraDlmmPool) Quote(ctx context.Context, solClient *sol.Client, inputMint string, inputAmount cosmosmath.Int) (cosmosmath.Int, error) {
	pool.orgActiveId = pool.activeId
	totalAmountOut := cosmosmath.ZeroInt()

	if err := pool.validateSwapActivation(); err != nil {
		return cosmosmath.ZeroInt(), fmt.Errorf("swap activation validation failed: %w", err)
	}
	pool.UpdateReferences()

	amountLeft := inputAmount
	swapForY := inputMint == pool.TokenXMint.String()

	// Process active bin arrays
	for amountLeft.IsPositive() {
		// Get the current active bin array
		activeBinArray, err := pool.getCurrentActiveBinArray(swapForY)
		if err != nil {
			return cosmosmath.ZeroInt(), err
		}

		// Process active bins
		for {
			withinRange, err := activeBinArray.IsBinIDWithinRange(pool.activeId)
			if err != nil {
				return cosmosmath.ZeroInt(), fmt.Errorf("failed to check bin ID range: %w", err)
			}
			if !withinRange || inputAmount.IsZero() {
				if err := pool.AdvanceActiveBin(swapForY); err != nil {
					return cosmosmath.ZeroInt(), fmt.Errorf("failed to advance active bin: %w", err)
				}
				break
			} else {
				// Update volatility accumulator
				if err := pool.UpdateVolatilityAccumulator(); err != nil {
					return cosmosmath.ZeroInt(), fmt.Errorf("failed to update volatility accumulator: %w", err)
				}

				activeBin, err := activeBinArray.GetBinMut(pool.activeId)
				if err != nil {
					return cosmosmath.ZeroInt(), fmt.Errorf("failed to get active bin: %w", err)
				}

				if !activeBin.IsEmpty(!swapForY) {
					swapResult, err := pool.Swap(
						activeBin,
						amountLeft.Uint64(),
						swapForY,
					)
					if err != nil {
						return cosmosmath.ZeroInt(), fmt.Errorf("swap failed: %w", err)
					}
					amountLeft = amountLeft.Sub(cosmosmath.NewInt(int64(swapResult.amountInWithFees)))
					totalAmountOut = totalAmountOut.Add(cosmosmath.NewInt(int64(swapResult.amountOut)))
				}
				if err := pool.AdvanceActiveBin(swapForY); err != nil {
					return cosmosmath.ZeroInt(), fmt.Errorf("failed to advance active bin: %w", err)
				}
			}
		}
	}

	pool.activeId = pool.orgActiveId
	return totalAmountOut, nil
}

// validateSwapActivation checks if the swap is allowed based on pair status and activation conditions
func (pool *MeteoraDlmmPool) validateSwapActivation() error {
	currentTimestamp := uint64(time.Now().Unix())
	currentSlot := uint64(pool.Clock.Slot)

	// Check pair status
	if pool.status != uint8(PairStatusEnabled) {
		return errors.New("pair is disabled")
	}

	// For permissioned pairs, check activation time
	if pool.pairType == uint8(PairTypePermission) {
		var currentPoint uint64
		switch pool.activationType {
		case uint8(ActivationTypeSlot):
			currentPoint = currentSlot
		case uint8(ActivationTypeTimestamp):
			currentPoint = currentTimestamp
		default:
			return errors.New("invalid activation type")
		}
		// Check if activation point has been reached
		if currentPoint < pool.activationPoint {
			return errors.New("pair is not yet activated")
		}
	}
	return nil
}

// UpdateReferences updates the volatility reference parameters based on elapsed time
func (pool *MeteoraDlmmPool) UpdateReferences() {
	elapsed := int64(pool.Clock.UnixTimestamp) - pool.vParameters.lastUpdateTimestamp
	if elapsed >= int64(pool.parameters.filterPeriod) {
		pool.vParameters.indexReference = pool.activeId
		if elapsed < int64(pool.parameters.decayPeriod) {
			// Note: JS SDK and Rust SDK have different implementations
			// JS uses multiplication, Rust uses subtraction
			volatilityAccumulator := pool.vParameters.volatilityAccumulator * uint32(pool.parameters.reductionFactor)
			volatilityReference := volatilityAccumulator / BasisPointMax

			pool.vParameters.volatilityReference = volatilityReference
		} else {
			pool.vParameters.volatilityReference = 0
		}
	}
}

// SwapResult represents the result of a swap operation
type SwapResult struct {
	// Amount of token swapped into the bin (including fees)
	amountInWithFees uint64
	// Amount of token swapped out from the bin
	amountOut uint64
	// Swap fee, includes protocol fee
	fee uint64
	// Protocol fee portion
	protocolFee uint64
	// Indicates whether we reached exact out amount
	isExactOutAmount bool
}

// Swap performs a swap operation on a specific bin
func (pool *MeteoraDlmmPool) Swap(bin *Bin, amountIn uint64, swapForY bool) (*SwapResult, error) {
	price, err := bin.GetOrStoreBinPrice(pool.activeId, pool.binStep)
	if err != nil {
		return nil, fmt.Errorf("failed to get bin price: %w", err)
	}

	maxAmountOut := bin.GetMaxAmountOut(swapForY)
	maxAmountIn, err := bin.GetMaxAmountIn(price, swapForY)
	if err != nil {
		return nil, fmt.Errorf("failed to get max amount in: %w", err)
	}
	maxFee, err := pool.ComputeFee(maxAmountIn.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to compute max fee: %w", err)
	}
	maxAmountIn = maxAmountIn.Add(maxAmountIn, big.NewInt(int64(maxFee))) // Go automatically checks overflow

	var (
		amountInWithFees uint64
		amountOut        uint64
		fee              uint64
		protocolFee      uint64
	)

	// Determine actual swap amount and fees
	if amountIn > maxAmountIn.Uint64() {
		amountInWithFees = maxAmountIn.Uint64()
		amountOut = maxAmountOut
		fee = maxFee
		protocolFee, err = pool.ComputeProtocolFee(maxFee)
		if err != nil {
			return nil, fmt.Errorf("failed to compute protocol fee: %w", err)
		}
	} else {
		fee, err = pool.ComputeFeeFromAmount(amountIn)
		if err != nil {
			return nil, fmt.Errorf("failed to compute fee from amount: %w", err)
		}
		amountInAfterFee := amountIn - fee
		amountOutTemp, err := bin.GetAmountOut(amountInAfterFee, price, swapForY)
		if err != nil {
			return nil, fmt.Errorf("failed to get amount out: %w", err)
		}

		amountOut = min(amountOutTemp.Uint64(), maxAmountOut)
		amountInWithFees = amountIn

		protocolFee, err = pool.ComputeProtocolFee(fee)
		if err != nil {
			return nil, fmt.Errorf("failed to compute protocol fee: %w", err)
		}
	}

	amountIntoBin := amountInWithFees - fee

	// Update bin amounts
	if swapForY {
		bin.amountX += amountIntoBin
		if bin.amountY < amountOut {
			return nil, fmt.Errorf("insufficient Y amount")
		}
		bin.amountY -= amountOut
	} else {
		bin.amountY += amountIntoBin
		if bin.amountX < amountOut {
			return nil, fmt.Errorf("insufficient X amount")
		}
		bin.amountX -= amountOut
	}

	return &SwapResult{
		amountInWithFees: amountInWithFees,
		amountOut:        amountOut,
		fee:              fee,
		protocolFee:      protocolFee,
		isExactOutAmount: false,
	}, nil
}

// NextBinArrayIndexWithLiquidityInternal finds the next bin array index with liquidity using internal bitmap
func (pool *MeteoraDlmmPool) NextBinArrayIndexWithLiquidityInternal(swapForY bool, startArrayIndex int32) (int32, bool, error) {
	// Convert binArrayBitmap to big integer type (using math/big package)
	binArrayBitmap := FromLimbs(pool.binArrayBitmap[:])
	arrayOffset := GetBinArrayOffset(startArrayIndex)
	bitmapDetail := BitmapTypeDetail(U1024)
	minBitmapID, maxBitmapID := BitmapRange()

	if swapForY {
		bitmapRange := uint(maxBitmapID - minBitmapID)
		offsetBitMap := binArrayBitmap.Lsh(binArrayBitmap, bitmapRange-uint(arrayOffset))
		mostSignificantBit := MostSignificantBit(offsetBitMap, bitmapDetail.Bits)
		if mostSignificantBit < 0 {
			return minBitmapID - 1, false, nil
		} else {
			return startArrayIndex - int32(mostSignificantBit), true, nil
		}
	} else {
		offsetBitMap := binArrayBitmap.Rsh(binArrayBitmap, uint(arrayOffset))
		lsb := LeastSignificantBit(offsetBitMap, bitmapDetail.Bits)

		if lsb < 0 {
			return maxBitmapID + 1, false, nil
		} else {
			return startArrayIndex + int32(lsb), true, nil
		}
	}
}

// UpdateVolatilityAccumulator updates the volatility accumulator based on index changes
func (pool *MeteoraDlmmPool) UpdateVolatilityAccumulator() error {
	// Calculate delta_id (absolute difference of indices)
	deltaID := int64(pool.vParameters.indexReference) - int64(pool.activeId)

	// Take absolute value
	if deltaID < 0 {
		deltaID = -deltaID
	}

	// Calculate deltaID * BASIS_POINT_MAX
	deltaIdWithBasisPoint := deltaID * int64(BasisPointMax)

	// Calculate volatility_accumulator
	volatilityAccumulator := uint64(pool.vParameters.volatilityReference) + uint64(deltaIdWithBasisPoint)

	// Take the smaller value
	minValue := uint64(math.Min(
		float64(volatilityAccumulator),
		float64(pool.parameters.maxVolatilityAccumulator),
	))

	// Update accumulator value
	pool.vParameters.volatilityAccumulator = uint32(minValue)

	return nil
}

// ComputeProtocolFee calculates the protocol fee from the total fee amount
func (pool *MeteoraDlmmPool) ComputeProtocolFee(feeAmount uint64) (uint64, error) {
	// Convert feeAmount to uint128
	feeAmountBig := uint128.From64(feeAmount)

	// Convert protocol_share to uint128
	protocolShare := uint128.From64(uint64(pool.parameters.protocolShare))

	// Calculate feeAmount * protocol_share
	protocolFee := feeAmountBig.Mul(protocolShare)

	// Divide by BASIS_POINT_MAX
	basisPointMax := uint128.From64(BasisPointMax)
	if basisPointMax.IsZero() {
		return 0, fmt.Errorf("division by zero")
	}
	protocolFee = protocolFee.Div(basisPointMax)

	// Check if result can be safely converted to uint64
	if protocolFee.Hi != 0 {
		return 0, fmt.Errorf("protocol fee exceeds uint64 range")
	}

	return protocolFee.Lo, nil
}

// ComputeFeeFromAmount calculates the fee from an amount including fees
func (pool *MeteoraDlmmPool) ComputeFeeFromAmount(amountWithFees uint64) (uint64, error) {
	// Get total fee rate
	totalFeeRate, err := pool.GetTotalFee()
	if err != nil {
		return 0, fmt.Errorf("failed to get total fee: %w", err)
	}

	// Convert to big.Int for calculation
	amount := new(big.Int).SetUint64(amountWithFees)
	feeRate := totalFeeRate

	// Calculate amount * totalFeeRate
	feeAmount := new(big.Int).Mul(amount, feeRate)

	// Add FEE_PRECISION - 1
	feeAmount = feeAmount.Add(feeAmount, big.NewInt(FeePrecision-1))

	// Divide by FEE_PRECISION
	feeAmount = feeAmount.Div(feeAmount, big.NewInt(FeePrecision))

	return feeAmount.Uint64(), nil
}

// GetTotalFee calculates the total fee rate by combining base and variable fees
func (pool *MeteoraDlmmPool) GetTotalFee() (*big.Int, error) {
	// Get base fee
	baseFee, err := pool.GetBaseFee()
	if err != nil {
		return big.NewInt(0), fmt.Errorf("failed to get base fee: %w", err)
	}

	// Get variable fee
	variableFee, err := pool.GetVariableFee()
	if err != nil {
		return big.NewInt(0), fmt.Errorf("failed to get variable fee: %w", err)
	}
	// Calculate total fee rate
	totalFeeRate := baseFee.Add(baseFee, variableFee)

	// Compare with max fee rate, take the smaller value
	maxFeeRate := big.NewInt(MaxFeeRate)
	if totalFeeRate.Cmp(maxFeeRate) > 0 {
		totalFeeRate = maxFeeRate
	}

	return totalFeeRate, nil
}

// GetBaseFee calculates the base fee based on pool parameters
func (pool *MeteoraDlmmPool) GetBaseFee() (*big.Int, error) {
	// Create big.Int for calculation
	result := new(big.Int).SetUint64(uint64(pool.parameters.baseFactor))

	// Multiply by bin_step
	result.Mul(result, new(big.Int).SetUint64(uint64(pool.binStep)))

	// Multiply by 10
	result.Mul(result, big.NewInt(10))

	// Calculate 10^base_fee_power_factor
	powerOf10 := new(big.Int).Exp(
		big.NewInt(10),
		new(big.Int).SetUint64(uint64(pool.parameters.baseFeePowerFactor)),
		nil,
	)

	// Final multiplication
	result.Mul(result, powerOf10)

	// Check if result exceeds uint128 range
	if result.BitLen() > 128 {
		return big.NewInt(0), fmt.Errorf("result exceeds uint128 range")
	}
	return result, nil
}

// GetVariableFee gets the variable fee based on current volatility accumulator
func (pool *MeteoraDlmmPool) GetVariableFee() (*big.Int, error) {
	return pool.ComputeVariableFee(pool.vParameters.volatilityAccumulator)
}

// ComputeVariableFee calculates the variable fee based on volatility accumulator
func (pool *MeteoraDlmmPool) ComputeVariableFee(volatilityAccumulator uint32) (*big.Int, error) {
	// If variable fee control is 0, return 0 directly
	if pool.parameters.variableFeeControl == 0 {
		return big.NewInt(0), nil
	}

	// Convert to uint128
	volatilityAccumulatorBig := cosmosmath.NewInt(int64(volatilityAccumulator))
	binStep := cosmosmath.NewInt(int64(pool.binStep))
	variableFeeControl := cosmosmath.NewInt(int64(pool.parameters.variableFeeControl))

	// Calculate (volatility_accumulator * bin_step)^2
	squareVfaBin := volatilityAccumulatorBig.Mul(binStep)
	if squareVfaBin.IsZero() && volatilityAccumulatorBig.IsZero() && binStep.IsZero() {
		return big.NewInt(0), fmt.Errorf("multiplication overflow")
	}

	squareVfaBin = squareVfaBin.Mul(squareVfaBin)
	vFee := variableFeeControl.Mul(squareVfaBin)
	scaledVFee := vFee.Add(cosmosmath.NewInt(99_999_999_999))
	divisor := cosmosmath.NewInt(100_000_000_000)
	scaledVFee = scaledVFee.Quo(divisor)

	return scaledVFee.BigInt(), nil
}

// AdvanceActiveBin advances the active bin ID based on swap direction
func (pool *MeteoraDlmmPool) AdvanceActiveBin(swapForY bool) error {
	var nextActiveBinID int32

	// Calculate next bin ID based on swap direction
	if swapForY {
		// Check for subtraction overflow
		if pool.activeId == math.MinInt32 {
			return fmt.Errorf("bin id underflow")
		}
		nextActiveBinID = pool.activeId - 1
	} else {
		// Check for addition overflow
		if pool.activeId == math.MaxInt32 {
			return fmt.Errorf("bin id overflow")
		}
		nextActiveBinID = pool.activeId + 1
	}

	// Check if new bin ID is within valid range
	if nextActiveBinID < MinBinID || nextActiveBinID > MaxBinID {
		return fmt.Errorf("insufficient liquidity: bin id %d out of range [%d, %d]",
			nextActiveBinID, MinBinID, MaxBinID)
	}

	// Update active bin ID
	pool.activeId = nextActiveBinID

	return nil
}

// GetBinArrayPubkeysForSwap retrieves bin array public keys needed for swap operations
func (pool *MeteoraDlmmPool) GetBinArrayPubkeysForSwap(swapForY bool, takeCount uint8) ([]solana.PublicKey, error) {
	binArrayPubkeys := make([]solana.PublicKey, 0)

	startBinArrayIdx := BinIDToBinArrayIndex(pool.activeId)

	var binArrayIdx []int64
	increment := int64(1)
	if swapForY {
		increment = -1
	}
	for i := 0; i < int(takeCount); i++ {
		if IsOverflowDefaultBinArrayBitmap(int32(startBinArrayIdx)) {
			if pool.bitmapExtension == nil {
				break
			}
			nextBinArrayIdx, hasLiquidity, err := pool.bitmapExtension.NextBinArrayIndexWithLiquidity(swapForY, int32(startBinArrayIdx))
			if err != nil {
				break
			}
			if hasLiquidity {
				binArrayIdx = append(binArrayIdx, int64(nextBinArrayIdx))
				pda, _ := DeriveBinArrayPDA(pool.PoolId, int64(nextBinArrayIdx))
				binArrayPubkeys = append(binArrayPubkeys, pda)
				startBinArrayIdx = int64(nextBinArrayIdx) + increment
			} else {
				// Switch to internal bitmap
				startBinArrayIdx = int64(nextBinArrayIdx)
			}
		} else {
			nextBinArrayIdx, hasLiquidity, err := pool.NextBinArrayIndexWithLiquidityInternal(swapForY, int32(startBinArrayIdx))
			if err != nil {
				break
			}
			if hasLiquidity {
				binArrayIdx = append(binArrayIdx, int64(nextBinArrayIdx))
				pda, _ := DeriveBinArrayPDA(pool.PoolId, int64(nextBinArrayIdx))
				binArrayPubkeys = append(binArrayPubkeys, pda)
				startBinArrayIdx = int64(nextBinArrayIdx) + increment
			} else {
				// Switch to external bitmap
				startBinArrayIdx = int64(nextBinArrayIdx)
			}
		}
	}

	return binArrayPubkeys, nil
}

// getCurrentActiveBinArray retrieves the current active bin array for swap operations
func (pool *MeteoraDlmmPool) getCurrentActiveBinArray(swapForY bool) (BinArray, error) {
	startBinArrayIdx := BinIDToBinArrayIndex(pool.activeId)

	var binArrayIdx int64
	increment := int64(1)
	if swapForY {
		increment = -1
	}

	if IsOverflowDefaultBinArrayBitmap(int32(startBinArrayIdx)) {
		if pool.bitmapExtension == nil {
			return BinArray{}, errors.New("bitmapExtension is nil")
		}
		nextBinArrayIdx, hasLiquidity, err := pool.bitmapExtension.NextBinArrayIndexWithLiquidity(swapForY, int32(startBinArrayIdx))
		if err != nil {
			return BinArray{}, err
		}
		if hasLiquidity {
			binArrayIdx = int64(nextBinArrayIdx)
			startBinArrayIdx = int64(nextBinArrayIdx) + increment
		} else {
			// Switch to internal bitmap
			startBinArrayIdx = int64(nextBinArrayIdx)
		}
	} else {
		nextBinArrayIdx, hasLiquidity, err := pool.NextBinArrayIndexWithLiquidityInternal(swapForY, int32(startBinArrayIdx))
		if err != nil {
			return BinArray{}, err
		}
		if hasLiquidity {
			binArrayIdx = int64(nextBinArrayIdx)
			startBinArrayIdx = int64(nextBinArrayIdx) + increment
		} else {
			// Switch to external bitmap
			startBinArrayIdx = int64(nextBinArrayIdx)
		}
	}

	// Generate PDA address for bin array
	pda, _ := DeriveBinArrayPDA(pool.PoolId, binArrayIdx)

	binArray, exists := pool.BinArrays[pda.String()]
	if !exists {
		return BinArray{}, errors.New("active bin array not found")
	}
	return binArray, nil
}
