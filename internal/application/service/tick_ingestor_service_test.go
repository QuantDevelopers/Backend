package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantDevelopers/Backend/bootstrap"
	crawlerdto "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	infralogger "github.com/QuantDevelopers/Backend/internal/infrastructure/logger"
	tsetmcMocks "github.com/QuantDevelopers/Backend/mocks/infrastructure/tsetmc"
	"github.com/stretchr/testify/suite"
)

type TickIngestorServiceTestSuite struct {
	suite.Suite
	client  *tsetmcMocks.TSETMCClientMock
	service *TickIngestorService
}

func (s *TickIngestorServiceTestSuite) SetupTest() {
	cfg := bootstrap.Run()
	log, err := infralogger.NewLogger(&bootstrap.Logger{
		LogLevel:      "error",
		ConsoleOutput: "false",
	}, cfg.Constants)
	s.Require().NoError(err)

	s.client = tsetmcMocks.NewTSETMCClientMock()
	s.service = NewTickIngestorService(s.client, log)
}

func (s *TickIngestorServiceTestSuite) TestFormatHEvenAndComputeSHA256() {
	s.Equal("00:00:00", FormatHEven(0))
	s.Equal("09:00:05", FormatHEven(90005))
	s.Equal("10:38:06", FormatHEven(103806))
	s.Equal("12:30:07", FormatHEven(123007))

	s.Equal(
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		ComputeSHA256(nil),
	)
}

func (s *TickIngestorServiceTestSuite) TestIngestSampleOfflineFixture() {
	fixturePath := filepath.Join("..", "..", "..", "tests", "fixtures", "sample_trade_history.json")
	dataDir := s.T().TempDir()

	res, err := s.service.IngestSample(context.Background(), crawlerdto.TickIngestOptions{
		InsCode:           "42511156565416187",
		DateStr:           "20261028",
		ContractSize:      1000,
		UseOfflineFixture: true,
		FixturePath:       fixturePath,
		DataDir:           dataDir,
	})
	s.Require().NoError(err)

	s.Equal("ok", res.Manifest.Status)
	s.Equal(10, res.Manifest.RecordsCount)
	s.Equal("tsetmc_cdn_trade_api", res.Manifest.Source)
	s.FileExists(res.RawPath)
	s.FileExists(res.ManifestPath)
	s.FileExists(res.CleanPath)

	rawExpected, err := os.ReadFile(fixturePath)
	s.Require().NoError(err)
	rawSaved, err := os.ReadFile(res.RawPath)
	s.Require().NoError(err)
	s.Equal(rawExpected, rawSaved)
	s.Equal(ComputeSHA256(rawExpected), res.Manifest.SHA256)
	s.Equal(len(rawExpected), res.Manifest.RawSizeBytes)

	// Chronological order by nTran
	s.Require().Len(res.Records, 10)
	s.Equal(1, res.Records[0].NTran)
	s.Equal("10:38:06", res.Records[0].Time)
	s.InDelta(27002000.0, res.Records[0].NotionalValue, 0.001)
	s.False(res.Records[0].IsCanceled)

	s.Equal(11, res.Records[9].NTran)
	s.Equal("12:30:07", res.Records[9].Time)
	s.InDelta(62000000.0, res.Records[9].NotionalValue, 0.001)

	manifestBytes, err := os.ReadFile(res.ManifestPath)
	s.Require().NoError(err)
	var diskManifest crawlerdto.TickManifest
	s.Require().NoError(json.Unmarshal(manifestBytes, &diskManifest))
	s.Equal(res.Manifest.SHA256, diskManifest.SHA256)
	s.Equal(10, diskManifest.RecordsCount)
}

func (s *TickIngestorServiceTestSuite) TestCanceledFlagAndEmptyStatus() {
	payload := `{
		"tradeHistory": [
			{"nTran": 2, "hEven": 90501, "pTran": 1500, "qTitTran": 4, "canceled": 1},
			{"nTran": 1, "hEven": 90005, "pTran": 1400, "qTitTran": 2, "canceled": 0}
		]
	}`
	records, err := s.service.ParseTickPayload([]byte(payload), "999", "20261028", 1000)
	s.Require().NoError(err)
	s.Require().Len(records, 2)
	s.Equal(1, records[0].NTran)
	s.Equal("09:00:05", records[0].Time)
	s.False(records[0].IsCanceled)
	s.Equal(2, records[1].NTran)
	s.Equal("09:05:01", records[1].Time)
	s.True(records[1].IsCanceled)

	emptyFixture := filepath.Join(s.T().TempDir(), "empty.json")
	s.Require().NoError(os.WriteFile(emptyFixture, []byte(`{"trade":[]}`), 0o644))

	res, err := s.service.IngestSample(context.Background(), crawlerdto.TickIngestOptions{
		InsCode:           "999",
		DateStr:           "20261028",
		UseOfflineFixture: true,
		FixturePath:       emptyFixture,
		DataDir:           s.T().TempDir(),
	})
	s.Require().NoError(err)
	s.Equal("empty", res.Manifest.Status)
	s.Equal(0, res.Manifest.RecordsCount)
}

func TestTickIngestorServiceSuite(t *testing.T) {
	suite.Run(t, new(TickIngestorServiceTestSuite))
}
