package huobi

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

var flock *FileLock

var CollectMessageCmd = &cobra.Command{
	Use: "collect",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		err = os.MkdirAll("/tmp/huobi", 0777)
		if err != nil {
			panic(err)
		}
		flock, err = NewFileLock("/tmp/huobi/filelock")
		if err != nil {
			panic(err)
		}
		lf, err := os.Create("/tmp/huobi/logfile")
		if err != nil {
			panic(err)
		}
		log.SetOutput(lf)
		log.Println("CollectMessageCmd started, pid ", os.Getpid())

		go threadGc()
		for _, symbol := range watchSymbolList {
			symbol := symbol
			go func() {
				for {
					receiveSingleMessage(symbol)
					time.Sleep(time.Second)
				}
			}()
		}
		select {}
	},
}

func receiveSingleMessage(symbol string) {
	ws, _, err := websocket.DefaultDialer.Dial("wss://api.huobi.pro/ws", nil)
	if err != nil {
		log.Println("receiveSingleMessage dial ", symbol, err)
		return
	}
	defer ws.Close()

	type Trade struct {
		ID        float64 `json:"id"`
		Amount    float64 `json:"amount"`
		Direction string  `json:"direction"`
		Price     float64 `json:"price"`
		TradeID   int64   `json:"tradeId"`
		Ts        int64   `json:"ts"`
	}
	type FullTradeData struct {
		ID     string  `json:"id"`
		Data   []Trade `json:"data"`
		Rep    string  `json:"rep"`
		Status string  `json:"status"`
		Ts     int64   `json:"ts"`
	}

	for {
		_, p, err := ws.ReadMessage()
		if err != nil {
			log.Println("receiveSingleMessage read ", symbol, err)
			return
		}
		res, err := UnGzip(p)
		if err != nil {
			log.Println("receiveSingleMessage UnGzip ", symbol, err)
			return
		}
		resMap := make(map[string]interface{})
		err = json.Unmarshal(res, &resMap)
		if err != nil {
			panic(err)
		}
		if v, ok := resMap["ping"]; ok {
			pingMap := make(map[string]interface{})
			pingMap["pong"] = v
			pingParams, err := json.Marshal(pingMap)
			if err != nil {
				panic(err)
			}
			err = ws.WriteMessage(websocket.TextMessage, pingParams)
			if err != nil {
				log.Println("huobi server ping client error " + err.Error())
				return
			}
			var reqMap struct {
				Req string `json:"req"`
				Id  string `json:"id"`
			}
			reqMap.Id = strconv.Itoa(time.Now().Nanosecond())
			reqMap.Req = "market." + symbol + ".trade.detail" // 交易数据
			reqBytes, err := json.Marshal(reqMap)
			if err != nil {
				panic(err)
			}
			err = ws.WriteMessage(websocket.TextMessage, reqBytes)
			if err != nil {
				log.Println("send req response error " + err.Error())
				return
			}
			continue
		}
		if _, ok := resMap["rep"]; ok {
			_, ok := resMap["id"].(string)
			if !ok {
				log.Println(`43ezc64qkp id is empty `, res)
				continue
			}
			var tmp FullTradeData
			err = json.Unmarshal(res, &tmp)
			if err != nil {
				log.Println("receiveSingleMessage unmarshal json failed", err)
				continue
			}
			var tradeMapByTime = map[string]*SimpleTradeForAnalysisV2{}
			for _, one := range tmp.Data {
				timeS := timeToString(time.Unix(one.Ts/1000/60*60, 0))
				info := tradeMapByTime[timeS]
				if info == nil {
					info = &SimpleTradeForAnalysisV2{
						TimeTruncate: timeS,
						TradeMap:     map[int64]SingleTrade{},
					}
					tradeMapByTime[timeS] = info
				}
				info.TradeMap[one.TradeID] = SingleTrade{
					Id:     one.TradeID,
					Price:  one.Price,
					Amount: one.Amount,
				}
			}
			for key, info := range tradeMapByTime {
				var infoInRedis SimpleTradeForAnalysisV2
				value, err := GetRedisStore().HGet(context.Background(), symbol, key).Result()
				if err != nil && err != redis.Nil {
					panic(err)
				}
				if value != `` {
					err = json.Unmarshal([]byte(value), &infoInRedis)
					if err != nil {
						panic(err)
					}
				}
				for key1, value1 := range infoInRedis.TradeMap {
					info.TradeMap[key1] = value1
				}
				info.LastWriteTime = time.Now()
				filed, err := json.Marshal(info)
				if err != nil {
					panic(err)
				}
				err = GetRedisStore().HSet(context.Background(), symbol, key, filed).Err()
				if err != nil {
					panic(err)
				}
			}
			log.Println("receiveSingleMessage save data ", symbol, len(tmp.Data))
			continue
		}
		log.Println(`4f6ntwatvq unknown res data`, string(res))
	}
}

func UnGzip(byte []byte) (data []byte, err error) {
	b := bytes.NewBuffer(byte)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	data, err = ioutil.ReadAll(r)
	_ = r.Close()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func threadGc() {
	for {
		gcTime := time.Now().Add(-2 * time.Hour)
		for _, symbol := range watchSymbolList {
			var delList []string
			for _, one := range getTradeList(symbol) {
				if one.LastWriteTime.Before(gcTime) {
					delList = append(delList, one.TimeTruncate)
				}
			}
			if len(delList) == 0 {
				continue
			}
			err := GetRedisStore().HDel(context.Background(), symbol, delList...).Err()
			if err != nil {
				panic(err)
			}
			log.Println("gcThread", timeToString(gcTime), "del", symbol, len(delList))
		}
		time.Sleep(time.Hour)
	}
}
