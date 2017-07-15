package config

type Server struct {
	Name      string
	Accesskey string
	Secretkey string
	Timeout   int
}

type Judge struct {
	Name   string
	Bourse []string
	Ticker int
	Profit string
	Huidu  bool
}

type Config struct {
	Listen string

	Debug    bool
	LogPath  string
	LogLevel string

	Chbtc    []Server
	Yunbi    []Server
	Huobi    []Server
	Btctrade []Server

	Judge []Judge
}
