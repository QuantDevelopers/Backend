package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	domainrepo "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	dbMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/database"
	repoMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/repository/postgres"
	tsetmcMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/tsetmc"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type DiscoveryServiceTestSuite struct {
	suite.Suite
	client         *tsetmcMocks.TSETMCClientMock
	underlyingRepo *repoMocks.UnderlyingRepositoryMock
	optionRepo     *repoMocks.OptionContractRepositoryMock
	db             *dbMocks.DatabaseMock
	service        *DiscoveryService
}

func (s *DiscoveryServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	log, err := infralogger.NewLogger(&bootstrap.Logger{
		LogLevel:      "error",
		ConsoleOutput: "false",
	}, cfg.Constants)
	s.Require().NoError(err)

	s.client = tsetmcMocks.NewTSETMCClientMock()
	s.underlyingRepo = repoMocks.NewUnderlyingRepositoryMock()
	s.optionRepo = repoMocks.NewOptionContractRepositoryMock()
	s.db = dbMocks.NewDatabaseMock()
	s.service = NewDiscoveryService(s.client, s.underlyingRepo, s.optionRepo, s.db, log)
}

func (s *DiscoveryServiceTestSuite) TestParseMarketWatchOfflineFixture() {
	fixturePath := filepath.Join("..", "..", "..", "tests", "fixtures", "sample_option_watch.json")
	registryDir := s.T().TempDir()

	res, err := s.service.Discover(context.Background(), crawlerdto.DiscoveryOptions{
		UseOfflineFixture: true,
		FixturePath:       fixturePath,
		RegistryDir:       registryDir,
	})
	s.Require().NoError(err)
	s.Len(res.Underlyings, 27)
	s.Len(res.Contracts, 1634)
	s.FileExists(res.UnderlyingsPath)
	s.FileExists(res.ContractsPath)

	var foundAhram *crawlerdto.DiscoveredUnderlying
	for i := range res.Underlyings {
		if res.Underlyings[i].Symbol == "اهرم" {
			foundAhram = &res.Underlyings[i]
			break
		}
	}
	s.Require().NotNil(foundAhram)
	s.Equal("17914401175772326", foundAhram.InsCode)
	s.Equal("صندوق", foundAhram.Market)
	s.Equal(foundAhram.CallCount+foundAhram.PutCount, foundAhram.ActiveOptionsCount)
	s.Greater(foundAhram.ActiveOptionsCount, 0)

	rawSaved, err := os.ReadFile(res.UnderlyingsPath)
	s.Require().NoError(err)
	var decoded []crawlerdto.DiscoveredUnderlying
	s.Require().NoError(json.Unmarshal(rawSaved, &decoded))
	s.Len(decoded, 27)
}

func (s *DiscoveryServiceTestSuite) TestParseNormalizesArabicAndCalculatesPriceChange() {
	payload := `{
		"instrumentOptMarketWatch": [
			{
				"uaInsCode": "12345",
				"lval30_UA": "صندوق  اهرم كاري",
				"pDrCotVal_UA": 1100,
				"pClosing_UA": 1080,
				"priceYesterday_UA": 1000,
				"beginDate": "20261017",
				"endDate": "20270113",
				"strikePrice": 5000,
				"contractSize": 1000,
				"remainedDay": 60,
				"insCode_C": "9991",
				"lVal18AFC_C": "ضهرم101",
				"lVal30_C": "اختيارخ اهرم-5000",
				"insCode_P": "9992",
				"lVal18AFC_P": "طهرم101",
				"lVal30_P": "اختيارف اهرم-5000"
			}
		]
	}`

	underlyings, contracts, err := s.service.ParseMarketWatch([]byte(payload))
	s.Require().NoError(err)
	s.Require().Len(underlyings, 1)
	s.Require().Len(contracts, 2)

	s.Equal("صندوق اهرم کاری", underlyings[0].Symbol)
	s.Equal("صندوق", underlyings[0].Market)
	s.InDelta(10.0, underlyings[0].PriceChange, 0.001)
	s.Equal(1, underlyings[0].CallCount)
	s.Equal(1, underlyings[0].PutCount)
	s.Equal("Call", contracts[0].Side)
	s.Equal("Put", contracts[1].Side)
	s.Equal("اختیارخ اهرم-5000", contracts[0].Name)
}

func (s *DiscoveryServiceTestSuite) TestOfflinePersistDB() {
	miniWatch := `{
		"instrumentOptMarketWatch": [
			{
				"uaInsCode": "17914401175772326",
				"lval30_UA": "اهرم",
				"pDrCotVal_UA": 72740,
				"priceYesterday_UA": 70744,
				"beginDate": "20261017",
				"endDate": "20270113",
				"strikePrice": 70000,
				"contractSize": 1000,
				"remainedDay": 40,
				"insCode_C": "64308663607254212",
				"lVal18AFC_C": "ضهرم1055",
				"lVal30_C": "اختيارخ اهرم-70000"
			}
		]
	}`
	tmpFixture := filepath.Join(s.T().TempDir(), "watch.json")
	s.Require().NoError(os.WriteFile(tmpFixture, []byte(miniWatch), 0o644))

	existingUA := &entity.UnderlyingAsset{
		Model:    database.Model{ID: 7},
		SymbolFA: "اهرم",
	}
	s.underlyingRepo.On("GetBySymbol", s.db, "اهرم").Return(existingUA, nil).Once()
	s.underlyingRepo.On("Upsert", s.db, existingUA).Return(nil).Once()
	s.optionRepo.On("SaveBulk", s.db, mock.MatchedBy(func(items []*entity.OptionContract) bool {
		return len(items) == 1 && items[0].UnderlyingAssetID == 7 && items[0].ContractType == "CALL"
	})).Return(1, nil).Once()

	res, err := s.service.Discover(context.Background(), crawlerdto.DiscoveryOptions{
		UseOfflineFixture: true,
		FixturePath:       tmpFixture,
		RegistryDir:       s.T().TempDir(),
		PersistDB:         true,
	})
	s.Require().NoError(err)
	s.Equal(1, res.PersistedDBRows)
	s.client.AssertExpectations(s.T())
	s.underlyingRepo.AssertExpectations(s.T())
	s.optionRepo.AssertExpectations(s.T())
}

func (s *DiscoveryServiceTestSuite) TestDiscoverFullRegistryE15() {
	fixturePath := filepath.Join("..", "..", "..", "tests", "fixtures", "sample_option_watch.json")
	registryDir := s.T().TempDir()
	expiredIns := "99988877766655544"

	s.optionRepo.On("GetAll", s.db, domainrepo.OptionFilter{}).Return([]*entity.OptionContract{
		{
			Model:          database.Model{ID: 99},
			SymbolFA:       "ضفول9999",
			NameFA:         "اختیارخ فولاد منقضی",
			ContractType:   "CALL",
			StrikePrice:    4000,
			ContractSize:   1000,
			ExpirationDate: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			TSETMCID:       &expiredIns,
			IsActive:       false,
			Status:         "EXPIRED",
			UnderlyingAsset: &entity.UnderlyingAsset{
				SymbolFA: "فولاد",
			},
		},
	}, nil).Once()

	res, err := s.service.DiscoverFullRegistry(context.Background(), crawlerdto.FullRegistryOptions{
		UseOfflineFixture: true,
		FixturePath:       fixturePath,
		RegistryDir:       registryDir,
		IncludeExpired:    true,
	})
	s.Require().NoError(err)
	s.Equal(27, res.Summary.UnderlyingsCount)
	s.Equal(1634+1, res.Summary.ActiveOptionsCount+res.Summary.ExpiredOptionsCount)
	s.Equal(27+1634+1, res.Summary.TotalInstruments)
	s.Greater(res.Summary.CallCount, 0)
	s.Greater(res.Summary.PutCount, 0)
	s.Greater(res.Summary.ExpiriesCount, 0)

	s.FileExists(res.InstrumentsPath)
	s.FileExists(res.UnderlyingsPath)
	s.FileExists(res.ActivePath)
	s.FileExists(res.ExpiredPath)
	s.FileExists(res.SummaryPath)

	rawSaved, err := os.ReadFile(res.InstrumentsPath)
	s.Require().NoError(err)
	var decoded []crawlerdto.RegistryInstrument
	s.Require().NoError(json.Unmarshal(rawSaved, &decoded))
	s.Len(decoded, 27+1634+1)

	// Validate sorting and asset classes
	hasETF := false
	hasStock := false
	hasCall := false
	hasPut := false
	for _, inst := range decoded {
		s.NotEmpty(inst.InsCode)
		s.NotEmpty(inst.Symbol)
		switch inst.AssetClass {
		case "ETF":
			hasETF = true
		case "STOCK":
			hasStock = true
		case "OPTION_CALL":
			hasCall = true
		case "OPTION_PUT":
			hasPut = true
		}
	}
	s.True(hasETF, "should contain ETF instruments like اهرم")
	s.True(hasStock, "should contain STOCK instruments")
	s.True(hasCall, "should contain Call option instruments")
	s.True(hasPut, "should contain Put option instruments")
}

func TestDiscoveryServiceSuite(t *testing.T) {
	suite.Run(t, new(DiscoveryServiceTestSuite))
}

func (s *DiscoveryServiceTestSuite) TestMissingWatchDoesNotExpireFutureContracts() {
	path := filepath.Join(s.T().TempDir(), "watch.json")
	s.Require().NoError(os.WriteFile(path, []byte(`{"instrumentOptMarketWatch":[]}`), 0o644))
	code, base := "123", "456"
	s.optionRepo.On("GetAll", s.db, domainrepo.OptionFilter{}).Return([]*entity.OptionContract{{TSETMCID: &code, SymbolFA: "future", ContractType: "CALL", ExpirationDate: time.Now().AddDate(0, 1, 0), UnderlyingAsset: &entity.UnderlyingAsset{TSETMCID: &base, SymbolFA: "base"}}}, nil)
	res, err := s.service.DiscoverFullRegistry(context.Background(), crawlerdto.FullRegistryOptions{UseOfflineFixture: true, FixturePath: path, RegistryDir: s.T().TempDir(), IncludeExpired: true})
	s.Require().NoError(err)
	s.Equal(1, res.Summary.ActiveOptionsCount)
	s.Zero(res.Summary.ExpiredOptionsCount)
	s.Equal(1, res.Summary.UnderlyingsCount)
}

func (s *DiscoveryServiceTestSuite) TestExpiryDayStillActiveAndExpiredFlagHonored() {
	raw := fmt.Sprintf(`{"instrumentOptMarketWatch":[{"uaInsCode":"123","lval30_UA":"base","endDate":"%s","remainedDay":0,"insCode_C":"456","lVal18AFC_C":"call"}]}`, marketToday())
	path := filepath.Join(s.T().TempDir(), "watch.json")
	s.Require().NoError(os.WriteFile(path, []byte(raw), 0o644))
	res, err := s.service.DiscoverFullRegistry(context.Background(), crawlerdto.FullRegistryOptions{UseOfflineFixture: true, FixturePath: path, RegistryDir: s.T().TempDir()})
	s.Require().NoError(err)
	s.Equal(1, res.Summary.ActiveOptionsCount)
	raw = strings.ReplaceAll(raw, marketToday(), "20200101")
	s.Require().NoError(os.WriteFile(path, []byte(raw), 0o644))
	res, err = s.service.DiscoverFullRegistry(context.Background(), crawlerdto.FullRegistryOptions{UseOfflineFixture: true, FixturePath: path, RegistryDir: s.T().TempDir(), IncludeExpired: false})
	s.Require().NoError(err)
	s.Empty(res.ExpiredContracts)
	s.Empty(res.ActiveContracts)
}

func (s *DiscoveryServiceTestSuite) TestHistoricalDiscoveryPersistsPreviouslyUnknownExpiredContract() {
	dir := s.T().TempDir()
	historyFixture(s.T(), dir, "watch.json", `{"instrumentOptMarketWatch":[]}`)
	historyFixture(s.T(), dir, "20241026.json", `{"closingPriceDailyHistoryWithInstDetails":[{"insCode":"111","lVal18AFC":"قدیمی","lVal30":"سهم قدیمی","cIsin":"IRO1001"},{"insCode":"222","lVal18AFC":"ضقدیم1","lVal30":"اختیارخ قدیمی-1000-2024/10/30","cIsin":"IRO9001"}]}`)
	s.optionRepo.On("GetAll", s.db, domainrepo.OptionFilter{}).Return([]*entity.OptionContract{}, nil).Once()
	s.underlyingRepo.On("GetBySymbol", s.db, "قدیمی").Return((*entity.UnderlyingAsset)(nil), nil).Once()
	s.underlyingRepo.On("Upsert", s.db, mock.Anything).Run(func(a mock.Arguments) { a.Get(1).(*entity.UnderlyingAsset).ID = 77 }).Return(nil).Once()
	baseCode := "111"
	s.underlyingRepo.On("GetByTSETMCID", s.db, "111").Return(&entity.UnderlyingAsset{Model: database.Model{ID: 77}, TSETMCID: &baseCode, SymbolFA: "قدیمی"}, nil).Once()
	s.optionRepo.On("SaveBulk", s.db, mock.MatchedBy(func(rows []*entity.OptionContract) bool {
		return len(rows) == 1 && rows[0].UnderlyingAssetID == 77 && rows[0].TSETMCID != nil && *rows[0].TSETMCID == "222" && rows[0].ISIN != nil && *rows[0].ISIN == "IRO9001" && !rows[0].IsActive && rows[0].Status == "EXPIRED"
	})).Return(1, nil).Once()
	res, err := s.service.DiscoverFullRegistry(context.Background(), crawlerdto.FullRegistryOptions{UseOfflineFixture: true, FixturePath: filepath.Join(dir, "watch.json"), RegistryDir: s.T().TempDir(), HistoryFixtureDir: dir, HistoryDates: []string{"20241026"}, IncludeExpired: true, PersistDB: true})
	s.Require().NoError(err)
	s.Equal(1, res.PersistedDBRows)
	s.Len(res.ExpiredContracts, 1)
	s.underlyingRepo.AssertExpectations(s.T())
	s.optionRepo.AssertExpectations(s.T())
}
