package text

import (
	"regexp"
	"strings"
)

var (
	arabicToPersianReplacer = strings.NewReplacer(
		"ي", "ی",
		"ى", "ی",
		"ك", "ک",
		"ة", "ه",
		"ؤ", "و",
		"إ", "ا",
		"أ", "ا",
		"آ", "آ",
		"١", "۱",
		"٢", "۲",
		"٣", "۳",
		"٤", "۴",
		"٥", "۵",
		"٦", "۶",
		"٧", "۷",
		"٨", "۸",
		"٩", "۹",
		"٠", "۰",
	)

	digitsToASCIIReplacer = strings.NewReplacer(
		"۰", "0", "۱", "1", "۲", "2", "۳", "3", "۴", "4",
		"۵", "5", "۶", "6", "۷", "7", "۸", "8", "۹", "9",
		"٠", "0", "١", "1", "٢", "2", "٣", "3", "٤", "4",
		"٥", "5", "٦", "6", "٧", "7", "٨", "8", "٩", "9",
	)

	multiSpaceRe = regexp.MustCompile(`\s+`)
	multiZWNJRe  = regexp.MustCompile("\u200c+")
)

// NormalizePersian normalizes Arabic glyphs/digits into standard Persian typography
// and collapses repeated whitespace and zero-width non-joiners.
func NormalizePersian(input string) string {
	if input == "" {
		return ""
	}
	normalized := arabicToPersianReplacer.Replace(input)
	normalized = multiZWNJRe.ReplaceAllString(normalized, "\u200c")
	normalized = multiSpaceRe.ReplaceAllString(normalized, " ")
	return strings.TrimSpace(normalized)
}

// ToASCIIDigits converts Persian and Arabic-Indic numerals into ASCII digits (0-9).
func ToASCIIDigits(input string) string {
	if input == "" {
		return ""
	}
	return digitsToASCIIReplacer.Replace(input)
}
