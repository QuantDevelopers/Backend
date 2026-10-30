"""Options analytics and financial domain calculations for Tehran Stock Exchange.

Card: E9-1 · Options Primer
Provides programmatic calculations for:
- Payoff at expiration (Long/Short Call/Put)
- Break-even prices
- Intrinsic and Time Value breakdown
- Moneyness classification (ITM, ATM, OTM)
- Effective leverage ratio
"""

from dataclasses import dataclass
from enum import StrEnum


class ContractType(StrEnum):
    CALL = "CALL"
    PUT = "PUT"


class PositionType(StrEnum):
    LONG = "LONG"    # خریدار (دارنده حق)
    SHORT = "SHORT"  # فروشنده (متعهد)


class Moneyness(StrEnum):
    ITM = "ITM"  # در سود (In-The-Money)
    ATM = "ATM"  # بی‌تفاوت (At-The-Money)
    OTM = "OTM"  # در زیان (Out-Of-The-Money)


@dataclass
class OptionValueBreakdown:
    premium: float
    intrinsic_value: float
    time_value: float
    moneyness: Moneyness
    breakeven_price: float


def calculate_breakeven(contract_type: ContractType | str, strike: float, premium: float) -> float:
    """Calculate the break-even underlying spot price at expiration.

    Call Breakeven = Strike + Premium
    Put Breakeven = Strike - Premium
    """
    c_type = ContractType(contract_type.upper())
    if c_type == ContractType.CALL:
        return strike + premium
    return strike - premium


def calculate_intrinsic_value(contract_type: ContractType | str, strike: float, spot_price: float) -> float:
    """Calculate the intrinsic value (ارزش ذاتی) of an option.

    Call Intrinsic Value = max(0, Spot - Strike)
    Put Intrinsic Value = max(0, Strike - Spot)
    """
    c_type = ContractType(contract_type.upper())
    if c_type == ContractType.CALL:
        return max(0.0, spot_price - strike)
    return max(0.0, strike - spot_price)


def classify_moneyness(
    contract_type: ContractType | str,
    strike: float,
    spot_price: float,
    atm_threshold_pct: float = 0.01,
) -> Moneyness:
    """Determine whether an option is In-The-Money (ITM), At-The-Money (ATM), or Out-Of-The-Money (OTM).

    ATM is defined as spot being within +/- atm_threshold_pct (default 1%) of strike.
    """
    c_type = ContractType(contract_type.upper())
    rel_diff = abs(spot_price - strike) / strike

    if rel_diff <= atm_threshold_pct:
        return Moneyness.ATM

    if c_type == ContractType.CALL:
        return Moneyness.ITM if spot_price > strike else Moneyness.OTM
    else:
        return Moneyness.ITM if spot_price < strike else Moneyness.OTM


def analyze_option_value(
    contract_type: ContractType | str,
    strike: float,
    premium: float,
    spot_price: float,
) -> OptionValueBreakdown:
    """Break down the option premium into Intrinsic Value and Time Value, and classify moneyness."""
    intrinsic = calculate_intrinsic_value(contract_type, strike, spot_price)
    time_val = max(0.0, premium - intrinsic)
    moneyness = classify_moneyness(contract_type, strike, spot_price)
    be = calculate_breakeven(contract_type, strike, premium)

    return OptionValueBreakdown(
        premium=premium,
        intrinsic_value=intrinsic,
        time_value=time_val,
        moneyness=moneyness,
        breakeven_price=be,
    )


def calculate_payoff(
    contract_type: ContractType | str,
    strike: float,
    premium: float,
    spot_at_expiry: float,
    position: PositionType | str = PositionType.LONG,
    contract_size: int = 1000,
) -> float:
    """Calculate net profit/loss (سود و زیان نهایی) at expiration per contract or total.

    Long Call:  max(S - K, 0) - Premium
    Short Call: -(max(S - K, 0)) + Premium
    Long Put:   max(K - S, 0) - Premium
    Short Put:  -(max(K - S, 0)) + Premium
    """
    c_type = ContractType(contract_type.upper())
    pos = PositionType(position.upper())

    if c_type == ContractType.CALL:
        gross_payoff = max(0.0, spot_at_expiry - strike)
    else:
        gross_payoff = max(0.0, strike - spot_at_expiry)

    if pos == PositionType.LONG:
        net_per_share = gross_payoff - premium
    else:
        net_per_share = premium - gross_payoff

    return net_per_share * contract_size


def calculate_effective_leverage(spot_price: float, premium: float, delta: float = 1.0) -> float:
    """Calculate the operational leverage ratio: Omega = |Delta| * (Spot / Premium).

    Measures the percentage change in option price relative to 1% change in underlying.
    """
    if premium <= 0:
        return 0.0
    return abs(delta) * (spot_price / premium)
