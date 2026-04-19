package database

import (
	"bethapi/config"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

var AsynqInspector *asynq.Inspector
var AsynqClient *asynq.Client
var RedisClient *redis.Client

func InitAsynq() {
	redisConn := asynq.RedisClientOpt{Addr: config.AppConfig.RedisAddr}
	AsynqClient = asynq.NewClient(redisConn)
	AsynqInspector = asynq.NewInspector(redisConn)
	RedisClient = redis.NewClient(&redis.Options{Addr: config.AppConfig.RedisAddr})
}

func GetRedisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: config.AppConfig.RedisAddr}
}
