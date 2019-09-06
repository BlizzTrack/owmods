package core

import (
	"github.com/blizztrack/owmods/system"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

type RedisClient struct {
	Client   *redis.Client
	Addr     string
	Password string
	Database int
}

var RedisManager *RedisClient

func NewRedis() {
	db, err := strconv.Atoi(system.Redis().Database)
	if err != nil {
		db = 0
	}

	client := redis.NewClient(&redis.Options{
		Addr:     system.Redis().Host,
		Password: system.Redis().Password, // no password set
		DB:       db,                     // use default DB
	})

	_, err = client.Ping().Result()

	if err != nil {
		os.Exit(0)
	}

	RedisManager = &RedisClient{client, client.Options().Addr, client.Options().Password, client.Options().DB}
}

func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return r.Client.Set(key, value, expiration)
}

func (r *RedisClient) Get(key string) (string, error) {
	return r.Client.Get(key).Result()
}

func (r *RedisClient) Keys(pattern string) ([]string, error) {
	return r.Client.Keys(pattern).Result()
}

func (r *RedisClient) Delete(keys ...string) (int64, error) {
	return r.Client.Del(keys...).Result()
}

/*
	Your keys count must match the exist amount or not all keys exist
	Exists in redis doesn't return what keys actually existed so this function is pretty basic
*/
func (r *RedisClient) Exist(keys ...string) bool {
	test, _ := r.Client.Exists(keys...).Result()
	keyCount := len(keys)

	return test == int64(keyCount)
}

func (r *RedisClient) TTL(key string) (time.Duration, error) {
	return r.Client.TTL(key).Result()
}
