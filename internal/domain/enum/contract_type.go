package enum

import (
	"fmt"
	"strings"
)

type ContractType string

const (
	ContractTypeCall ContractType = "CALL"
	ContractTypePut  ContractType = "PUT"
)

func (c ContractType) String() string {
	switch c {
	case ContractTypeCall:
		return "اختیار خرید (Call)"
	case ContractTypePut:
		return "اختیار فروش (Put)"
	default:
		return "unknown"
	}
}

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

func GetAllContractTypes() []ContractType {
	return []ContractType{ContractTypeCall, ContractTypePut}
}
