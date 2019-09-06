package system

import (
	"github.com/kataras/iris/sessions/sessiondb/redis"
	"github.com/kataras/iris/sessions/sessiondb/redis/service"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	redisIP       = kingpin.Flag("redis", "Redis Server").Envar("REDIS_HOST").Default("127.0.0.1:6379").String()
	redisPassword = kingpin.Flag("redis_password", "Redis Server Password").Envar("REDIS_PASSWORD").Default("").String()
	redisDatabase = kingpin.Flag("redis_database", "Redis Server Database").Envar("REDIS_DATABASE").Default("2").String()
)

type RedisConfig struct {
	Host     string
	Password string
	Database string

	redis *redis.Database
}

var redisConfig *RedisConfig

func newRedis() *RedisConfig {
	return &RedisConfig{
		Host:     *redisIP,
		Password: *redisPassword,
		Database: *redisDatabase,
	}
}

func Redis() *RedisConfig {
	if redisConfig != nil {
		return redisConfig
	}

	redisConfig = newRedis()
	return redisConfig
}

func (rc *RedisConfig) Client() *redis.Database {
	if rc.redis == nil {
		rc.redis = redis.New(service.Config{
			Network:   "tcp",
			Addr:      rc.Host,
			MaxActive: 10,
			Password:  rc.Password,
			Database:  rc.Database,
			Prefix:    "owmods-",
		})
	}

	return rc.redis
}
