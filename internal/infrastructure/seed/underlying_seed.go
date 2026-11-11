package seed

import (
	"fmt"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	repository "github.com/QuantDevelopers/Backend/internal/domain/repository/postgres"
	"github.com/QuantDevelopers/Backend/internal/infrastructure/database"
)

func strPtr(v string) *string {
	return &v
}

var Default25Underlyings = []entity.UnderlyingAsset{
	{SymbolFA: "اهرم", NameFA: "صندوق س. ا. کاریسما-شتاب", TSETMCID: strPtr("17914401175772326"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "خودرو", NameFA: "ایران خودرو", TSETMCID: strPtr("65883838195688438"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "خساپا", NameFA: "سایپا", TSETMCID: strPtr("44891482026867833"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شستا", NameFA: "سرمایه گذاری تامین اجتماعی", TSETMCID: strPtr("2400322364771558"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "فملی", NameFA: "ملی صنایع مس ایران", TSETMCID: strPtr("35425587644337450"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "فولاد", NameFA: "فولاد مبارکه اصفهان", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "طلا", NameFA: "صندوق پشتوانه طلای لوتوس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "عیار", NameFA: "صندوق طلای عیار مفید", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "کهربا", NameFA: "صندوق طلای کهربا", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "زر", NameFA: "صندوق طلای زرین ترنج", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "گوهر", NameFA: "صندوق طلای گوهر نفیس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "زرفام", NameFA: "صندوق طلای زرفام ترنج", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "ناب", NameFA: "صندوق طلای ناب مفید", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "وبملت", NameFA: "بانک ملت", TSETMCID: strPtr("778253364357513"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "وتجارت", NameFA: "بانک تجارت", TSETMCID: strPtr("63917421733088077"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "دی", NameFA: "بانک دی", MarketType: "Farabourse", IsActive: true},
	{SymbolFA: "خگستر", NameFA: "گسترش سرمایه گذاری ایران خودرو", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شتران", NameFA: "پالایش نفت تهران", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شپنا", NameFA: "پالایش نفت اصفهان", TSETMCID: strPtr("7745894403636165"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "شبندر", NameFA: "پالایش نفت بندرعباس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "تاپیکو", NameFA: "س. نفت و گاز و پتروشیمی تامین", TSETMCID: strPtr("22560050433388046"), MarketType: "Bourse", IsActive: true},
	{SymbolFA: "فارس", NameFA: "صنایع پتروشیمی خلیج فارس", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "وغدیر", NameFA: "سرمایه گذاری غدیر", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "کگل", NameFA: "معدنی و صنعتی گل گهر", MarketType: "Bourse", IsActive: true},
	{SymbolFA: "کچاد", NameFA: "معدنی و صنعتی چادرملو", MarketType: "Bourse", IsActive: true},
}

type UnderlyingSeeder struct {
	underlyingRepository repository.UnderlyingRepository
	db                   database.Database
}

func NewUnderlyingSeeder(
	underlyingRepository repository.UnderlyingRepository,
	db database.Database,
) *UnderlyingSeeder {
	return &UnderlyingSeeder{
		underlyingRepository: underlyingRepository,
		db:                   db,
	}
}

func (seeder *UnderlyingSeeder) SeedDefaultUnderlyings() {
	for _, item := range Default25Underlyings {
		existing, err := seeder.underlyingRepository.GetBySymbol(seeder.db, item.SymbolFA)
		if err != nil {
			panic(fmt.Errorf("error checking underlying %s: %w", item.SymbolFA, err))
		}
		if existing != nil {
			if existing.TSETMCID == nil && item.TSETMCID != nil {
				existing.TSETMCID = item.TSETMCID
				if err := seeder.underlyingRepository.Upsert(seeder.db, existing); err != nil {
					panic(fmt.Errorf("error updating tsetmc_id for %s: %w", item.SymbolFA, err))
				}
			}
			continue
		}

		asset := item
		if err := seeder.underlyingRepository.Create(seeder.db, &asset); err != nil {
			panic(fmt.Errorf("error seeding underlying %s: %w", item.SymbolFA, err))
		}
	}
}
