package util

import (
	"common/defs"
	"common/tlog"
	"database/sql"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"strings"
	"sync"
)

type MysqlMgr struct {
	mutex        sync.RWMutex
	dbs          map[string]*sql.DB
	gorm         map[string]*gorm.DB
	user         string
	pwd          string
	maxOpenConns int
	maxIdleConns int
}

// dbUser format: user:password
func NewGormDB(sqlDB *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn: sqlDB,
	}), &gorm.Config{})
	return gormDB, err
}

// dbUser format: user:password
func NewMysqlMgr(dbUser string, env string) *MysqlMgr {
	maxOpenConns := 10
	maxIdleConns := 5
	if env == defs.EnvProd {
		maxOpenConns = 30
		maxIdleConns = 10
	}

	var user string
	var pwd string
	idx := strings.Index(dbUser, ":")
	if idx > 0 {
		user = dbUser[:idx]
		pwd = dbUser[idx+1:]
	} else {
		user = dbUser
	}

	dbs := make(map[string]*sql.DB)
	gorm := make(map[string]*gorm.DB)
	return &MysqlMgr{dbs: dbs, gorm: gorm, user: user, pwd: pwd, maxOpenConns: maxOpenConns, maxIdleConns: maxIdleConns}
}

// dbMark format: dbAddr/dbName
func (this *MysqlMgr) FetchDatabase(dbMark string) (*sql.DB, error) {
	this.mutex.RLock()
	if db, ok := this.dbs[dbMark]; ok {
		this.mutex.RUnlock()
		return db, nil
	}
	this.mutex.RUnlock()

	var addr string
	var dbname string
	idx := strings.Index(dbMark, "/")
	if idx > 0 {
		addr = dbMark[:idx]
		dbname = dbMark[idx+1:]
	} else {
		return nil, fmt.Errorf("unknown dbmark: %s", dbMark)
	}

	conf := MysqlConfig{
		Addr:         addr,
		User:         this.user,
		Pwd:          this.pwd,
		Dbname:       dbname,
		MaxOpenConns: this.maxOpenConns,
		MaxIdleConns: this.maxIdleConns,
	}
	db, err := NewMysql(&conf)
	if err != nil {
		return nil, err
	}

	this.mutex.Lock()
	this.dbs[dbMark] = db
	this.mutex.Unlock()

	return db, nil
}

//根据请求域名，初始化数据库
func (this *MysqlMgr) DbInitBydomain(domain string) (*sql.DB, error) {
	//获取用户dbMark format: dbAddr/dbName
	dbMark := "10.7.9.61/oshop-system" // Todo 通过数据库获取
	db, err := this.FetchDatabase(dbMark)

	if err != nil {
		return nil, err
	}

	return db, nil
}

//根据请求域名，获取gorm实例
func (this *MysqlMgr) GormInitByDomain(domain string) (*gorm.DB, error) {
	mysqlDb, err := this.DbInitBydomain(domain)
	if err != nil {
		tlog.Error("failed to connect database")
		return nil, err
	}

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn: mysqlDb,
	}), &gorm.Config{})
	if err != nil {
		tlog.Error("gorm init failed")
		return nil, err
	}
	return db, nil
}

////根据请求域名，获取gorm实例--废弃
//func (this *MysqlMgr) SystemGormInit(dbMark string) (*gorm.DB,error) {
//	mysqlDb,err := this.FetchDatabase(dbMark)
//	if err != nil {
//		tlog.Error("failed to connect database")
//		return nil,err
//	}
//
//	db, err := gorm.Open(mysql.New(mysql.Config{
//		Conn:mysqlDb ,
//	}), &gorm.Config{})
//	if err != nil {
//		tlog.Error("gorm init failed")
//		return nil,err
//	}
//	return db,nil
//}
//根据库地址，获取gorm实例
func (this *MysqlMgr) GormInitByDbmark(dbMark string) (*gorm.DB, error) {
	mysqlDb, err := this.FetchDatabase(dbMark)
	if err != nil {
		tlog.Error("failed to connect database")
		return nil, err
	}
	this.mutex.RLock()
	if db, ok := this.gorm[dbMark]; ok {
		this.mutex.RUnlock()
		return db, nil
	}
	this.mutex.RUnlock()

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn: mysqlDb,
	}), &gorm.Config{})
	if err != nil {
		tlog.Error("gorm init failed")
		return nil, err
	}

	this.mutex.Lock()
	this.gorm[dbMark] = db
	this.mutex.Unlock()
	return db, nil
}
