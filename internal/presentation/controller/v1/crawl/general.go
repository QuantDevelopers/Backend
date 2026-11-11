package crawl

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/QuantDevelopers/Backend/bootstrap"
	_ "github.com/QuantDevelopers/Backend/internal/application/dto/crawler"
	"github.com/QuantDevelopers/Backend/internal/application/usecase"
	"github.com/QuantDevelopers/Backend/internal/presentation/controller"
	"github.com/labstack/echo/v4"
)

type GeneralCrawlController struct {
	crawlerCfg        *bootstrap.Crawler
	productionCrawler usecase.ProductionCrawlerService
}

func NewGeneralCrawlController(
	crawlerCfg *bootstrap.Crawler,
	productionCrawler usecase.ProductionCrawlerService,
) *GeneralCrawlController {
	return &GeneralCrawlController{
		crawlerCfg:        crawlerCfg,
		productionCrawler: productionCrawler,
	}
}

// GetCrawlStatus returns PostgreSQL ingestion coverage, manifest disk counts, and latest job state.
// @Summary Get crawler coverage and progress status
// @Description Reports underlying/option registry coverage, daily_history/ticks_raw DB row counts, disk manifest statistics, and latest crawl run summary.
// @Tags Crawler
// @Produce json
// @Success 200 {object} controller.SingleMessageResponse{data=crawlerdto.CrawlStatusResponse}
// @Failure 500 {object} controller.SingleMessageResponse
// @Router /api/v1/crawl/status [get]
func (cc *GeneralCrawlController) GetCrawlStatus(ctx echo.Context) error {
	dataDir := "data"
	if cc.crawlerCfg != nil && strings.TrimSpace(cc.crawlerCfg.RawDataPath) != "" {
		clean := filepath.Clean(cc.crawlerCfg.RawDataPath)
		if filepath.Base(clean) == "raw" {
			dataDir = filepath.Dir(clean)
		} else {
			dataDir = clean
		}
	}

	status, err := cc.productionCrawler.GetStatus(dataDir)
	if err != nil {
		panic(err)
	}
	return controller.Response(ctx, http.StatusOK, "", status)
}
