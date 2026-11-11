package enum

type ContractStatus string

const (
	ContractStatusActive   ContractStatus = "ACTIVE"
	ContractStatusExpired  ContractStatus = "EXPIRED"
	ContractStatusDelisted ContractStatus = "DELISTED"
)

func (s ContractStatus) String() string {
	switch s {
	case ContractStatusActive:
		return "فعال"
	case ContractStatusExpired:
		return "منقضی شده"
	case ContractStatusDelisted:
		return "حذف شده"
	default:
		return "unknown"
	}
}
