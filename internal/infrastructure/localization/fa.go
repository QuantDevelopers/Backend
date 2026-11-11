package localization

var Persian = map[string]interface{}{
	"underlyingAsset": "دارایی پایه",
	"optionContract":  "قرارداد اختیار معامله",
	"dailyHistory":    "تاریخچه روزانه",
	"tickRaw":         "تیک خام",
	"tickClean":       "تیک پالایش‌شده",
	"qualityLog":      "گزارش کیفیت داده",
	"symbolFA":        "نماد فارسی",
	"contractType":    "نوع قرارداد",
	"strikePrice":     "قیمت اعمال",
	"page":            "صفحه",
	"pageSize":        "اندازه صفحه",
	"errors": map[string]interface{}{
		"generic":        "خطایی رخ داده است، لطفا دوباره تلاش کنید.",
		"numeric":        "`{0}` باید عدد باشد.",
		"fileRequired":   "وارد کردن فایل اجباری است.",
		"required":       "بخش {0} باید کامل شود.",
		"invalid":        "مقدار {0} نامعتبر است.",
		"notExist":       "{0} وجود ندارد.",
		"alreadyExist":   "{0} از قبل وجود دارد.",
		"notFound":       "{0} یافت نشد.",
		"forbiddenError": "شما دسترسی لازم به {0} را ندارید.",
		"unauthorized":   "ابتدا باید وارد شوید.",
		"rateLimit":      "تعداد درخواست‌ها از حد مجاز بیشتر شده است.",
	},
}
