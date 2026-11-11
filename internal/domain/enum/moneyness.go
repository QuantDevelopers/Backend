package enum

type Moneyness string

const (
	MoneynessITM Moneyness = "ITM"
	MoneynessATM Moneyness = "ATM"
	MoneynessOTM Moneyness = "OTM"
)

func (m Moneyness) String() string {
	switch m {
	case MoneynessITM:
		return "در سود (ITM)"
	case MoneynessATM:
		return "بی‌تفاوت (ATM)"
	case MoneynessOTM:
		return "در زیان (OTM)"
	default:
		return "unknown"
	}
}
