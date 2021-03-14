package huobi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cobra"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var WatchCmd = &cobra.Command{
	Use: "watch",
	Run: func(cmd *cobra.Command, args []string) {
		myPrice, _ := cmd.Flags().GetFloat64("price")
		durMinute, _ := cmd.Flags().GetInt("minute")
		watchSymbol, _ := cmd.Flags().GetString("symbol")
		ListPrice(ListPriceReq{
			Symbol:     watchSymbol,
			HasMyPrice: myPrice != 0,
			MyPrice:    myPrice,
			DurMinute:  durMinute,
		})
	},
}

func init() {
	WatchCmd.Flags().IntP("minute", "m", 10, "watch duration in minute")
	WatchCmd.Flags().Float64P("price", "p", 0, "my price")
	WatchCmd.Flags().StringP("symbol", "s", "", "watch symbol")
}

type ListPriceReq struct {
	Symbol     string
	HasMyPrice bool
	MyPrice    float64
	DurMinute  int
}

func IsInWatchList(symbol string) bool {
	for _, watchSymbol := range watchSymbolList {
		if watchSymbol == symbol {
			return true
		}
	}
	return false
}

func ListPrice(req ListPriceReq) {
	if !IsInWatchList(req.Symbol) {
		fmt.Println(strconv.Quote(req.Symbol), "is not in watch symbol list: ", strings.Join(watchSymbolList, " , "))
		os.Exit(-1)
	}
	if req.DurMinute <= 0 {
		fmt.Println("invalid durMinute", req.DurMinute)
		os.Exit(-1)
	}
	for {
		listPriceL1(req)
		time.Sleep(time.Second * 5)
	}
}

func listPriceL1(req ListPriceReq) {
	TradeList := getTradeList(req.Symbol)
	fmt.Println("watch symbol", req.Symbol, len(TradeList))
	if len(TradeList) == 0 {
		return
	}
	if req.DurMinute > 0 && len(TradeList) > req.DurMinute {
		TradeList = TradeList[:req.DurMinute]
	}
	minPrice, maxPrice := getMinPriceMaxPrice(TradeList)

	for idx := len(TradeList); idx > 0; idx-- {
		one := TradeList[idx-1]
		isMinOrMax := ``
		if one.TimeTruncate == minPrice.TimeTruncate {
			isMinOrMax = `-> min`
		} else if one.TimeTruncate == maxPrice.TimeTruncate {
			isMinOrMax = `-> max`
		}
		cur := `    `
		if idx == 1 {
			cur = `[cur]`
		}
		fmt.Printf("%s %12.6f %s %s\n", one.TimeTruncate, one.AvgPrice, cur, isMinOrMax)
	}
	fmt.Printf("%12.6f -> %12.6f , rate %v\n", minPrice.AvgPrice, maxPrice.AvgPrice, getRateString(minPrice.AvgPrice, maxPrice.AvgPrice))
	fmt.Printf("buy  %12.6f -> %12.6f, rate %v\n", TradeList[0].AvgPrice, maxPrice.AvgPrice, getRateString(TradeList[0].AvgPrice, maxPrice.AvgPrice))
	fmt.Printf("sale %12.6f -> %12.6f, rate %v\n", minPrice.AvgPrice, TradeList[0].AvgPrice, getRateString(minPrice.AvgPrice, TradeList[0].AvgPrice))
	if req.HasMyPrice {
		fmt.Printf("my sale cur %12.6f -> %12.6f, rate %v\n", req.MyPrice, TradeList[0].AvgPrice, getRateString(req.MyPrice, TradeList[0].AvgPrice))
		fmt.Printf("my sale max %12.6f -> %12.6f, rate %v\n", req.MyPrice, maxPrice.AvgPrice, getRateString(req.MyPrice, maxPrice.AvgPrice))
	}
	fmt.Println("===========================================================================")
}

var gTimeZone *time.Location
var gTimeZoneOnce sync.Once

func timeToString(t time.Time) string {
	gTimeZoneOnce.Do(func() {
		gTimeZone = time.FixedZone("UTF+8", int((time.Hour * 8).Seconds()))
	})
	return t.In(gTimeZone).Format("2006-01-02 15:04:05")
}

type SimpleTradeForAnalysisV2 struct {
	TimeTruncate  string // per Minute
	LastWriteTime time.Time
	TradeMap      map[int64]SingleTrade //
	AvgPrice      float64
}

type SingleTrade struct {
	Id     int64
	Price  float64
	Amount float64
}

func rangeTradeList(symbol string, cb func(key string, data SimpleTradeForAnalysisV2)) {
	keyList, err := GetRedisStore().HKeys(context.Background(), symbol).Result()
	if err != nil {
		panic(err)
	}
	for _, key := range keyList {
		value, err := GetRedisStore().HGet(context.Background(), symbol, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue // deleted
			}
			panic(err)
		}
		var data SimpleTradeForAnalysisV2
		err = json.Unmarshal([]byte(value), &data)
		if err != nil {
			panic(err)
		}
		cb(key, data)
	}
}

func getTradeList(symbol string) []SimpleTradeForAnalysisV2 {
	var TradeList []SimpleTradeForAnalysisV2
	rangeTradeList(symbol, func(key string, data SimpleTradeForAnalysisV2) {
		var money float64
		var amount float64
		for _, one := range data.TradeMap {
			money += one.Price * one.Amount
			amount += one.Amount
		}
		data.AvgPrice = money / amount
		TradeList = append(TradeList, data)
	})
	sort.Slice(TradeList, func(i, j int) bool {
		return TradeList[i].TimeTruncate > TradeList[j].TimeTruncate // 反向排序
	})
	return TradeList
}

func getRateString(minPrice float64, maxPrice float64) string {
	return strconv.FormatFloat((maxPrice-minPrice)/minPrice*100, 'f', 2, 64) + "%"
}

func getRateFloat(minPrice float64, maxPrice float64) float64 {
	return (maxPrice - minPrice) / minPrice
}
