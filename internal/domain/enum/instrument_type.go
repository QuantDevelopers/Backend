package enum

type InstrumentType string

const (
	InstrumentTypeUnderlying InstrumentType = "UNDERLYING"
	InstrumentTypeOption     InstrumentType = "OPTION"
)

func (i InstrumentType) String() string {
	switch i {
	case InstrumentTypeUnderlying:
		return "دارایی پایه"
	case InstrumentTypeOption:
		return "قرارداد اختیار"
	default:
		return "unknown"
	}
}
