package huobi

import (
	"fmt"
	"github.com/spf13/cobra"
	"sort"
)

var SearchCmd = &cobra.Command{
	Use: "search",
	Run: func(cmd *cobra.Command, args []string) {
		limit, _ := cmd.Flags().GetInt("limit")
		if limit < 0 {
			fmt.Println("invalid limit value", limit)
			return
		}
		type tmp1 struct {
			symbol   string
			rate     float64
			minPrice SimpleTradeForAnalysisV2
			maxPrice SimpleTradeForAnalysisV2
		}
		var tmpList []tmp1
		for _, symbol := range watchSymbolList {
			list := getTradeList(symbol)
			if len(list) == 0 {
				continue
			}
			minPrice, maxPrice := getMinPriceMaxPrice(list)
			tmpList = append(tmpList, tmp1{
				symbol:   symbol,
				rate:     getRateFloat(minPrice.AvgPrice, maxPrice.AvgPrice),
				minPrice: minPrice,
				maxPrice: maxPrice,
			})
		}
		sort.Slice(tmpList, func(i, j int) bool {
			return tmpList[i].rate > tmpList[j].rate
		})
		if 0 < limit && limit < len(tmpList) {
			tmpList = tmpList[:limit]
		}
		for _, one := range tmpList {
			fmt.Printf("%10s %12.6f -> %12.6f rate %s\n", one.symbol, one.minPrice.AvgPrice, one.maxPrice.AvgPrice, getRateString(one.minPrice.AvgPrice, one.maxPrice.AvgPrice))
		}
	},
}

func init() {
	SearchCmd.Flags().IntP(`limit`, `l`, 3, "search limit num")
}

func getMinPriceMaxPrice(list []SimpleTradeForAnalysisV2) (minPrice SimpleTradeForAnalysisV2, maxPrice SimpleTradeForAnalysisV2) {
	minPrice = list[0]
	maxPrice = list[0]
	for _, one := range list {
		if one.AvgPrice < minPrice.AvgPrice {
			minPrice = one
		}
		if one.AvgPrice > maxPrice.AvgPrice {
			maxPrice = one
		}
	}
	return minPrice, maxPrice
}
