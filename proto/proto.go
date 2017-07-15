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
