//go:build integration
// +build integration

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationSuiteBase struct {
	suite.Suite

	adminDB    *sqlx.DB
	DB         *sqlx.DB
	testDBName string
}

func (s *IntegrationSuiteBase) SetupSuite() {
	host := envOrDefault("MYSQL_HOST", "127.0.0.1")
	port := envOrDefault("MYSQL_PORT", "3306")
	rootUser := envOrDefault("MYSQL_ROOT_USER", "root")
	rootPassword := envOrDefault("MYSQL_ROOT_PASSWORD", "root")
	database := envOrDefault("MYSQL_TEST_DATABASE", envOrDefault("MYSQL_DATABASE", "ringover")+"_test")
	params := envOrDefault("MYSQL_PARAMS", "parseTime=true&multiStatements=true")

	adminDB, err := sqlx.Connect("mysql", mysqlDSN(rootUser, rootPassword, host, port, "", params))
	if err != nil {
		s.T().Skipf("skipping integration suite: could not connect to mysql: %v", err)
	}
	s.adminDB = adminDB

	_, err = s.adminDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", database))
	s.Require().NoError(err)

	db, err := sqlx.Connect("mysql", mysqlDSN(rootUser, rootPassword, host, port, database, params))
	s.Require().NoError(err)
	s.DB = db
	s.testDBName = database
}

func (s *IntegrationSuiteBase) TearDownSuite() {
	if s.DB != nil {
		s.Require().NoError(s.DB.Close())
	}

	// Drop test database to keep local environment clean after integration runs.
	if s.adminDB != nil && s.testDBName != "" && strings.HasSuffix(s.testDBName, "_test") {
		_, err := s.adminDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", s.testDBName))
		s.Require().NoError(err)
	}

	if s.adminDB != nil {
		s.Require().NoError(s.adminDB.Close())
	}
}

func (s *IntegrationSuiteBase) ResetDatabase() {
	applyTestMigrations(s.T(), s.DB)
}

func applyTestMigrations(t *testing.T, db *sqlx.DB) {
	t.Helper()

	_, err := db.Exec(`
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS categories;
`)
	require.NoError(t, err)

	for _, file := range []string{
		"20260213003947_create_categories_table.up.sql",
		"20260213004222_create_tasks_table.up.sql",
	} {
		content, readErr := os.ReadFile(filepath.Join(projectRoot(t), "db", "migrations", file))
		require.NoError(t, readErr)
		_, execErr := db.Exec(string(content))
		require.NoError(t, execErr)
	}
}

func projectRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", ".."))
}

func mysqlDSN(user, password, host, port, database, params string) string {
	if database == "" {
		return fmt.Sprintf("%s:%s@tcp(%s:%s)/?%s", user, password, host, port, params)
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", user, password, host, port, database, params)
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
