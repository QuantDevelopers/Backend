from src.domain.options_calculator import (
    ContractType,
    Moneyness,
    PositionType,
    analyze_option_value,
    calculate_breakeven,
    calculate_effective_leverage,
    calculate_intrinsic_value,
    calculate_payoff,
    classify_moneyness,
)


def test_call_option_calculations():
    # Example: Khodro stock Spot = 3000, Strike = 2800, Premium = 350
    spot = 3000.0
    strike = 2800.0
    premium = 350.0

    # 1. Breakeven = 2800 + 350 = 3150
    be = calculate_breakeven("CALL", strike, premium)
    assert be == 3150.0

    # 2. Intrinsic Value = max(0, 3000 - 2800) = 200
    intrinsic = calculate_intrinsic_value("CALL", strike, spot)
    assert intrinsic == 200.0

    # 3. Moneyness: spot > strike => ITM
    moneyness = classify_moneyness("CALL", strike, spot)
    assert moneyness == Moneyness.ITM

    # 4. Value Breakdown: Intrinsic = 200, Time Value = 350 - 200 = 150
    breakdown = analyze_option_value(ContractType.CALL, strike, premium, spot)
    assert breakdown.intrinsic_value == 200.0
    assert breakdown.time_value == 150.0
    assert breakdown.moneyness == Moneyness.ITM
    assert breakdown.breakeven_price == 3150.0

    # 5. Payoff at Expiry:
    # If S_T = 3500 => Gross = 700, Net per share = 700 - 350 = 350, Total (1000 shares) = 350,000
    payoff_profit = calculate_payoff("CALL", strike, premium, spot_at_expiry=3500.0, position="LONG")
    assert payoff_profit == 350000.0

    # If S_T = 2500 (below strike) => Option expires worthless, Net loss = -350 * 1000 = -350,000
    payoff_loss = calculate_payoff("CALL", strike, premium, spot_at_expiry=2500.0, position="LONG")
    assert payoff_loss == -350000.0

    # Short position mirror:
    short_gain = calculate_payoff("CALL", strike, premium, spot_at_expiry=2500.0, position="SHORT")
    assert short_gain == 350000.0


def test_put_option_calculations():
    # Example: Ahram Put Option Spot = 70000, Strike = 75000, Premium = 8000
    spot = 70000.0
    strike = 75000.0
    premium = 8000.0

    # 1. Breakeven = 75000 - 8000 = 67000
    be = calculate_breakeven(ContractType.PUT, strike, premium)
    assert be == 67000.0

    # 2. Intrinsic Value = max(0, 75000 - 70000) = 5000
    intrinsic = calculate_intrinsic_value("PUT", strike, spot)
    assert intrinsic == 5000.0

    # 3. Moneyness: spot < strike => ITM for Put
    moneyness = classify_moneyness("PUT", strike, spot)
    assert moneyness == Moneyness.ITM

    # 4. Breakdown: Intrinsic = 5000, Time Value = 8000 - 5000 = 3000
    breakdown = analyze_option_value(ContractType.PUT, strike, premium, spot)
    assert breakdown.intrinsic_value == 5000.0
    assert breakdown.time_value == 3000.0

    # 5. Payoff at Expiry:
    # If S_T = 60000 => Gross = 15000, Net = 15000 - 8000 = 7000 * 1000 = 7,000,000
    profit = calculate_payoff("PUT", strike, premium, spot_at_expiry=60000.0, position=PositionType.LONG)
    assert profit == 7000000.0

    # If S_T = 80000 => Option expires worthless, loss = -8000 * 1000 = -8,000,000
    loss = calculate_payoff("PUT", strike, premium, spot_at_expiry=80000.0, position=PositionType.LONG)
    assert loss == -8000000.0


def test_otm_and_atm_classification():
    # Call with strike 1000
    assert classify_moneyness("CALL", strike=1000, spot_price=1005) == Moneyness.ATM
    assert classify_moneyness("CALL", strike=1000, spot_price=1200) == Moneyness.ITM
    assert classify_moneyness("CALL", strike=1000, spot_price=800) == Moneyness.OTM

    # Put with strike 1000
    assert classify_moneyness("PUT", strike=1000, spot_price=995) == Moneyness.ATM
    assert classify_moneyness("PUT", strike=1000, spot_price=800) == Moneyness.ITM
    assert classify_moneyness("PUT", strike=1000, spot_price=1200) == Moneyness.OTM


def test_effective_leverage():
    # Spot = 70000, Premium = 7000 => Leverage = 10x
    leverage = calculate_effective_leverage(spot_price=70000.0, premium=7000.0, delta=1.0)
    assert leverage == 10.0

    # Zero premium edge case
    assert calculate_effective_leverage(spot_price=70000.0, premium=0.0) == 0.0
