package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	dto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	domaintext "github.com/QuantDevelopers/Backend/internal/domain/text"
)

const archiveURL = "https://cdn.tsetmc.com/api/ClosingPrice/GetInstrmentsHistoryInDay/%s"
const instrumentSearchURL = "https://cdn.tsetmc.com/api/Instrument/GetInstrumentSearch/%s"

// numericCode preserves 64-bit identifiers even when upstream sends a JSON number.
type numericCode string

func (c *numericCode) UnmarshalJSON(raw []byte) error {
	if string(raw) == "null" {
		return nil
	}
	var s string
	if len(raw) > 0 && raw[0] == '"' {
		if err := json.Unmarshal(raw, &s); err != nil {
			return err
		}
	} else {
		s = string(raw)
	}
	if s != "" {
		if err := validateInsCode(s); err != nil {
			return err
		}
	}
	*c = numericCode(s)
	return nil
}

type archiveInstrument struct {
	InsCode      numericCode `json:"insCode"`
	Symbol       string      `json:"lVal18AFC"`
	Name         string      `json:"lVal30"`
	ISIN         string      `json:"cIsin"`
	InstrumentID string      `json:"instrumentID"`
	ContractSize int         `json:"contractSize"`
}
type archiveEntry struct {
	archiveInstrument
	Instrument *archiveInstrument `json:"instrument"`
}

func (e archiveEntry) identity() archiveInstrument {
	if e.Instrument == nil {
		return e.archiveInstrument
	}
	row := *e.Instrument
	if row.InsCode == "" {
		row.InsCode = e.InsCode
	}
	return row
}

func historicalDates(opts dto.FullRegistryOptions) ([]string, error) {
	start, end := opts.HistoryStartDate, opts.HistoryEndDate
	if len(opts.HistoryDates) > 0 && (start != "" || end != "") {
		return nil, fmt.Errorf("use history dates or a history range, not both")
	}
	if len(opts.HistoryDates) > 0 {
		dates := append([]string(nil), opts.HistoryDates...)
		for _, d := range dates {
			if _, err := parseTSETMCDate(d); err != nil {
				return nil, err
			}
		}
		for _, d := range dates {
			if d > marketToday() {
				return nil, fmt.Errorf("future archive date %s", d)
			}
		}
		dates = normalizeDateList(dates)
		sort.Sort(sort.Reverse(sort.StringSlice(dates)))
		return dates, nil
	}
	// The live default covers the project's two-year target. Offline watch-only
	// runs must not imply archive coverage when no archive fixtures were supplied.
	if opts.UseOfflineFixture && start == "" && end == "" {
		return nil, nil
	}
	today, _ := parseTSETMCDate(marketToday())
	if end == "" {
		end = today.AddDate(0, 0, -1).Format("20060102")
	}
	if start == "" {
		start = today.AddDate(-2, 0, 0).Format("20060102")
	}
	first, err := parseTSETMCDate(start)
	if err != nil {
		return nil, err
	}
	last, err := parseTSETMCDate(end)
	if err != nil {
		return nil, err
	}
	if first.After(last) || last.After(today) || last.Sub(first) > 366*24*time.Hour*20 {
		return nil, fmt.Errorf("history range must be ordered, not in the future, and at most 20 years")
	}
	dates := []string{}
	// Include every calendar day; a weekday assumption must not hide a snapshot.
	for day := last; !day.Before(first); day = day.AddDate(0, 0, -1) {
		dates = append(dates, day.Format("20060102"))
	}
	return dates, nil
}

type archiveStats struct {
	Dates          []string
	Added          int
	EstimatedSizes int
}

// discoverHistory scans the whole market, not prefixes of today's underlyings.
// Unknown mappings and malformed option names fail the run instead of silently
// losing a historical contract. Existing watch metadata wins over archive data.
func (s *DiscoveryService) discoverHistory(ctx context.Context, opts dto.FullRegistryOptions, underlyings []dto.DiscoveredUnderlying, contracts []dto.DiscoveredContract) ([]dto.DiscoveredUnderlying, []dto.DiscoveredContract, archiveStats, error) {
	for code, size := range opts.ContractSizes {
		if err := validateInsCode(code); err != nil {
			return nil, nil, archiveStats{}, err
		}
		if size <= 0 {
			return nil, nil, archiveStats{}, fmt.Errorf("invalid contract size for %s", code)
		}
	}
	dates, err := historicalDates(opts)
	stats := archiveStats{Dates: dates}
	if err != nil {
		return nil, nil, stats, err
	}
	bases := map[string]dto.DiscoveredUnderlying{}
	seenBases := map[string]bool{}
	seenOptions := map[string]bool{}
	baseIndexes := map[string]int{}
	optionIndexes := map[string]int{}
	for i, u := range underlyings {
		baseIndexes[u.InsCode] = i
		bases[domaintext.NormalizePersian(u.Symbol)] = u
		seenBases[u.InsCode] = true
	}
	for i, c := range contracts {
		optionIndexes[c.InsCode] = i
		seenOptions[c.InsCode] = true
	}
	cacheDir := opts.HistoryCacheDir
	if cacheDir == "" {
		cacheDir = filepath.Join(opts.RegistryDir, "history_cache")
	}
	if opts.RegistryDir == "" && opts.HistoryCacheDir == "" {
		cacheDir = "data/registry/history_cache"
	}
	for _, day := range dates {
		if err = ctx.Err(); err != nil {
			return nil, nil, stats, err
		}
		raw, e := s.archivePayload(ctx, opts, day, cacheDir)
		if e != nil {
			return nil, nil, stats, fmt.Errorf("archive %s: %w", day, e)
		}
		var payload struct {
			Rows []archiveEntry `json:"closingPriceDailyHistoryWithInstDetails"`
		}
		if e = json.Unmarshal(raw, &payload); e != nil {
			return nil, nil, stats, e
		}
		dayBases := map[string]dto.DiscoveredUnderlying{}
		// First pass also learns bases absent from today's watch.
		for _, entry := range payload.Rows {
			row := entry.identity()
			symbol := domaintext.NormalizePersian(row.Symbol)
			if row.InsCode == "" || symbol == "" {
				return nil, nil, stats, fmt.Errorf("archive %s: missing instrument identity", day)
			}
			if isEquity(row.ISIN) {
				market := "بورس"
				if strings.HasPrefix(row.ISIN, "IRT") {
					market = "صندوق"
				}
				base := dto.DiscoveredUnderlying{InsCode: string(row.InsCode), Symbol: symbol, Name: domaintext.NormalizePersian(row.Name), Market: market, ISIN: row.ISIN, InstrumentID: row.InstrumentID, DiscoveredBy: "tsetmc_daily_archive"}
				if previous, ok := dayBases[symbol]; ok && previous.InsCode != base.InsCode {
					return nil, nil, stats, fmt.Errorf("archive %s: ambiguous underlying symbol %s", day, symbol)
				}
				dayBases[symbol] = base
				if i, ok := baseIndexes[base.InsCode]; ok {
					if underlyings[i].ISIN == "" {
						underlyings[i].ISIN = base.ISIN
					}
					if underlyings[i].InstrumentID == "" {
						underlyings[i].InstrumentID = base.InstrumentID
					}
				}

				if !seenBases[base.InsCode] {
					baseIndexes[base.InsCode] = len(underlyings)
					underlyings = append(underlyings, base)
					seenBases[base.InsCode] = true
				}
				if _, ok := bases[symbol]; !ok {
					bases[symbol] = base
				}
			}
		}
		for _, entry := range payload.Rows {
			row := entry.identity()
			name := domaintext.NormalizePersian(normalizeDigits(row.Name))
			if !isOptionName(name) {
				continue
			}
			code := string(row.InsCode)
			if seenOptions[code] {
				i := optionIndexes[code]
				if contracts[i].ISIN == "" {
					contracts[i].ISIN = row.ISIN
				}
				if contracts[i].InstrumentID == "" {
					contracts[i].InstrumentID = row.InstrumentID
				}
				continue
			}

			baseSymbol, side, strike, expiry, e := parseHistoricalOption(name)
			if e != nil {
				return nil, nil, stats, fmt.Errorf("archive %s instrument %s: %w", day, code, e)
			}
			base, ok := dayBases[baseSymbol]
			if !ok {
				base, ok = bases[baseSymbol]
			}
			if !ok {
				base, e = s.resolveHistoricalUnderlying(ctx, opts, baseSymbol)
				if e != nil {
					return nil, nil, stats, fmt.Errorf("archive %s option %s: %w", day, code, e)
				}
				bases[baseSymbol] = base
				if !seenBases[base.InsCode] {
					baseIndexes[base.InsCode] = len(underlyings)
					underlyings = append(underlyings, base)
					seenBases[base.InsCode] = true
				}
			}
			size := row.ContractSize
			estimated := false
			if override, ok := opts.ContractSizes[code]; ok {
				size = override
			}
			if size <= 0 {
				size = 1000
				estimated = true
				stats.EstimatedSizes++
			}
			optionIndexes[code] = len(contracts)
			contracts = append(contracts, dto.DiscoveredContract{InsCode: code, Symbol: domaintext.NormalizePersian(row.Symbol), Name: name, Side: side, UaInsCode: base.InsCode, UnderlyingSymbol: base.Symbol, StrikePrice: strike, ContractSize: size, ContractSizeEstimated: estimated, ExpiryDate: expiry, ISIN: row.ISIN, InstrumentID: row.InstrumentID, DiscoveredBy: "tsetmc_daily_archive"})
			seenOptions[code] = true
			stats.Added++
		}
	}
	return underlyings, contracts, stats, nil
}

func isEquity(isin string) bool {
	return strings.HasPrefix(isin, "IRT") || strings.HasPrefix(isin, "IRO1") || strings.HasPrefix(isin, "IRO3") || strings.HasPrefix(isin, "IRO5") || strings.HasPrefix(isin, "IRO7")
}
func isOptionName(name string) bool {
	compact := strings.ReplaceAll(name, " ", "")
	return strings.HasPrefix(compact, "اختیارخ") || strings.HasPrefix(compact, "اختیارف")
}

func (s *DiscoveryService) archivePayload(ctx context.Context, opts dto.FullRegistryOptions, day, dir string) ([]byte, error) {
	if opts.UseOfflineFixture {
		if opts.HistoryFixtureDir == "" {
			return nil, fmt.Errorf("history-fixtures directory required for offline archive scan")
		}
		raw, err := os.ReadFile(filepath.Join(opts.HistoryFixtureDir, day+".json"))
		if err != nil {
			return nil, err
		}
		return raw, requireArray(raw, "closingPriceDailyHistoryWithInstDetails")
	}
	path := filepath.Join(dir, day+".json")
	metaPath := path + ".manifest.json"
	target := fmt.Sprintf(archiveURL, day)
	if !opts.RefreshHistory {
		raw, e := os.ReadFile(path)
		metadata, me := os.ReadFile(metaPath)
		var m struct {
			SHA256 string `json:"sha256"`
			URL    string `json:"url"`
		}
		if e == nil && me == nil && json.Unmarshal(metadata, &m) == nil && m.URL == target && m.SHA256 == ComputeSHA256(raw) && requireArray(raw, "closingPriceDailyHistoryWithInstDetails") == nil {
			return raw, nil
		}
	}
	if s.client == nil {
		return nil, fmt.Errorf("TSETMC client required")
	}
	raw, status, err := s.client.Get(ctx, target)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", status)
	}
	if err = requireArray(raw, "closingPriceDailyHistoryWithInstDetails"); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err = writeAtomic(path, raw); err != nil {
		return nil, err
	}
	if err = writeJSONPretty(metaPath, map[string]string{"url": target, "sha256": ComputeSHA256(raw)}); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *DiscoveryService) resolveHistoricalUnderlying(ctx context.Context, opts dto.FullRegistryOptions, symbol string) (dto.DiscoveredUnderlying, error) {
	var raw []byte
	var err error
	if opts.UseOfflineFixture {
		raw, err = os.ReadFile(filepath.Join(opts.HistoryFixtureDir, "search_"+symbol+".json"))
	} else {
		if s.client == nil {
			return dto.DiscoveredUnderlying{}, fmt.Errorf("TSETMC client required")
		}
		var status int
		raw, status, err = s.client.Get(ctx, fmt.Sprintf(instrumentSearchURL, url.PathEscape(symbol)))
		if err == nil && status != http.StatusOK {
			err = fmt.Errorf("underlying search HTTP %d", status)
		}
	}
	if err != nil {
		return dto.DiscoveredUnderlying{}, err
	}
	if err = requireArray(raw, "instrumentSearch"); err != nil {
		return dto.DiscoveredUnderlying{}, err
	}
	var payload struct {
		Rows []archiveInstrument `json:"instrumentSearch"`
	}
	if err = json.Unmarshal(raw, &payload); err != nil {
		return dto.DiscoveredUnderlying{}, err
	}
	var matches []archiveInstrument
	for _, row := range payload.Rows {
		if domaintext.NormalizePersian(row.Symbol) == symbol && row.InsCode != "" && !isOptionName(domaintext.NormalizePersian(row.Name)) {
			matches = append(matches, row)
		}
	}
	if len(matches) != 1 {
		return dto.DiscoveredUnderlying{}, fmt.Errorf("underlying %q requires one exact identity, got %d", symbol, len(matches))
	}
	row := matches[0]
	market := "بورس"
	if strings.HasPrefix(row.ISIN, "IRT") {
		market = "صندوق"
	}
	return dto.DiscoveredUnderlying{InsCode: string(row.InsCode), Symbol: symbol, Name: domaintext.NormalizePersian(row.Name), Market: market, ISIN: row.ISIN, InstrumentID: row.InstrumentID, DiscoveredBy: "tsetmc_instrument_search"}, nil
}

func normalizeDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '۰' && r <= '۹' {
			return '0' + r - '۰'
		}
		if r >= '٠' && r <= '٩' {
			return '0' + r - '٠'
		}
		return r
	}, s)
}
func parseHistoricalOption(name string) (base, side string, strike float64, expiry string, err error) {
	parts := strings.Split(strings.NewReplacer("–", "-", "−", "-", "—", "-").Replace(normalizeDigits(name)), "-")
	if len(parts) < 3 {
		err = fmt.Errorf("unrecognized option name %q", name)
		return
	}
	expiry, err = optionExpiry(strings.TrimSpace(parts[len(parts)-1]))
	if err != nil {
		return
	}
	strike, err = strconv.ParseFloat(strings.NewReplacer(",", "", "٬", "").Replace(strings.TrimSpace(parts[len(parts)-2])), 64)
	if err != nil || strike <= 0 {
		err = fmt.Errorf("invalid strike in %q", name)
		return
	}
	prefix := strings.Join(parts[:len(parts)-2], "-")
	prefix = strings.ReplaceAll(prefix, "اختیار ", "اختیار")
	for _, p := range []struct{ Text, Side string }{{"اختیار خرید", "Call"}, {"اختیار فروش", "Put"}, {"اختیارخرید", "Call"}, {"اختیارفروش", "Put"}, {"اختیارخ", "Call"}, {"اختیارف", "Put"}} {
		if strings.HasPrefix(prefix, p.Text) {
			base = strings.TrimSpace(strings.TrimPrefix(prefix, p.Text))
			side = p.Side
			break
		}
	}
	if base == "" {
		err = fmt.Errorf("missing underlying in %q", name)
	}
	return
}

func optionExpiry(s string) (string, error) {
	s = strings.ReplaceAll(s, "/", "")
	if len(s) != 8 {
		return "", fmt.Errorf("invalid option expiry %q", s)
	}
	year, e := strconv.Atoi(s[:4])
	if e != nil {
		return "", e
	}
	if year >= 1700 {
		d, err := parseTSETMCDate(s)
		if err != nil {
			return "", err
		}
		return d.Format("20060102"), nil
	}
	month, e := strconv.Atoi(s[4:6])
	if e != nil {
		return "", e
	}
	day, e := strconv.Atoi(s[6:])
	if e != nil {
		return "", e
	}
	d, e := jalaliDate(year, month, day)
	if e != nil {
		return "", e
	}
	return d.Format("20060102"), nil
}

// Borkowski's break-year calculation, bounded to the modern Iranian market era.
func jalaliDate(year, month, day int) (time.Time, error) {
	if year < 1300 || year > 1500 || month < 1 || month > 12 || day < 1 {
		return time.Time{}, fmt.Errorf("invalid Jalali date")
	}
	breaks := []int{-61, 9, 38, 199, 426, 686, 756, 818, 1111, 1181, 1210, 1635, 2060, 2097, 2192, 2262, 2324, 2394, 2456, 3178}
	gy := year + 621
	leaps := -14
	jp := breaks[0]
	jump := 0
	for _, jm := range breaks[1:] {
		jump = jm - jp
		if year < jm {
			break
		}
		leaps += jump/33*8 + (jump%33)/4
		jp = jm
	}
	n := year - jp
	leaps += n/33*8 + (n%33+3)/4
	if jump%33 == 4 && jump-n == 4 {
		leaps++
	}
	march := 20 + leaps - (gy/4 - ((gy/100+1)*3)/4 - 150)
	if jump-n < 6 {
		n = n - jump + (jump+4)/33*33
	}
	leap := ((n+1)%33 - 1) % 4
	if leap == -1 {
		leap = 4
	}
	limit := 31
	if month > 6 {
		limit = 30
	}
	if month == 12 && leap != 0 {
		limit = 29
	}
	if day > limit {
		return time.Time{}, fmt.Errorf("invalid Jalali day")
	}
	offset := (month - 1) * 31
	if month > 6 {
		offset = 186 + (month-7)*30
	}
	return time.Date(gy, time.March, march, 0, 0, 0, 0, time.UTC).AddDate(0, 0, offset+day-1), nil
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
