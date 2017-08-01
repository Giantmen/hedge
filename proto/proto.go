package proto

import "github.com/Giantmen/trader/proto"

const (
	CNY = "cny"
)

func Earn(sell, fsell, buy, fbuy float64) float64 {
	return sell*(1-fsell) - buy*(1+fbuy)
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
