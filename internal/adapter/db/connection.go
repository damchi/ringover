package db

import (
	"fmt"
	"net"
	"net/url"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"

	"ringover/internal/config"
)

func ConnectDB(conf *config.Config) (*sqlx.DB, error) {
	params := conf.DbParams
	if params == "" {
		params = "parseTime=true&multiStatements=true"
	}

	dsnConfig := mysqlDriver.NewConfig()
	dsnConfig.User = conf.DbUser
	dsnConfig.Passwd = conf.DbPassword
	dsnConfig.Net = "tcp"
	dsnConfig.Addr = net.JoinHostPort(conf.DbHost, conf.DbPort)
	dsnConfig.DBName = conf.DbName

	queryParams, err := url.ParseQuery(params)
	if err != nil {
		return nil, fmt.Errorf("invalid mysql params: %w", err)
	}
	if len(queryParams) > 0 {
		dsnConfig.Params = make(map[string]string, len(queryParams))
		for key, values := range queryParams {
			if len(values) == 0 {
				continue
			}
			dsnConfig.Params[key] = values[len(values)-1]
		}
	}

	db, err := sqlx.Connect("mysql", dsnConfig.FormatDSN())
	if err != nil {
		return nil, err
	}

	return db, nil
}
