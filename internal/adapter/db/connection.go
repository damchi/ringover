package db

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"ringover/internal/config"
)

func ConnectDB(conf *config.Config) (*sqlx.DB, error) {
	params := conf.DbParams
	if params == "" {
		params = "parseTime=true&multiStatements=true"
	}

	dsn := fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?%s",
		conf.DbUser,
		conf.DbPassword,
		conf.DbHost,
		conf.DbPort,
		conf.DbName,
		params,
	)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}
