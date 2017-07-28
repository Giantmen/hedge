package proto

import (
	"strings"

	"github.com/Giantmen/trader/proto"
)

const (
	CNY = "cny"
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
	case "bter_snt":
		return 0.0016
	case "yunbi_etc", "yunbi_eth", "yunbi_snt", "btctrade_etc", "huobi_etc", "huobi_eth":
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
	case proto.SNT:
		return proto.SNT_CNY
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
