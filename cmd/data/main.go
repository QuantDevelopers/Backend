// Command data provides E3-2 bulk loading/partitioning and E4-2 clean-data processing.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/QuantDevelopers/Backend/bootstrap"
	domain "github.com/QuantDevelopers/Backend/internal/domain/marketdata"
	store "github.com/QuantDevelopers/Backend/internal/infrastructure/marketdata"
	"github.com/jackc/pgx/v5"
)

func run() error {
	mode := flag.String("mode", "process", "partition | load-raw | process")
	from := flag.String("from", "", "inclusive trading date YYYY-MM-DD")
	to := flag.String("to", "", "exclusive trading date YYYY-MM-DD")
	input := flag.String("input", "-", "NDJSON raw tick file, or - for stdin")
	maxAge := flag.Duration("max-age", 0, "maximum same-day underlying tick age; 0 = unlimited")
	timeout := flag.Duration("timeout", 30*time.Minute, "operation timeout")
	flag.Parse()
	if *mode != "partition" && *mode != "load-raw" && *mode != "process" {
		return fmt.Errorf("unknown mode %q", *mode)
	}
	var window domain.Window
	if *mode != "load-raw" {
		var err error
		window.From, err = time.Parse("2006-01-02", *from)
		if err != nil {
			return fmt.Errorf("from: %w", err)
		}
		window.To, err = time.Parse("2006-01-02", *to)
		if err != nil {
			return fmt.Errorf("to: %w", err)
		}
		if err = window.Validate(); err != nil {
			return err
		}
	}
	cfg := bootstrap.Run()
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(signalCtx, *timeout)
	defer cancel()
	conn, err := pgx.Connect(ctx, cfg.Env.PrimaryDB.DSN())
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close(context.Background()) }()
	s := store.Store{Conn: conn}
	switch *mode {
	case "partition":
		if err = s.Partition(ctx, window); err != nil {
			return err
		}
		fmt.Println("monthly tick partitions ready; original tables retained as *_unpartitioned backups")
	case "load-raw":
		f := os.Stdin
		if *input != "-" {
			f, err = os.Open(*input)
			if err != nil {
				return err
			}
			defer func() { _ = f.Close() }()
		}
		count, e := s.ImportRaw(ctx, f)
		if e != nil {
			return e
		}
		fmt.Printf("raw rows loaded: %d\n", count)
	case "process":
		result, e := s.Process(ctx, window, *maxAge)
		if e != nil {
			return e
		}
		return json.NewEncoder(os.Stdout).Encode(result)
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
