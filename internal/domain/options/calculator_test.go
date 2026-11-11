package options

import "testing"

func TestCallOptionCalculations(t *testing.T) {
	// Example: Khodro stock Spot = 3000, Strike = 2800, Premium = 350
	const spot = 3000.0
	const strike = 2800.0
	const premium = 350.0

	// 1. Breakeven = 2800 + 350 = 3150
	be, err := CalculateBreakeven("call", strike, premium)
	if err != nil {
		t.Fatalf("CalculateBreakeven failed: %v", err)
	}
	if be != 3150.0 {
		t.Fatalf("Breakeven = %v, want 3150.0", be)
	}

	// 2. Intrinsic Value = max(0, 3000 - 2800) = 200
	intrinsic, err := CalculateIntrinsicValue(ContractTypeCall, strike, spot)
	if err != nil {
		t.Fatalf("CalculateIntrinsicValue failed: %v", err)
	}
	if intrinsic != 200.0 {
		t.Fatalf("IntrinsicValue = %v, want 200.0", intrinsic)
	}

	// 3. Moneyness: spot > strike => ITM
	moneyness, err := ClassifyMoneyness(ContractTypeCall, strike, spot, DefaultATMThresholdPct)
	if err != nil {
		t.Fatalf("ClassifyMoneyness failed: %v", err)
	}
	if moneyness != MoneynessITM {
		t.Fatalf("Moneyness = %v, want ITM", moneyness)
	}

	// 4. Value Breakdown: Intrinsic = 200, Time Value = 350 - 200 = 150
	breakdown, err := AnalyzeOptionValue(ContractTypeCall, strike, premium, spot)
	if err != nil {
		t.Fatalf("AnalyzeOptionValue failed: %v", err)
	}
	if breakdown.IntrinsicValue != 200.0 || breakdown.TimeValue != 150.0 || breakdown.Moneyness != MoneynessITM || breakdown.BreakevenPrice != 3150.0 {
		t.Fatalf("unexpected breakdown: %+v", breakdown)
	}

	// 5. Payoff at Expiry (1000 contract size):
	payoffProfit, err := CalculatePayoff(ContractTypeCall, strike, premium, 3500.0, PositionTypeLong, 1000)
	if err != nil {
		t.Fatalf("CalculatePayoff profit failed: %v", err)
	}
	if payoffProfit != 350000.0 {
		t.Fatalf("Long Call profit = %v, want 350000.0", payoffProfit)
	}

	payoffLoss, err := CalculatePayoff(ContractTypeCall, strike, premium, 2500.0, PositionTypeLong, 1000)
	if err != nil {
		t.Fatalf("CalculatePayoff loss failed: %v", err)
	}
	if payoffLoss != -350000.0 {
		t.Fatalf("Long Call loss = %v, want -350000.0", payoffLoss)
	}

	shortGain, err := CalculatePayoff(ContractTypeCall, strike, premium, 2500.0, PositionTypeShort, 1000)
	if err != nil {
		t.Fatalf("CalculatePayoff short gain failed: %v", err)
	}
	if shortGain != 350000.0 {
		t.Fatalf("Short Call gain = %v, want 350000.0", shortGain)
	}
}

func TestPutOptionCalculations(t *testing.T) {
	// Example: Ahram Put Option Spot = 70000, Strike = 75000, Premium = 8000
	const spot = 70000.0
	const strike = 75000.0
	const premium = 8000.0

	be, err := CalculateBreakeven(ContractTypePut, strike, premium)
	if err != nil {
		t.Fatalf("CalculateBreakeven failed: %v", err)
	}
	if be != 67000.0 {
		t.Fatalf("Put Breakeven = %v, want 67000.0", be)
	}

	intrinsic, err := CalculateIntrinsicValue(ContractTypePut, strike, spot)
	if err != nil {
		t.Fatalf("CalculateIntrinsicValue failed: %v", err)
	}
	if intrinsic != 5000.0 {
		t.Fatalf("Put IntrinsicValue = %v, want 5000.0", intrinsic)
	}

	moneyness, err := ClassifyMoneyness(ContractTypePut, strike, spot, DefaultATMThresholdPct)
	if err != nil {
		t.Fatalf("ClassifyMoneyness failed: %v", err)
	}
	if moneyness != MoneynessITM {
		t.Fatalf("Put Moneyness = %v, want ITM", moneyness)
	}

	breakdown, err := AnalyzeOptionValue(ContractTypePut, strike, premium, spot)
	if err != nil {
		t.Fatalf("AnalyzeOptionValue failed: %v", err)
	}
	if breakdown.IntrinsicValue != 5000.0 || breakdown.TimeValue != 3000.0 {
		t.Fatalf("unexpected Put breakdown: %+v", breakdown)
	}

	profit, err := CalculatePayoff(ContractTypePut, strike, premium, 60000.0, PositionTypeLong, 1000)
	if err != nil {
		t.Fatalf("CalculatePayoff profit failed: %v", err)
	}
	if profit != 7000000.0 {
		t.Fatalf("Long Put profit = %v, want 7000000.0", profit)
	}

	loss, err := CalculatePayoff(ContractTypePut, strike, premium, 80000.0, PositionTypeLong, 1000)
	if err != nil {
		t.Fatalf("CalculatePayoff loss failed: %v", err)
	}
	if loss != -8000000.0 {
		t.Fatalf("Long Put loss = %v, want -8000000.0", loss)
	}
}

func TestOTMAndATMClassification(t *testing.T) {
	cases := []struct {
		cType ContractType
		spot  float64
		want  Moneyness
	}{
		{ContractTypeCall, 1005, MoneynessATM},
		{ContractTypeCall, 1200, MoneynessITM},
		{ContractTypeCall, 800, MoneynessOTM},
		{ContractTypePut, 995, MoneynessATM},
		{ContractTypePut, 800, MoneynessITM},
		{ContractTypePut, 1200, MoneynessOTM},
	}

	for _, tc := range cases {
		got, err := ClassifyMoneyness(tc.cType, 1000, tc.spot, DefaultATMThresholdPct)
		if err != nil {
			t.Fatalf("ClassifyMoneyness(%s, 1000, %v) failed: %v", tc.cType, tc.spot, err)
		}
		if got != tc.want {
			t.Fatalf("ClassifyMoneyness(%s, 1000, %v) = %s, want %s", tc.cType, tc.spot, got, tc.want)
		}
	}
}

func TestEffectiveLeverage(t *testing.T) {
	if got := CalculateEffectiveLeverage(70000.0, 7000.0, 1.0); got != 10.0 {
		t.Fatalf("CalculateEffectiveLeverage = %v, want 10.0", got)
	}
	if got := CalculateEffectiveLeverage(70000.0, 0.0, 1.0); got != 0.0 {
		t.Fatalf("CalculateEffectiveLeverage zero premium = %v, want 0.0", got)
	}
}

func TestValidationErrors(t *testing.T) {
	if _, err := CalculateBreakeven("INVALID", 1000, 100); err == nil {
		t.Fatal("expected error for invalid contract type")
	}
	if _, err := CalculatePayoff(ContractTypeCall, 1000, 100, 1100, "HOLD", 1000); err == nil {
		t.Fatal("expected error for invalid position type")
	}
	if _, err := ClassifyMoneyness(ContractTypeCall, 0, 1000, DefaultATMThresholdPct); err == nil {
		t.Fatal("expected error for zero strike price")
	}
}
