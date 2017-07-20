package proto

import (
	"strings"

	"github.com/Giantmen/trader/proto"
)

const (
	Huobi    = "Huobi"
	Huobi_2  = "Huobi2"
	Chbtc    = "Chbtc"
	Yunbi    = "Yunbi"
	Btctrade = "Btctrade"
)

//手续费
const (
	CNY = "cny"

	Huobi_btc = 0.002
	Huobi_ltc
	Chbtc_btc
	Chbtc_ltc

	Yunbi_btc = 0.0005 //7月13日12:00-7月16日12:00 0.01%
	Btctrade_eth

	Chbtc_etc = 0.00046
	Chbtc_eth

	Yunbi_etc = 0.001
	Yunbi_eth
	Btctrade_etc
	Huobi_etc
	Huobi_eth
)

func Earn(sell, fsell, buy, fbuy float64) float64 {
	return sell*(1-fsell) - buy*(1+fbuy)
}

func ConvertFee(brouse string) float64 {
	switch strings.ToLower(brouse) {
	case "huobi_btc", "huobi_ltc", "chbtc_btc", "chbtc_ltc":
		return 0.002
	case "yunbi_btc", "btctrade_eth":
		return 0.0005
	case "chbtc_etc", "chbtc_eth":
		return 0.00046
	case "yunbi_etc", "yunbi_eth", "btctrade_etc", "huobi_etc", "huobi_eth":
		return 0.001
	default:
		return 0
	}
}

func ConvertCurrencyPair(currency string) string {
	switch currency {
	case proto.BTC:
		return proto.BTC_CNY
	case proto.LTC:
		return proto.LTC_CNY
	case proto.ETH:
		return proto.ETH_CNY
	case proto.ETC:
		return proto.ETC_CNY
	case proto.EOS:
		return proto.EOS_CNY
	}
	return ""
}

type HuiduQuery struct {
	Judge string `validate:"required" json:"judge"`
	Value bool   `json:"value"`
}

type ConfigQuery struct {
	Judge string  `validate:"required" json:"judge"`
	Value float64 `validate:"required" json:"value"`
}

type FirstQuery struct {
	Judge string `validate:"required" json:"judge"`
	Value string `validate:"required" json:"value"`
}

type JudgeQuery struct {
	Judge string `validate:"required" json:"judge"`
}

type ConfigReply struct {
	Ticker    int
	First     string
	Huidu     bool
	Depth     float64
	Amount    float64
	RightEarn float64
	LeftEarn  float64
}
