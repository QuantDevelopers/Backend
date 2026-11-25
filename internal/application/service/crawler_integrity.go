package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata"
)

var marketLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Tehran")
	if err != nil {
		panic(err)
	}
	return loc
}()

func marketToday() string { return time.Now().In(marketLocation).Format("20060102") }
func validateInsCode(code string) error {
	if len(code) == 0 || len(code) > 20 {
		return fmt.Errorf("invalid insCode %q", code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return fmt.Errorf("invalid insCode %q", code)
		}
	}
	return nil
}
func requireArray(raw []byte, key string) error {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return err
	}
	value, ok := object[key]
	if !ok || !strings.HasPrefix(strings.TrimSpace(string(value)), "[") {
		return fmt.Errorf("upstream response requires array %q", key)
	}
	return nil
}
func writeAtomic(path string, content []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".quant-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if err = f.Chmod(0o644); err == nil {
		_, err = f.Write(content)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, path)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
