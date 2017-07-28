package store

import (
	"fmt"
	"strings"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/log"
	"github.com/solomoner/gozilla"

	"github.com/Giantmen/trader/bourse"
	"github.com/Giantmen/trader/bourse/btctrade"
	"github.com/Giantmen/trader/bourse/bter"
	"github.com/Giantmen/trader/bourse/chbtc"
	"github.com/Giantmen/trader/bourse/huobiN"
	"github.com/Giantmen/trader/bourse/huobiO"
	"github.com/Giantmen/trader/bourse/yunbi"
	"github.com/Giantmen/trader/proto"
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

	return &Service{
		Bourses: bourses,
	}, nil
}

func (s *Service) GetPriceOfDepth(ctx *gozilla.Context, r *proto.DepthQuery) (*proto.Price, error) {
	bou, ok := s.Bourses[strings.ToUpper(r.Bourse)]
	if !ok {
		log.Errorf("get %s err", r.Bourse)
		return nil, fmt.Errorf("get %s err", r.Bourse)
	}
	return bou.GetPriceOfDepth(r.Size, r.Depth, r.Currency)
}

func (s *Service) GetAccount(ctx *gozilla.Context, r *proto.AccountQuery) (*proto.AccountReply, error) {
	bou, ok := s.Bourses[strings.ToUpper(r.Bourse)]
	if !ok {
		log.Errorf("get %s err", r.Bourse)
		return nil, fmt.Errorf("get %s err", r.Bourse)
	}
	account, err := bou.GetAccount()
	if err != nil {
		log.Error("GetAccount err", err)
		return nil, err
	}
	//var Accounts = make(map[string]proto.SubAccount)
	// for _, currency := range r.Accounts {
	// 	if sub, ok := account.SubAccounts[currency]; ok {
	// 		Accounts[currency] = sub
	// 	} else {
	// 		log.Error("can not find", currency)
	// 	}
	// }
	return &proto.AccountReply{
		Bourse:   account.Bourse,
		Asset:    account.Asset,
		Accounts: account.SubAccounts,
	}, nil
}

func (s *Service) Sell(ctx *gozilla.Context, r *proto.OrderQuery) (*proto.Order, error) {
	bou, ok := s.Bourses[strings.ToUpper(r.Bourse)]
	if !ok {
		log.Errorf("get %s err", r.Bourse)
		return nil, fmt.Errorf("get %s err", r.Bourse)
	}
	order, err := bou.Sell(r.Amount, r.Price, r.Currency)
	if err != nil {
		log.Error("sell err", err)
		return nil, err
	}
	return bou.GetOneOrder(order.OrderID, order.Currency)
}

func (s *Service) Buy(ctx *gozilla.Context, r *proto.OrderQuery) (*proto.Order, error) {
	bou, ok := s.Bourses[strings.ToUpper(r.Bourse)]
	if !ok {
		log.Errorf("get %s err", r.Bourse)
		return nil, fmt.Errorf("get %s err", r.Bourse)
	}
	order, err := bou.Buy(r.Amount, r.Price, r.Currency)
	if err != nil {
		log.Error("sell err", err)
		return nil, err
	}
	return bou.GetOneOrder(order.OrderID, order.Currency)
}

func (s *Service) CancelOrder(ctx *gozilla.Context, r *proto.OneOrderQuery) (bool, error) {
	bou, ok := s.Bourses[strings.ToUpper(r.Bourse)]
	if !ok {
		log.Errorf("get %s err", r.Bourse)
		return false, fmt.Errorf("get %s err", r.Bourse)
	}
	return bou.CancelOrder(r.OrderID, r.Currency)
}

func (s *Service) GetOneOrder(ctx *gozilla.Context, r *proto.OneOrderQuery) (*proto.Order, error) {
	bou, ok := s.Bourses[strings.ToUpper(r.Bourse)]
	if !ok {
		log.Errorf("get %s err", r.Bourse)
		return nil, fmt.Errorf("get %s err", r.Bourse)
	}
	return bou.GetOneOrder(r.OrderID, r.Currency)
}
