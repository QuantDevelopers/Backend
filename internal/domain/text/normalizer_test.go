package text_test

import (
	"testing"

	"github.com/QuantDevelopers/Backend/internal/domain/text"
	"github.com/stretchr/testify/suite"
)

type NormalizerTestSuite struct {
	suite.Suite
}

func (s *NormalizerTestSuite) TestNormalizePersian() {
	s.Equal("", text.NormalizePersian(""))
	s.Equal("", text.NormalizePersian("   \t\n "))
	s.Equal("اختیار\u200c خ کولاد-۱۲۳", text.NormalizePersian("  اختيار\u200c\u200c خ   كولاد-١٢٣  "))

	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "arabic yeh and kaf",
			input:    "صندوق سرمايه گذاري كاري",
			expected: "صندوق سرمایه گذاری کاری",
		},
		{
			name:     "arabic digits to persian",
			input:    "اهرم-١٤٠٥/١٠/٢٣",
			expected: "اهرم-۱۴۰۵/۱۰/۲۳",
		},
		{
			name:     "collapse repeated spaces and zwnj",
			input:    "  نرخ\u200c\u200c\u200cهای   اختیار   ",
			expected: "نرخ\u200cهای اختیار",
		},
		{
			name:     "alef variants and ta marbuta",
			input:    "إدارة أزمة مؤقتة",
			expected: "اداره ازمه موقته",
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			s.Equal(tc.expected, text.NormalizePersian(tc.input))
		})
	}
}

func (s *NormalizerTestSuite) TestToASCIIDigits() {
	s.Equal("", text.ToASCIIDigits(""))
	s.Equal("1405/10/23", text.ToASCIIDigits("۱۴۰۵/۱۰/۲۳"))
	s.Equal("9876543210", text.ToASCIIDigits("٩٨٧٦٥٤٣٢١٠"))
	s.Equal("abc-123-09", text.ToASCIIDigits("abc-۱۲۳-٠٩"))
}

func TestNormalizerSuite(t *testing.T) {
	suite.Run(t, new(NormalizerTestSuite))
}
