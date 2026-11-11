package enum

type QualityStatus string

const (
	QualityStatusPassed  QualityStatus = "PASSED"
	QualityStatusWarning QualityStatus = "WARNING"
	QualityStatusFailed  QualityStatus = "FAILED"
)

func (s QualityStatus) String() string {
	switch s {
	case QualityStatusPassed:
		return "موفق"
	case QualityStatusWarning:
		return "هشدار"
	case QualityStatusFailed:
		return "ناموفق"
	default:
		return "unknown"
	}
}
