package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: migrate [-dir dir] <command>\n")
		fmt.Fprintf(os.Stderr, "commands: up, down, status, version\n")
		fmt.Fprintf(os.Stderr, "requires DATABASE_URL\n")
		flag.PrintDefaults()
	}
	dir := flag.String("dir", "migrations", "migrations directory")
	flag.Parse()
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(2)
	}
	cmd := flag.Arg(0)

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("ping: %v", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	switch cmd {
	case "up":
		err = goose.UpContext(ctx, db, *dir)
	case "down":
		err = goose.DownContext(ctx, db, *dir)
	case "status":
		err = goose.StatusContext(ctx, db, *dir)
	case "version":
		var v int64
		v, err = goose.GetDBVersionContext(ctx, db)
		if err == nil {
			fmt.Println(v)
		}
	default:
		log.Fatalf("unknown command %q", cmd)
	}
	if err != nil {
		log.Fatal(err)
	}
}
