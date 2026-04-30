package main

import (
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/rozoomcool/sihkaromicro/sources/migrations"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN is required")
	}

	source, err := iofs.New(migrations.FS, "sql")
	if err != nil {
		log.Fatal("failed to create source:", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", source, dsn)
	if err != nil {
		log.Fatal("failed to create migrator:", err)
	}
	defer m.Close()

	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatal("failed to migrate up:", err)
		}
		fmt.Println("✅ migrations applied successfully")

	case "down":
		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
			log.Fatal("failed to migrate down:", err)
		}
		fmt.Println("✅ migration rolled back")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatal("failed to get version:", err)
		}
		fmt.Printf("version: %d, dirty: %v\n", version, dirty)

	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}
