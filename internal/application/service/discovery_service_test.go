package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/domain/entity"
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

func (s *DiscoveryServiceTestSuite) TestLiveFallbackAndPersistDB() {
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

	s.client.On("Get", mock.Anything, TSETMCOptionWatchURL).
		Return([]byte(nil), http.StatusServiceUnavailable, errors.New("upstream down")).Once()

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
		UseOfflineFixture: false,
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

func TestDiscoveryServiceSuite(t *testing.T) {
	suite.Run(t, new(DiscoveryServiceTestSuite))
}
