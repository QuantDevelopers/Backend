// Package options provides financial domain calculations for Tehran Stock Exchange options.
//
// Card: E9-1 · Options Primer
// Provides programmatic calculations for:
//   - Payoff at expiration (Long/Short Call/Put)
//   - Break-even prices
//   - Intrinsic and Time Value breakdown
//   - Moneyness classification (ITM, ATM, OTM)
//   - Effective leverage ratio
package options

import (
	"fmt"
	"math"
	"strings"
)

// DefaultATMThresholdPct defines the default +/-1% moneyness band around strike for ATM classification.
const DefaultATMThresholdPct = 0.01

// ContractType identifies whether an option is a Call or Put contract.
type ContractType string

const (
	ContractTypeCall ContractType = "CALL"
	ContractTypePut  ContractType = "PUT"
)

// ParseContractType normalizes and validates a contract type string.
func ParseContractType(raw string) (ContractType, error) {
	switch ContractType(strings.ToUpper(strings.TrimSpace(raw))) {
	case ContractTypeCall:
		return ContractTypeCall, nil
	case ContractTypePut:
		return ContractTypePut, nil
	default:
		return "", fmt.Errorf("invalid contract type %q: must be CALL or PUT", raw)
	}
}

// PositionType identifies whether the trader holds a Long (buyer) or Short (writer) position.
type PositionType string

const (
	PositionTypeLong  PositionType = "LONG"
	PositionTypeShort PositionType = "SHORT"
)

// ParsePositionType normalizes and validates a position type string.
func ParsePositionType(raw string) (PositionType, error) {
	switch PositionType(strings.ToUpper(strings.TrimSpace(raw))) {
	case PositionTypeLong:
		return PositionTypeLong, nil
	case PositionTypeShort:
		return PositionTypeShort, nil
	default:
		return "", fmt.Errorf("invalid position type %q: must be LONG or SHORT", raw)
	}
}

// Moneyness classifies an option's intrinsic profitability relative to spot price.
type Moneyness string

const (
	MoneynessITM Moneyness = "ITM"
	MoneynessATM Moneyness = "ATM"
	MoneynessOTM Moneyness = "OTM"
)

// OptionValueBreakdown splits option premium into intrinsic and time value alongside moneyness and break-even.
type OptionValueBreakdown struct {
	Premium        float64
	IntrinsicValue float64
	TimeValue      float64
	Moneyness      Moneyness
	BreakevenPrice float64
}

// CalculateBreakeven computes the underlying spot price at expiration where net profit is zero.
//
// Call Breakeven = Strike + Premium
// Put Breakeven  = Strike - Premium
func CalculateBreakeven(contractType ContractType, strike, premium float64) (float64, error) {
	cType, err := ParseContractType(string(contractType))
	if err != nil {
		return 0, err
	}
	if cType == ContractTypeCall {
		return strike + premium, nil
	}
	return strike - premium, nil
}

// CalculateIntrinsicValue computes the immediate exercise value of an option.
//
// Call Intrinsic Value = max(0, Spot - Strike)
// Put Intrinsic Value  = max(0, Strike - Spot)
func CalculateIntrinsicValue(contractType ContractType, strike, spotPrice float64) (float64, error) {
	cType, err := ParseContractType(string(contractType))
	if err != nil {
		return 0, err
	}
	if cType == ContractTypeCall {
		return math.Max(0.0, spotPrice-strike), nil
	}
	return math.Max(0.0, strike-spotPrice), nil
}

// ClassifyMoneyness classifies an option as ITM, ATM, or OTM using atmThresholdPct around strike.
func ClassifyMoneyness(contractType ContractType, strike, spotPrice, atmThresholdPct float64) (Moneyness, error) {
	cType, err := ParseContractType(string(contractType))
	if err != nil {
		return "", err
	}
	if strike <= 0 {
		return "", fmt.Errorf("strike price must be positive, got %v", strike)
	}

	relDiff := math.Abs(spotPrice-strike) / strike
	if relDiff <= atmThresholdPct {
		return MoneynessATM, nil
	}

	if cType == ContractTypeCall {
		if spotPrice > strike {
			return MoneynessITM, nil
		}
		return MoneynessOTM, nil
	}

	if spotPrice < strike {
		return MoneynessITM, nil
	}
	return MoneynessOTM, nil
}

// AnalyzeOptionValue decomposes an option's market premium into intrinsic and time value.
func AnalyzeOptionValue(contractType ContractType, strike, premium, spotPrice float64) (OptionValueBreakdown, error) {
	intrinsic, err := CalculateIntrinsicValue(contractType, strike, spotPrice)
	if err != nil {
		return OptionValueBreakdown{}, err
	}
	moneyness, err := ClassifyMoneyness(contractType, strike, spotPrice, DefaultATMThresholdPct)
	if err != nil {
		return OptionValueBreakdown{}, err
	}
	breakeven, err := CalculateBreakeven(contractType, strike, premium)
	if err != nil {
		return OptionValueBreakdown{}, err
	}

	return OptionValueBreakdown{
		Premium:        premium,
		IntrinsicValue: intrinsic,
		TimeValue:      math.Max(0.0, premium-intrinsic),
		Moneyness:      moneyness,
		BreakevenPrice: breakeven,
	}, nil
}

// CalculatePayoff computes net profit or loss at expiration scaled by contractSize.
func CalculatePayoff(
	contractType ContractType,
	strike, premium, spotAtExpiry float64,
	position PositionType,
	contractSize int,
) (float64, error) {
	cType, err := ParseContractType(string(contractType))
	if err != nil {
		return 0, err
	}
	pos, err := ParsePositionType(string(position))
	if err != nil {
		return 0, err
	}

	var grossPayoff float64
	if cType == ContractTypeCall {
		grossPayoff = math.Max(0.0, spotAtExpiry-strike)
	} else {
		grossPayoff = math.Max(0.0, strike-spotAtExpiry)
	}

	var netPerShare float64
	if pos == PositionTypeLong {
		netPerShare = grossPayoff - premium
	} else {
		netPerShare = premium - grossPayoff
	}

	return netPerShare * float64(contractSize), nil
}

// CalculateEffectiveLeverage computes Omega = |Delta| * (Spot / Premium).
func CalculateEffectiveLeverage(spotPrice, premium, delta float64) float64 {
	if premium <= 0 {
		return 0.0
	}
	return math.Abs(delta) * (spotPrice / premium)
}
