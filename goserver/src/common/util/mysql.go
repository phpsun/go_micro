package util

import (
	"common/tlog"
	"database/sql"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"time"
)

type MysqlConfig struct {
	Addr         string `toml:"addr"`
	User         string `toml:"user"`
	Pwd          string `toml:"pwd"`
	Dbname       string `toml:"dbname"`
	MaxOpenConns int    `toml:"max_open_conns"`
	MaxIdleConns int    `toml:"max_idle_conns"`
}

var ErrorInsertDuplicate = errors.New("insert duplicated")

func NewMysql(c *MysqlConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", formatDataSource(c.Addr, c.User, c.Pwd, c.Dbname))
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(c.MaxOpenConns)
	db.SetMaxIdleConns(c.MaxIdleConns)
	db.SetConnMaxLifetime(7 * time.Hour)
	return db, nil
}

func MysqlSelect(db *sql.DB, query string, maxRows int, fnScan func(*sql.Rows) error, args ...interface{}) (int, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		tlog.Error(err)
		return 0, err
	}

	count := 0
	for rows.Next() {
		err = fnScan(rows)
		if err == nil {
			count++
		} else {
			tlog.Error(err)
		}
		if count >= maxRows {
			break
		}
	}
	err = rows.Err()
	if err != nil {
		tlog.Error(err)
	}

	rows.Close()
	return count, err
}

func TxSelect(tx *sql.Tx, query string, maxRows int, fnScan func(*sql.Rows) error, args ...interface{}) (int, error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		tlog.Error(err)
		return 0, err
	}

	count := 0
	for rows.Next() {
		err = fnScan(rows)
		if err == nil {
			count++
		} else {
			tlog.Error(err)
		}
		if count >= maxRows {
			break
		}
	}
	err = rows.Err()
	if err != nil {
		tlog.Error(err)
	}

	rows.Close()
	return count, err
}

func MysqlExec(db *sql.DB, query string, args ...interface{}) (int, error) {
	result, err := db.Exec(query, args...)
	if err == nil {
		var affectRows int64
		affectRows, err = result.RowsAffected()
		if err == nil {
			return int(affectRows), nil
		}
	}
	tlog.Error(err)
	return 0, err
}

func TxExec(tx *sql.Tx, query string, args ...interface{}) (int, error) {
	result, err := tx.Exec(query, args...)
	if err == nil {
		var affectRows int64
		affectRows, err = result.RowsAffected()
		if err == nil {
			return int(affectRows), nil
		}
	}
	tlog.Error(err)
	return 0, err
}

func MysqlExecAndGetInsertId(db *sql.DB, query string, args ...interface{}) (int64, error) {
	result, err := db.Exec(query, args...)
	if err == nil {
		var lastId int64
		lastId, err = result.LastInsertId()
		if err != nil {
			tlog.Error(err)
		}
		return lastId, err
	}

	tlog.Error(err)
	merr, ok := err.(*mysql.MySQLError)
	if ok && merr.Number == 1062 { // Duplicate
		return 0, ErrorInsertDuplicate
	}
	return 0, err
}

func TxExecAndGetInsertId(tx *sql.Tx, query string, args ...interface{}) (int64, error) {
	result, err := tx.Exec(query, args...)
	if err == nil {
		var lastId int64
		lastId, err = result.LastInsertId()
		if err != nil {
			tlog.Error(err)
		}
		return lastId, err
	}

	tlog.Error(err)
	merr, ok := err.(*mysql.MySQLError)
	if ok && merr.Number == 1062 { // Duplicate
		return 0, ErrorInsertDuplicate
	}
	return 0, err
}

func formatDataSource(addr string, user string, pwd string, dbname string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&timeout=2s", user, pwd, addr, dbname)
}
