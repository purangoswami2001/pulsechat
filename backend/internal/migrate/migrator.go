package migrate

import (
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/postgres/*.sql
var postgresFS embed.FS

type Migrator struct {
	driver string
	dsn    string
}

func NewMigrator(driver, dsn string) *Migrator {
	return &Migrator{
		driver: driver,
		dsn:    dsn,
	}
}

func (m *Migrator) RunAutoMigrate() error {
	slog.Info("running auto migrations...")
	return m.Up()
}

func (m *Migrator) Up() error {
	mig, err := m.getMigrateInstance()
	if err != nil {
		return err
	}
	defer mig.Close()

	if err := mig.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("database schema is up to date (no changes)")
			return nil
		}
		return err
	}
	slog.Info("applied all migrations successfully")
	return nil
}

func (m *Migrator) Down() error {
	mig, err := m.getMigrateInstance()
	if err != nil {
		return err
	}
	defer mig.Close()

	if err := mig.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("no changes to rollback")
			return nil
		}
		return err
	}
	slog.Info("rolled back all migrations successfully")
	return nil
}

func (m *Migrator) Steps(n int) error {
	mig, err := m.getMigrateInstance()
	if err != nil {
		return err
	}
	defer mig.Close()

	if err := mig.Steps(n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			slog.Info("no migration changes applied")
			return nil
		}
		return err
	}
	slog.Info("migration steps run successfully", "n", n)
	return nil
}

func (m *Migrator) Version() (uint, bool, error) {
	mig, err := m.getMigrateInstance()
	if err != nil {
		return 0, false, err
	}
	defer mig.Close()

	v, dirty, err := mig.Version()
	if err != nil {
		return 0, false, err
	}
	return v, dirty, nil
}

func (m *Migrator) Force(v int) error {
	mig, err := m.getMigrateInstance()
	if err != nil {
		return err
	}
	defer mig.Close()

	return mig.Force(v)
}

func (m *Migrator) getMigrateInstance() (*migrate.Migrate, error) {
	var sourceDriver source.Driver
	var err error

	switch m.driver {
	case "postgres":
		sourceDriver, err = iofs.New(postgresFS, "migrations/postgres")
		if err != nil {
			return nil, fmt.Errorf("failed to load embedded postgres migrations: %w", err)
		}
	case "mysql":
		return nil, fmt.Errorf("mysql migrations are not implemented")
	default:
		return nil, fmt.Errorf("unsupported migration driver: %s", m.driver)
	}

	mig, err := migrate.NewWithSourceInstance("iofs", sourceDriver, m.dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to init migration instance: %w", err)
	}
	return mig, nil
}
