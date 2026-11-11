package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type Database interface {
	GetDB() *gorm.DB
	WithTransaction(fn func(Database) error) error
	Ping(ctx context.Context) error
}

type PostgresDatabase struct {
	DB *gorm.DB
}

var (
	dbOnce     sync.Once
	dbInstance *PostgresDatabase
)

func NewPostgresDatabase(dbConfig *bootstrap.Database) (*PostgresDatabase, error) {
	var initErr error
	dbOnce.Do(func() {
		logMode := gormlogger.Silent
		if dbConfig.Echo {
			logMode = gormlogger.Info
		}

		gormCfg := &gorm.Config{
			Logger: gormlogger.Default.LogMode(logMode),
		}

		var db *gorm.DB
		for attempt := 1; attempt <= 10; attempt++ {
			db, initErr = gorm.Open(postgres.Open(dbConfig.DSN()), gormCfg)
			if initErr == nil {
				sqlDB, sqlErr := db.DB()
				if sqlErr == nil {
					if pingErr := sqlDB.Ping(); pingErr == nil {
						maxOpen := dbConfig.PoolSize + dbConfig.MaxOverflow
						if maxOpen <= 0 {
							maxOpen = 10
						}
						maxIdle := dbConfig.PoolSize
						if maxIdle <= 0 {
							maxIdle = 5
						}
						sqlDB.SetMaxOpenConns(maxOpen)
						sqlDB.SetMaxIdleConns(maxIdle)
						sqlDB.SetConnMaxLifetime(time.Hour)
						initErr = nil
						break
					} else {
						initErr = pingErr
					}
				} else {
					initErr = sqlErr
				}
			}
			if attempt < 10 {
				time.Sleep(2 * time.Second)
			}
		}
		if initErr != nil {
			initErr = fmt.Errorf("failed to connect database: %w", initErr)
			return
		}

		dbInstance = &PostgresDatabase{DB: db}
	})

	if initErr != nil {
		return nil, initErr
	}
	return dbInstance, nil
}

func (pg *PostgresDatabase) GetDB() *gorm.DB {
	if pg != nil && pg.DB != nil {
		return pg.DB
	}
	if dbInstance != nil {
		return dbInstance.DB
	}
	return nil
}

func (pg *PostgresDatabase) WithTransaction(fn func(Database) error) error {
	return pg.GetDB().Transaction(func(tx *gorm.DB) error {
		txWrapper := &PostgresDatabase{DB: tx}
		return fn(txWrapper)
	})
}

func (pg *PostgresDatabase) Ping(ctx context.Context) error {
	db := pg.GetDB()
	if db == nil {
		return fmt.Errorf("database instance is not initialized")
	}
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var one int
	return db.WithContext(pingCtx).Raw("SELECT 1").Scan(&one).Error
}
