package judge

import (
	"strings"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/log"
	"github.com/Giantmen/hedge/store"
)

type Processer interface {
	Process()
	Stop()
}

type Judge struct {
	judges map[string]Processer
}

func NewJudge(cfg *config.Config, sr *store.Service) (*Judge, error) {
	var judges = make(map[string]Processer)
	for _, c := range cfg.Judge {
		switch c.Name {
		case "etc_btctrade_chbtc":
			if etcChYun, err := NewEtcChBtctrade(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = etcChYun
			}
		case "ltc_huobi2_chbtc": //策略name
		}
	}

	return &Judge{
		judges: judges,
	}, nil
}

func (j *Judge) Start() {
	for name, judge := range j.judges {
		go judge.Process()
		log.Info("judge", name, "start", "ok")
	}
}

func (j *Judge) Stop() {
	for name, judge := range j.judges {
		judge.Stop()
		log.Info("judge", name, "stop", "ok")
	}
}
