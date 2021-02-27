package logic

import (
	"common/discovery"
	"common/metrics"
	"common/proto/config"
	"common/tlog"
	"common/util"
	"database/sql"
	"fmt"
	"github.com/facebookgo/grace/gracehttp"
	"github.com/go-redis/redis"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type Config struct {
	Host     string           `toml:"server_host"`
	ServerId int              `toml:"server_id"`
	Log      tlog.Config      `toml:"Log"`
	Redis    util.RedisConfig `toml:"Redis"`
	Db       util.MysqlConfig `toml:"Db"`
	Etcd     []string         `toml:"etcd"`
	Env      string           `toml:"env"`
	EtcdEnv  string           `toml:"etcd_env"`
}

type Server struct {
	Env           string
	EtcdEnv       string
	CacheRedis    *redis.Client
	Mysql         *sql.DB
	GormDB        *gorm.DB
	Output        *util.HttpOutput
	ConfigGrpc    config.ConfigClient
	JsonMarshaler *util.JsonMarshaler
}

var ThisServer *Server

func NewServer(c *Config) error {
	serverName := "api"
	metrics.Init(serverName, c.Env)

	cacheRedis, err := util.NewRedisClient(&c.Redis)
	if err != nil {
		tlog.Fatal(err)
		return err
	}
	db, err := util.NewMysql(&c.Db)
	if err != nil {
		tlog.Fatal(err)
		return err
	}

	grpcEnv := c.EtcdEnv
	configGrpc := discovery.ResolverConfigServer(grpcEnv)
	gormDB, err := util.NewGormDB(db)
	if err != nil {
		tlog.Fatal(err)
		return err
	}

	ThisServer = &Server{
		Env:           c.Env,
		EtcdEnv:       c.EtcdEnv,
		Mysql:         db,
		GormDB:        gormDB,
		CacheRedis:    cacheRedis,
		Output:        util.NewHttpOutput(),
		JsonMarshaler: util.NewJsonMarshaler(),
		ConfigGrpc:    configGrpc,
	}

	handler := GetHttpHandler()
	fmt.Println(util.FormatFullTime(time.Now()), "running ...")

	return gracehttp.Serve(&http.Server{Addr: c.Host, Handler: handler})
}

func DestroyServer() {
}
