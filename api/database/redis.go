package database

import (
	"bethapi/config"
	"github.com/hibiken/asynq"
)

var AsynqInspector *asynq.Inspector
var AsynqClient *asynq.Client

func InitAsynq() {
	redisConn := asynq.RedisClientOpt{Addr: config.AppConfig.RedisAddr}
	AsynqClient = asynq.NewClient(redisConn)
	AsynqInspector = asynq.NewInspector(redisConn)
}

func GetRedisOpt() asynq.RedisClientOpt {
	return asynq.RedisClientOpt{Addr: config.AppConfig.RedisAddr}
}
