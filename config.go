package huobi

import (
	"context"
	"github.com/go-redis/redis/v8"
	"sync"
)

const (
	BtcUsdt  = "btcusdt"
	NearUsdt = "nearusdt"
	LtcUsdt  = "ltcusdt"
	DogeUsdt = "dogeusdt"
	EthUsdt  = "ethusdt"
	TopUsdt  = "topusdt"
	SocUsdt  = "socusdt"
)

var watchSymbolList = []string{
	BtcUsdt, NearUsdt, LtcUsdt, DogeUsdt, EthUsdt, TopUsdt, SocUsdt,
}

var gRedisClient *redis.Client
var gRedisClientOnce sync.Once

func GetRedisStore() *redis.Client {
	gRedisClientOnce.Do(func() {
		gRedisClient = redis.NewClient(&redis.Options{
			Network: "tcp",
			Addr:    "127.0.0.1:6379",
		})
		err := gRedisClient.Ping(context.Background()).Err()
		if err != nil {
			panic(err)
		}
	})
	return gRedisClient
}
