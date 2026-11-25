package database

import (
	"context"
	"fmt"
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

func NewPostgresDatabase(cfg *bootstrap.Database) (*PostgresDatabase, error) {
	if cfg == nil {
		return nil, fmt.Errorf("database config is required")
	}
	level := gormlogger.Silent
	if cfg.Echo {
		level = gormlogger.Info
	}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{Logger: gormlogger.Default.LogMode(level), DisableAutomaticPing: true})
	if err != nil {
		return nil, err
	}
	pool, err := db.DB()
	if err != nil {
		return nil, err
	}
	maxOpen := cfg.PoolSize + cfg.MaxOverflow
	if maxOpen <= 0 {
		maxOpen = 10
	}
	maxIdle := cfg.PoolSize
	if maxIdle <= 0 {
		maxIdle = 5
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}
	pool.SetMaxOpenConns(maxOpen)
	pool.SetMaxIdleConns(maxIdle)
	pool.SetConnMaxLifetime(time.Hour)
	for attempt := 0; attempt < 10; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err = pool.PingContext(ctx)
		cancel()
		if err == nil {
			return &PostgresDatabase{DB: db}, nil
		}
		if attempt < 9 {
			time.Sleep(2 * time.Second)
		}
	}
	_ = pool.Close()
	return nil, fmt.Errorf("failed to connect database: %w", err)
}

func (pg *PostgresDatabase) GetDB() *gorm.DB {
	if pg == nil {
		return nil
	}
	return pg.DB
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
