package proto

const (
	Huobi    = "Huobi"
	Huobi_2  = "Huobi2"
	Chbtc    = "Chbtc"
	Yunbi    = "Yunbi"
	Btctrade = "Btctrade"
)

//手续费
const (
	Huobi_btc = 0.002
	Huobi_ltc
	Chbtc_btc
	Chbtc_ltc

	Chbtc_etc = 0.0005
	Huobi_etc //7月13日12:00-7月16日12:00 0.01%
	Yunbi_btc

	Yunbi_etc = 0.001
	Btctrade_etc
)

func Earn(sell, fsell, buy, fbuy float64) float64 {
	return sell*(1-fsell) - buy*(1+fbuy)
}

func ConvertFee(brouse string) float64 {
	switch brouse {
	case "Huobi_btc", "Huobi_ltc", "Chbtc_btc", "Chbtc_ltc":
		return 0.002
	case "Chbtc_etc", "Huobi_etc", "Yunbi_btc":
		return 0.0005
	case "Yunbi_etc", "Btctrade_etc":
		return 0.001
	default:
		return 0
	}
}

type HuiduQuery struct {
	Judge string `validate:"required" json:"judge"`
	Value bool   `json:"value"`
}

type ConfigQuery struct {
	Judge string  `validate:"required" json:"judge"`
	Value float64 `validate:"required" json:"value"`
}

type JudgeQuery struct {
	Judge string `validate:"required" json:"judge"`
}

type ConfigReply struct {
	Ticker    int
	Huidu     bool
	Depth     float64
	Amount    float64
	RightEarn float64
	LeftEarn  float64
}
