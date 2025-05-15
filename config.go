package main

import (
	"context"
	"fmt"
	"github.com/BurntSushi/toml"
	"os"
	"time"

	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	Dsn       string
	Listen    string
	Scriptd   string
	QueueSize int
}

var (
	C       Config
	Verbose bool
	DB      *sql.DB
)

func Init(f string) error {
	r, e := os.Open(f)
	if e != nil {
		return e
	}
	defer r.Close()
	if _, e := toml.DecodeReader(r, &C); e != nil {
		return fmt.Errorf("TOML: %s", e)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	e = DBInit(ctx, "mysql", C.Dsn)
	if e != nil {
		return e
	}
	return nil
}

func DBInit(ctx context.Context, driver string, dsn string) error {
	var e error
	DB, e = sql.Open(driver, dsn)
	if e != nil {
		return e
	}
	// Force MySQL into strict mode
	_, e = DB.ExecContext(ctx, `SET SESSION sql_mode = 'TRADITIONAL,NO_AUTO_VALUE_ON_ZERO,NO_BACKSLASH_ESCAPES'`)


	DB.SetMaxIdleConns(2)
	DB.SetMaxOpenConns(10)
	DB.SetConnMaxLifetime(3 * time.Minute)

	return e
}

func DBClose() error {
	return DB.Close()
}
