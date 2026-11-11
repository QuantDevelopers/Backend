package bootstrap

type Constants struct {
	Context          Context
	LogLevel         LogLevel
	Field            ErrorField
	Tag              ErrorTag
	Metrics          Metrics
	InstrumentTypes  InstrumentTypes
	ContractStatuses ContractStatuses
	QualityStatuses  QualityStatuses
}

type Context struct {
	Translator                   string
	IsLoadedValidationTranslator string
	RequestID                    string
}

type LogLevel struct {
	Debug string
	Info  string
	Warn  string
	Error string
	Fatal string
}

type ErrorField struct {
	UnderlyingAsset string
	OptionContract  string
	DailyHistory    string
	TickRaw         string
	TickClean       string
	QualityLog      string
	SymbolFA        string
	ContractType    string
	StrikePrice     string
	Page            string
	PageSize        string
}

type ErrorTag struct {
	Invalid       string
	NotExist      string
	AlreadyExist  string
	Required      string
	InvalidNumber string
	Forbidden     string
	Unauthorized  string
	RateLimited   string
}

type Metrics struct {
	HTTPRequestsTotal   MetricOptions
	HTTPRequestDuration MetricOptions
}

type MetricOptions struct {
	Name string
	Help string
}

type InstrumentTypes struct {
	Underlying string
	Option     string
}

type ContractStatuses struct {
	Active   string
	Expired  string
	Delisted string
}

type QualityStatuses struct {
	Passed  string
	Warning string
	Failed  string
}

func NewConstants() *Constants {
	return &Constants{
		Context: Context{
			Translator:                   "translator",
			IsLoadedValidationTranslator: "isLoadedValidationTranslator",
			RequestID:                    "requestID",
		},
		LogLevel: LogLevel{
			Debug: "debug",
			Info:  "info",
			Warn:  "warn",
			Error: "error",
			Fatal: "fatal",
		},
		Field: ErrorField{
			UnderlyingAsset: "underlyingAsset",
			OptionContract:  "optionContract",
			DailyHistory:    "dailyHistory",
			TickRaw:         "tickRaw",
			TickClean:       "tickClean",
			QualityLog:      "qualityLog",
			SymbolFA:        "symbolFA",
			ContractType:    "contractType",
			StrikePrice:     "strikePrice",
			Page:            "page",
			PageSize:        "pageSize",
		},
		Tag: ErrorTag{
			Invalid:       "invalid",
			NotExist:      "notExist",
			AlreadyExist:  "alreadyExist",
			Required:      "required",
			InvalidNumber: "invalidNumber",
			Forbidden:     "forbidden",
			Unauthorized:  "unauthorized",
			RateLimited:   "rateLimited",
		},
		Metrics: Metrics{
			HTTPRequestsTotal: MetricOptions{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			HTTPRequestDuration: MetricOptions{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration in seconds",
			},
		},
		InstrumentTypes: InstrumentTypes{
			Underlying: "UNDERLYING",
			Option:     "OPTION",
		},
		ContractStatuses: ContractStatuses{
			Active:   "ACTIVE",
			Expired:  "EXPIRED",
			Delisted: "DELISTED",
		},
		QualityStatuses: QualityStatuses{
			Passed:  "PASSED",
			Warning: "WARNING",
			Failed:  "FAILED",
		},
	}
}
