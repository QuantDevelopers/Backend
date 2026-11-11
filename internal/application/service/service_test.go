package service

import (
	"context"
	"testing"

	"github.com/QuantDevelopers/Backend/internal/domain/entity"
	"github.com/QuantDevelopers/Backend/internal/domain/repository"
)

type stubUnderlyingRepo struct {
	seedCalled bool
	assets     []entity.UnderlyingAsset
	bySym      map[string]*entity.UnderlyingAsset
}

func (s *stubUnderlyingRepo) SeedDefaults(_ context.Context) (int, error) {
	s.seedCalled = true
	return len(s.assets), nil
}

func (s *stubUnderlyingRepo) GetAll(_ context.Context) ([]entity.UnderlyingAsset, error) {
	return s.assets, nil
}

func (s *stubUnderlyingRepo) GetBySymbol(_ context.Context, symbolFA string) (*entity.UnderlyingAsset, error) {
	return s.bySym[symbolFA], nil
}

type stubOptionRepo struct {
	contracts []entity.OptionContract
	saved     *entity.OptionContract
}

func (s *stubOptionRepo) GetAll(_ context.Context, _ repository.OptionFilter) ([]entity.OptionContract, error) {
	return s.contracts, nil
}

func (s *stubOptionRepo) GetBySymbol(_ context.Context, symbolFA string) (*entity.OptionContract, error) {
	for i := range s.contracts {
		if s.contracts[i].SymbolFA == symbolFA {
			return &s.contracts[i], nil
		}
	}
	return nil, nil
}

func (s *stubOptionRepo) Save(_ context.Context, contract entity.OptionContract) (*entity.OptionContract, error) {
	s.saved = &contract
	return &contract, nil
}

func (s *stubOptionRepo) SaveBulk(_ context.Context, contracts []entity.OptionContract) (int, error) {
	return len(contracts), nil
}

func TestUnderlyingServiceListAndLookup(t *testing.T) {
	repo := &stubUnderlyingRepo{
		assets: []entity.UnderlyingAsset{{ID: 1, SymbolFA: "اهرم"}},
		bySym:  map[string]*entity.UnderlyingAsset{"اهرم": {ID: 1, SymbolFA: "اهرم"}},
	}
	svc := NewUnderlyingService(repo)

	got, err := svc.ListUnderlyings(context.Background())
	if err != nil {
		t.Fatalf("ListUnderlyings error: %v", err)
	}
	if !repo.seedCalled || len(got) != 1 || got[0].SymbolFA != "اهرم" {
		t.Fatalf("unexpected ListUnderlyings result: seedCalled=%v got=%+v", repo.seedCalled, got)
	}

	one, err := svc.GetUnderlyingBySymbol(context.Background(), "اهرم")
	if err != nil || one == nil || one.SymbolFA != "اهرم" {
		t.Fatalf("unexpected GetUnderlyingBySymbol result: one=%+v err=%v", one, err)
	}
}

func TestOptionServiceCRUD(t *testing.T) {
	repo := &stubOptionRepo{
		contracts: []entity.OptionContract{{ID: 1, SymbolFA: "ضهرم1033", ContractType: "CALL"}},
	}
	svc := NewOptionService(repo)

	list, err := svc.ListOptions(context.Background(), repository.OptionFilter{})
	if err != nil || len(list) != 1 {
		t.Fatalf("ListOptions unexpected: %+v err=%v", list, err)
	}

	found, err := svc.GetBySymbol(context.Background(), "ضهرم1033")
	if err != nil || found == nil || found.SymbolFA != "ضهرم1033" {
		t.Fatalf("GetBySymbol unexpected: %+v err=%v", found, err)
	}

	created, err := svc.CreateOrUpdate(context.Background(), entity.OptionContract{SymbolFA: "طهرم1033"})
	if err != nil || created.SymbolFA != "طهرم1033" || repo.saved == nil {
		t.Fatalf("CreateOrUpdate unexpected: %+v err=%v", created, err)
	}
}
