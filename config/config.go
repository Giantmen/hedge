package config

type Server struct {
	Name      string
	Accountid string
	Accesskey string
	Secretkey string
	Timeout   int
}

type Judge struct {
	Name      string
	Bourse    []string
	Ticker    int
	Depth     string
	Amount    string
	Rightearn string
	Leftearn  string
	Huidu     bool
}

type Config struct {
	Listen string

	Debug    bool
	LogPath  string
	LogLevel string

	Chbtc    []Server
	Yunbi    []Server
	HuobiN   []Server
	HuobiO   []Server
	Btctrade []Server
	Bter     []Server
	Poloniex []Server

	Judge []Judge
}
