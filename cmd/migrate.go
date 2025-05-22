package cmd

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	migrateCmd = &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
		Long:  `Create and run database migrations.`,
	}

	migrateUpCmd = &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations("up")
		},
	}

	migrateDownCmd = &cobra.Command{
		Use:   "down",
		Short: "Rollback the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrations("down")
		},
	}

	migrateCreateCmd = &cobra.Command{
		Use:   "create [name]",
		Short: "Create a new migration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return createMigration(args[0])
		},
	}
)

func init() {
	rootCmd.AddCommand(migrateCmd)
	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateCreateCmd)
}

func runMigrations(direction string) error {
	cfg := GetConfig()

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			logger.Error("Failed to sync logger", zap.Error(err))
		}
	}()

	db, err := connectPostgres(cfg.Postgres)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", zap.Error(err))
		}
	}()

	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	files, err := getMigrationFiles()
	if err != nil {
		return fmt.Errorf("get migration files: %w", err)
	}

	applied, err := getAppliedMigrations(db, logger)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	if direction == "up" {
		for _, file := range files {
			if !applied[filepath.Base(file)] {
				if err := runMigration(db, file, "up", logger); err != nil {
					return fmt.Errorf("run migration %s: %w", file, err)
				}
			}
		}
	} else {
		if len(applied) == 0 {
			fmt.Println("No migrations to rollback")
			return nil
		}

		var lastMigration string
		for _, file := range files {
			if applied[file] {
				lastMigration = file
			}
		}

		if err := runMigration(db, lastMigration, "down", logger); err != nil {
			return fmt.Errorf("rollback migration %s: %w", lastMigration, err)
		}
	}

	return nil
}

func createMigration(name string) error {
	if err := os.MkdirAll("migrations", 0755); err != nil {
		return fmt.Errorf("create migrations directory: %w", err)
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.sql", timestamp, strings.ToLower(name))
	path := filepath.Join("migrations", filename)

	content := fmt.Sprintf(`-- Migration: %s
-- Created at: %s

-- Up Migration
CREATE TABLE IF NOT EXISTS example (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Down Migration
DROP TABLE IF EXISTS example;
`, name, time.Now().Format(time.RFC3339))

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write migration file: %w", err)
	}

	fmt.Printf("Created migration: %s\n", path)
	return nil
}

func createMigrationsTable(db *sql.DB) error {
	query := `
		CREATE TABLE IF NOT EXISTS migrations (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)`
	_, err := db.Exec(query)
	return err
}

func getMigrationFiles() ([]string, error) {
	files, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func closeRows(rows *sql.Rows, logger *zap.Logger) {
	if err := rows.Close(); err != nil {
		logger.Error("Failed to close rows", zap.Error(err))
	}
}

func getAppliedMigrations(db *sql.DB, logger *zap.Logger) (map[string]bool, error) {
	query := `SELECT name FROM migrations ORDER BY applied_at`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer closeRows(rows, logger)

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}
	return applied, rows.Err()
}

func rollbackTx(tx *sql.Tx, logger *zap.Logger) {
	if err := tx.Rollback(); err != nil {
		logger.Error("Failed to rollback transaction", zap.Error(err))
	}
}

func runMigration(db *sql.DB, filename string, direction string, logger *zap.Logger) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("read migration file: %w", err)
	}

	parts := strings.Split(string(content), "-- Down Migration")
	if len(parts) != 2 {
		return fmt.Errorf("invalid migration file format")
	}

	upMigration := strings.TrimPrefix(parts[0], "-- Up Migration\n")
	downMigration := strings.TrimSpace(parts[1])

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer rollbackTx(tx, logger)

	var migrationSQL string
	if direction == "up" {
		migrationSQL = upMigration
		_, err = tx.Exec("INSERT INTO migrations (name) VALUES ($1)", filepath.Base(filename))
	} else {
		migrationSQL = downMigration
		_, err = tx.Exec("DELETE FROM migrations WHERE name = $1", filepath.Base(filename))
	}
	if err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	if _, err := tx.Exec(migrationSQL); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	fmt.Printf("Executed %s migration: %s\n", direction, filename)
	return nil
}
