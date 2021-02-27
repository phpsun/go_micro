package logic

import (
	"common/metrics"
	"common/project"
	"common/proto/base"
	"common/tlog"
	"common/util"
	"context"
	"database/sql"

	"gorm.io/gorm"

	"github.com/go-redis/redis"
)

type Config struct {
	Log     tlog.Config      `toml:"Log"`
	Etcd    []string         `toml:"etcd"`
	Env     string           `toml:"env"`
	EtcdEnv string           `toml:"etcd_env"`
	Db      util.MysqlConfig `toml:"Db"`
	Redis   util.RedisConfig `toml:"Redis"`
}

type Server struct {
	Env        string
	EtcdEnv    string
	Config     *Config
	Mysql      *sql.DB
	GormDB     *gorm.DB
	CacheRedis *redis.Client
	Dingding   *project.RobotDingDing
}

var ThisServer *Server

func NewServer(c *Config) error {
	serverName := "config"
	metrics.Init(serverName, c.Env)

	db, err := util.NewMysql(&c.Db)
	if err != nil {
		tlog.Fatal(err)
		return err
	}

	cacheRedis, err := util.NewRedisClient(&c.Redis)
	if err != nil {
		tlog.Fatal(err)
		return err
	}
	gormDB, err := util.NewGormDB(db)
	if err != nil {
		tlog.Fatal(err)
		return err
	}

	ThisServer = &Server{
		Env:        c.Env,
		EtcdEnv:    c.EtcdEnv,
		Config:     c,
		Mysql:      db,
		CacheRedis: cacheRedis,
		GormDB:     gormDB,
		Dingding:   project.NewRobotDingDing(c.Env, serverName),
	}

	return nil
}

func DestroyServer() {
}

func (ThisServer *Server) Ping(ctx context.Context, req *base.Empty) (*base.Empty, error) {
	return &base.Empty{}, nil
}
