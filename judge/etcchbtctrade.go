package judge

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/log"
	mypro "github.com/Giantmen/hedge/proto"
	"github.com/Giantmen/hedge/store"

	"github.com/Giantmen/trader/bourse"
	"github.com/Giantmen/trader/proto"
)

type EtcChBtctrade struct {
	name     string
	profit   float64 //利润差
	huidu    bool    //测试开关
	right    bool    //顺向交易开关
	left     bool    //逆向交易开关
	ticker   *time.Ticker
	btctrade account
	chbtc    account
}

type account struct {
	name   string
	bourse bourse.Bourse
	cny    float64
	etc    float64
}

func NewEtcChBtctrade(cfg *config.Judge, sr *store.Service) (*EtcChBtctrade, error) {
	var service = make(map[string]bourse.Bourse, len(cfg.Bourse))
	for _, bourse := range cfg.Bourse {
		bou, ok := sr.Bourses[strings.ToUpper(bourse)]
		if !ok {
			log.Errorf("get %s err", bourse)
			return nil, fmt.Errorf("get %s err", bourse)
		}
		service[strings.ToUpper(bourse)] = bou
	}

	Profit, err := strconv.ParseFloat(cfg.Profit, 64)
	if err != nil {
		log.Error("cfg.Profit ParseFloat err", err)
		return nil, err
	}
	return &EtcChBtctrade{
		name:   cfg.Name,
		profit: Profit, //利润差
		huidu:  cfg.Huidu,
		ticker: time.NewTicker(time.Second * time.Duration(cfg.Ticker)),
		btctrade: account{
			name:   mypro.Btctrade,
			bourse: service[strings.ToUpper(mypro.Btctrade)],
		},
		chbtc: account{
			name:   mypro.Chbtc,
			bourse: service[strings.ToUpper(mypro.Chbtc)],
		},
	}, nil
}

func (j *EtcChBtctrade) Process() {
	j.getAccount()
	log.Infof("account btctrade cny:%f etc:%f", j.btctrade.cny, j.btctrade.etc)
	log.Infof("account chbtc cny:%f etc:%f", j.chbtc.cny, j.chbtc.etc)
	// TODO: 加入动态修改
	var depth float64 = 3
	var amount float64 = 1
	var accounter = time.NewTicker(time.Second * 100)
	defer accounter.Stop()
	for {
		select {
		case <-j.ticker.C:
			go j.getAccount() //检查账户
			j.judge(depth, amount)

		case <-accounter.C:
			log.Infof("account btctrade cny:%f etc:%f", j.btctrade.cny, j.btctrade.etc)
			log.Infof("account chbtc cny:%f etc:%f", j.chbtc.cny, j.chbtc.etc)
		}
	}
}

func (j *EtcChBtctrade) judge(depth, amount float64) {
	btctrade := j.getDepth(j.btctrade.bourse, depth)
	chbtc := j.getDepth(j.chbtc.bourse, depth)

	log.Debug("btctrade:", btctrade)
	log.Debug("chbtc", chbtc)
	if earn := mypro.Earn(btctrade.Buy, mypro.Btctrade_etc, chbtc.Sell, mypro.Chbtc_etc); earn > j.profit {
		if err := j.checkAccount(j.btctrade, j.chbtc, btctrade, chbtc, fmt.Sprintf("%f", amount)); err != nil {
			if j.right { //左边搬空 且向右开关开的
				log.Errorf("停止交易: %s -> %s %v", j.btctrade.name, j.chbtc.name, err)
				j.right = false
				return
			} else { //且向右开关关的
				log.Debugf("禁止交易:%s -> %s %v", j.btctrade.name, j.chbtc.name, err)
				log.Info("earn:", fmt.Sprintf("%0.2f", earn*amount), mypro.Btctrade, "sell:", btctrade.Buy, mypro.Chbtc, "buy:", chbtc.Sell)
				return
			}
		} else if !j.right { //仓位正常 且向右开关关的
			j.right = true
			log.Infof("恢复交易:  %s -> %s", j.btctrade.name, j.chbtc.name)
		}

		err := j.hedging(j.btctrade, j.chbtc, btctrade, chbtc, fmt.Sprintf("%f", amount))
		if err == nil {
			log.Info("earn:", fmt.Sprintf("%0.2f", earn*amount), mypro.Btctrade, "sell:", btctrade.Buy, mypro.Chbtc, "buy:", chbtc.Sell)
		} else {
			log.Error("hedging err", err)
		}
		//balance += amount
	} else if earn := mypro.Earn(chbtc.Buy, mypro.Chbtc_etc, btctrade.Sell, mypro.Btctrade_etc); earn > j.profit {
		if err := j.checkAccount(j.chbtc, j.btctrade, chbtc, btctrade, fmt.Sprintf("%f", amount)); err != nil {
			if j.left { //右边搬空 且向左开关开的
				log.Errorf("停止交易: %s -> %s %v", j.chbtc.name, j.btctrade.name, err)
				j.left = false
				return
			} else { //且向左开关关的
				log.Debugf("禁止交易:%s -> %s %v", j.chbtc.name, j.btctrade.name, err)
				log.Info("earn:", fmt.Sprintf("%0.2f", earn*amount), mypro.Chbtc, "sell:", chbtc.Buy, mypro.Btctrade, "buy:", btctrade.Sell)
				return
			}
		} else if !j.left { //仓位正常 且向左开关关的
			j.left = true
			log.Errorf("恢复交易:  %s -> %s", j.chbtc.name, j.btctrade.name)
		}

		err := j.hedging(j.chbtc, j.btctrade, chbtc, btctrade, fmt.Sprintf("%f", amount))
		if err == nil {
			log.Info("earn:", fmt.Sprintf("%0.2f", earn*amount), mypro.Chbtc, "sell:", chbtc.Buy, mypro.Btctrade, "buy:", btctrade.Sell)
		} else {
			log.Error("hedging err", err)
		}
		//balance -= amount
	}
}

func (j *EtcChBtctrade) checkAccount(sellSide, buySide account, priceS, pirceB *proto.Price, amount string) error {
	num, _ := strconv.ParseFloat(amount, 64)
	if sellSide.etc < num {
		return fmt.Errorf("%s:etc余额不足:%f cny:%f", sellSide.name, sellSide.etc, sellSide.cny)
	} else if buySide.cny < (pirceB.Sell * num) {
		return fmt.Errorf("%s:cny余额不足:%f etc:%f", buySide.name, buySide.cny, buySide.etc)
	}
	return nil
}

func (j *EtcChBtctrade) hedging(sellSide, buySide account, priceS, pirceB *proto.Price, amount string) error {
	if j.huidu {
		log.Debug("huidu on")
		return nil
	}
	//sell
	order, err := j.deal(sellSide.bourse, proto.SELL, amount, fmt.Sprintf("%0.3f", priceS.Buy))
	if err != nil {
		log.Error(err)
		return err
	}
	log.Info("sell", priceS.Buy, amount, order, order.OrderID, order.Status)
	//buy
	order, err = j.deal(buySide.bourse, proto.BUY, amount, fmt.Sprintf("%0.3f", pirceB.Sell))
	if err != nil {
		log.Error(err)
		if order, err = j.retryDeal(buySide.bourse, proto.BUY, amount); err != nil {
			return err //重试失败
		}
		log.Info(sellSide.name, "buy retry ok")
	}
	log.Info("buy:", pirceB.Sell, amount, order, order.OrderID, order.Status)
	return nil
}

func (j *EtcChBtctrade) retryDeal(bou bourse.Bourse, side, amount string) (*proto.Order, error) {
	sec := rand.Intn(10)
	for {
		time.Sleep(time.Duration(sec) * time.Millisecond)
		price := j.getDepth(bou, 1)
		if order, err := j.deal(bou, side, amount, fmt.Sprintf("%0.3f", price.Sell)); err == nil {
			return order, err
		} else {
			log.Errorf("retry err %v", err)
		}
		sec = sec << 1
		if sec > 100 {
			return nil, fmt.Errorf("retry err")
		}
	}
}

func (j *EtcChBtctrade) deal(bou bourse.Bourse, side, amount, price string) (*proto.Order, error) {
	var order *proto.Order
	var err error
	if side == proto.SELL {
		order, err = bou.Sell(amount, price, proto.ETC_CNY)
		if err != nil {
			log.Error("sell err", err)
			return nil, err
		}
	} else if side == proto.BUY {
		order, err = bou.Buy(amount, price, proto.ETC_CNY)
		if err != nil {
			log.Error("buy err", err)
			return nil, err
		}
	}
	return j.checkOrder(bou, order.OrderID, order.Currency)
}

func (j *EtcChBtctrade) checkOrder(bou bourse.Bourse, orderId, currencyPair string) (*proto.Order, error) {
	sec := 5
	for {
		if order, err := bou.GetOneOrder(orderId, currencyPair); err != nil || order.Status == proto.ORDER_UNFINISH {
			log.Error("retry ordercheck", order.Status, err)
		} else {
			return order, err
		}
		time.Sleep(time.Duration(sec) * time.Millisecond)
		sec = sec << 1
		if sec > 80 {
			if err := j.cancelOrder(bou, orderId, currencyPair); err != nil {
				log.Error(err)
				return nil, fmt.Errorf("%s retry ordercheck cancel err", orderId)
			}
			return nil, fmt.Errorf("%s retry ordercheck err cancel ok", orderId)
		}
	}
}

func (j *EtcChBtctrade) cancelOrder(bou bourse.Bourse, orderId, currencyPair string) error {
	sec := rand.Intn(5)
	for {
		if cancel, err := bou.CancelOrder(orderId, currencyPair); !cancel || err != nil {
			if order, err := bou.GetOneOrder(orderId, currencyPair); err != nil || order.Status == proto.ORDER_UNFINISH {
				log.Error("CancelOrder GetOneOrder err", order.Status, err)
			} else {
				return nil
			}
			log.Error("retry cancelOrder", cancel, err)
		} else {
			return nil
		}
		time.Sleep(time.Duration(sec) * time.Millisecond)
		sec = sec << 1
		if sec > 50 {
			return fmt.Errorf("retry cancelOrder err")
		}
	}
}

func (j *EtcChBtctrade) getAccount() {
	account, err := j.btctrade.bourse.GetAccount()
	if err != nil {
		log.Error("get account err btctrade:", err)
	} else {
		j.btctrade.cny = account.SubAccounts[proto.CNY].Available
		j.btctrade.etc = account.SubAccounts[proto.ETC].Available
	}

	account, err = j.chbtc.bourse.GetAccount()
	if err != nil {
		log.Error("get account err chbtc:", err)
	} else {
		j.chbtc.cny = account.SubAccounts[proto.CNY].Available
		j.chbtc.etc = account.SubAccounts[proto.ETC].Available
	}
}

func (j *EtcChBtctrade) getDepth(bou bourse.Bourse, depth float64) *proto.Price {
	var price *proto.Price
	var err error
	if price, err = bou.GetPriceOfDepth(50, depth, proto.ETC_CNY); err != nil {
		log.Error("getdepth err", err)
	}

	for err != nil {
		if price, err = bou.GetPriceOfDepth(50, depth, proto.ETC_CNY); err != nil {
			log.Error("getdepth err", err)
		}
	}
	return price
}

func (j *EtcChBtctrade) Stop() {
	j.ticker.Stop()
}
