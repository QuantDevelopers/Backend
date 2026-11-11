package localization

var English = map[string]interface{}{
	"underlyingAsset": "underlying asset",
	"optionContract":  "option contract",
	"dailyHistory":    "daily history",
	"tickRaw":         "raw tick",
	"tickClean":       "clean tick",
	"qualityLog":      "data quality log",
	"symbolFA":        "Persian symbol",
	"contractType":    "contract type",
	"strikePrice":     "strike price",
	"page":            "page",
	"pageSize":        "page size",
	"errors": map[string]interface{}{
		"generic":        "an error occurred, please try again.",
		"numeric":        "The {0} should be a numeric value.",
		"fileRequired":   "File is required.",
		"required":       "Field {0} is required.",
		"invalid":        "Field {0} is invalid.",
		"notExist":       "{0} does not exist.",
		"alreadyExist":   "{0} already exists.",
		"notFound":       "{0} not found.",
		"forbiddenError": "You do not have permission to access {0}.",
		"unauthorized":   "Unauthorized access.",
		"rateLimit":      "Rate limit exceeded. Please try again later.",
	},
}
