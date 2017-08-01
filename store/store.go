package store

import (
	"strings"

	"github.com/Giantmen/hedge/config"

	"github.com/Giantmen/trader/bourse"
	"github.com/Giantmen/trader/bourse/btctrade"
	"github.com/Giantmen/trader/bourse/bter"
	"github.com/Giantmen/trader/bourse/chbtc"
	"github.com/Giantmen/trader/bourse/huobiN"
	"github.com/Giantmen/trader/bourse/huobiO"
	"github.com/Giantmen/trader/bourse/poloniex"
	"github.com/Giantmen/trader/bourse/yunbi"
)

type Service struct {
	Bourses map[string]bourse.Bourse
}

func NewService(cfg *config.Config) (*Service, error) {
	var bourses = make(map[string]bourse.Bourse)
	for _, c := range cfg.Yunbi {
		if yunbi, err := yunbi.NewYunbi(c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = yunbi
		}
	}

	for _, c := range cfg.Chbtc {
		if chbtc, err := chbtc.NewChbtc(c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = chbtc
		}
	}

	for _, c := range cfg.Btctrade {
		if btctrade, err := btctrade.NewBtctrade(c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = btctrade
		}
	}

	for _, c := range cfg.HuobiN {
		if huobin, err := huobiN.NewHuobi(c.Accountid, c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = huobin
		}
	}

	for _, c := range cfg.HuobiO {
		if huobio, err := huobiO.NewHuobi(c.Accountid, c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = huobio
		}
	}

	for _, c := range cfg.Bter {
		if bter, err := bter.NewBter(c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = bter
		}
	}

	for _, c := range cfg.Poloniex {
		if poloniex, err := poloniex.NewPoloniex(c.Accesskey, c.Secretkey, c.Timeout); err != nil {
			return nil, err
		} else {
			bourses[strings.ToUpper(c.Name)] = poloniex
		}
	}

	return &Service{
		Bourses: bourses,
	}, nil
}
