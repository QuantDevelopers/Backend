package postgres

import "gorm.io/gorm"

// MigrateInstrumentIdentity removes the legacy globally-unique ticker indexes.
// An exchange can reuse a symbol for another expiry; tsetmc_id remains unique.
// Run after AutoMigrate, with schema-owner credentials, before serving requests.
func MigrateInstrumentIdentity(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`LOCK TABLE underlying_assets,option_contracts IN SHARE ROW EXCLUSIVE MODE`).Error; err != nil {
			return err
		}
		return tx.Exec(`DROP INDEX IF EXISTS ix_option_contracts_symbol_fa; DROP INDEX IF EXISTS ix_underlying_assets_symbol_fa; DROP INDEX IF EXISTS ix_underlying_assets_isin; DROP INDEX IF EXISTS ix_option_contracts_isin`).Error
	})
}
