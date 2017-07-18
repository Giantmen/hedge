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
	name      string
	huidu     bool    //测试开关
	right     bool    //顺向交易开关
	left      bool    //逆向交易开关
	depth     float64 //深度
	amount    float64 //交易数量
	rightEarn float64 //顺向交易利润差
	leftEarn  float64 //逆向交易利润差
	ticker    int
	loop      *time.Ticker
	stop      chan struct{}
	status    bool //程序开关
	btctrade  account
	chbtc     account
}

type account struct {
	name    string
	bourse  bourse.Bourse
	cny     float64
	etc     float64
	fee_etc float64
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
	// TODO: 加入动态修改
	Depth, err := strconv.ParseFloat(cfg.Depth, 64)
	if err != nil {
		log.Error("cfg.Depth ParseFloat err", err)
		return nil, err
	}
	Amount, err := strconv.ParseFloat(cfg.Amount, 64)
	if err != nil {
		log.Error("cfg.Amount ParseFloat err", err)
		return nil, err
	}
	RightEarn, err := strconv.ParseFloat(cfg.Rightearn, 64)
	if err != nil {
		log.Error("cfg.RightEarn ParseFloat err", err)
		return nil, err
	}
	LeftEarn, err := strconv.ParseFloat(cfg.Leftearn, 64)
	if err != nil {
		log.Error("cfg.LeftEarn ParseFloat err", err)
		return nil, err
	}

	return &EtcChBtctrade{
		name:      cfg.Name,
		huidu:     cfg.Huidu,
		depth:     Depth,
		amount:    Amount,
		rightEarn: RightEarn, //利润差
		leftEarn:  LeftEarn,
		ticker:    cfg.Ticker,
		loop:      time.NewTicker(time.Second * time.Duration(cfg.Ticker)),
		stop:      make(chan struct{}),
		btctrade: account{
			name:    mypro.Btctrade,
			bourse:  service[strings.ToUpper(mypro.Btctrade)],
			fee_etc: mypro.Btctrade_etc,
		},
		chbtc: account{
			name:    mypro.Chbtc,
			bourse:  service[strings.ToUpper(mypro.Chbtc)],
			fee_etc: mypro.Chbtc_etc,
		},
	}, nil
}

func (j *EtcChBtctrade) Process() error {
	if !j.status {
		j.status = true
		log.Info("process start")
	} else {
		return fmt.Errorf("%s is already start", j.name)
	}

	j.getAccount()
	log.Infof("account btctrade cny:%f etc:%f", j.btctrade.cny, j.btctrade.etc)
	log.Infof("account chbtc cny:%f etc:%f", j.chbtc.cny, j.chbtc.etc)
	var accounter = time.NewTicker(time.Second * 100)
	defer accounter.Stop()
	for {
		select {
		case <-j.loop.C:
			go j.getAccount() //检查账户
			j.judge()

		case <-accounter.C:
			log.Infof("account btctrade cny:%f etc:%f", j.btctrade.cny, j.btctrade.etc)
			log.Infof("account chbtc cny:%f etc:%f", j.chbtc.cny, j.chbtc.etc)
		case <-j.stop:
			log.Info("process stop!")
			return nil
		}
	}
}

func (j *EtcChBtctrade) judge() {
	btctrade := j.getDepth(j.btctrade.bourse, j.depth)
	chbtc := j.getDepth(j.chbtc.bourse, j.depth)

	log.Debugf("btctrade: buy:%v sell:%v", btctrade.Buy, btctrade.Sell)
	log.Debugf("chbtc: buy:%v sell:%v", chbtc.Buy, chbtc.Sell)
	if profit := mypro.Earn(btctrade.Buy, j.btctrade.fee_etc, chbtc.Sell, j.chbtc.fee_etc); profit > j.rightEarn {
		if err := j.checkAccount(j.btctrade, j.chbtc, btctrade, chbtc, fmt.Sprintf("%f", j.amount)); err != nil {
			if j.right { //左边搬空 且向右开关开的
				log.Errorf("停止交易: %s -> %s %v", j.btctrade.name, j.chbtc.name, err)
				j.right = false
				return
			} else { //且向右开关关的
				log.Debugf("禁止交易:%s -> %s %v", j.btctrade.name, j.chbtc.name, err)
				log.Info("profit:", fmt.Sprintf("%0.4f", profit*j.amount), mypro.Btctrade, "sell:", btctrade.Buy, mypro.Chbtc, "buy:", chbtc.Sell)
				return
			}
		} else if !j.right { //仓位正常 且向右开关关的
			j.right = true
			log.Infof("恢复交易:  %s -> %s", j.btctrade.name, j.chbtc.name)
		}

		earn, err := j.hedging(j.btctrade, j.chbtc, btctrade, chbtc, fmt.Sprintf("%f", j.amount))
		if err == nil {
			if j.huidu {
				log.Info("profit:", fmt.Sprintf("%0.4f", profit*j.amount), mypro.Btctrade, "sell:", btctrade.Buy, mypro.Chbtc, "buy:", chbtc.Sell)
			} else {
				log.Debug("profit:", fmt.Sprintf("%0.4f", profit*j.amount), mypro.Btctrade, "sell:", btctrade.Buy, mypro.Chbtc, "buy:", chbtc.Sell)
				log.Info("earn:", fmt.Sprintf("%0.4f", earn*j.amount), mypro.Btctrade, "sell:", btctrade.Buy, mypro.Chbtc, "buy:", chbtc.Sell)
			}
		} else {
			log.Error("hedging err", err)
		}
		//balance += amount
	} else if profit := mypro.Earn(chbtc.Buy, mypro.Chbtc_etc, btctrade.Sell, mypro.Btctrade_etc); profit > j.leftEarn {
		if err := j.checkAccount(j.chbtc, j.btctrade, chbtc, btctrade, fmt.Sprintf("%f", j.amount)); err != nil {
			if j.left { //右边搬空 且向左开关开的
				log.Errorf("停止交易: %s -> %s %v", j.chbtc.name, j.btctrade.name, err)
				j.left = false
				return
			} else { //且向左开关关的
				log.Debugf("禁止交易:%s -> %s %v", j.chbtc.name, j.btctrade.name, err)
				log.Info("profit:", fmt.Sprintf("%0.4f", profit*j.amount), mypro.Chbtc, "sell:", chbtc.Buy, mypro.Btctrade, "buy:", btctrade.Sell)
				return
			}
		} else if !j.left { //仓位正常 且向左开关关的
			j.left = true
			log.Errorf("恢复交易:  %s -> %s", j.chbtc.name, j.btctrade.name)
		}

		earn, err := j.hedging(j.chbtc, j.btctrade, chbtc, btctrade, fmt.Sprintf("%f", j.amount))
		if err == nil {
			if j.huidu {
				log.Info("profit:", fmt.Sprintf("%0.4f", profit*j.amount), mypro.Chbtc, "sell:", chbtc.Buy, mypro.Btctrade, "buy:", btctrade.Sell)
			} else {
				log.Debug("profit:", fmt.Sprintf("%0.4f", profit*j.amount), mypro.Chbtc, "sell:", chbtc.Buy, mypro.Btctrade, "buy:", btctrade.Sell)
				log.Info("earn:", fmt.Sprintf("%0.4f", earn*j.amount), mypro.Chbtc, "sell:", chbtc.Buy, mypro.Btctrade, "buy:", btctrade.Sell)
			}
		} else {
			log.Error("hedging err", err)
		}
		//balance -= amount
	}
}

func (j *EtcChBtctrade) checkAccount(sellSide, buySide account, priceS, pirceB *proto.Price, amount string) error {
	num, _ := strconv.ParseFloat(amount, 64)
	if sellSide.etc < num*2 {
		return fmt.Errorf("%s:etc余额不足:%f cny:%f", sellSide.name, sellSide.etc, sellSide.cny)
	} else if buySide.cny < (pirceB.Sell * (num * 2)) {
		return fmt.Errorf("%s:cny余额不足:%f etc:%f", buySide.name, buySide.cny, buySide.etc)
	}
	return nil
}

func (j *EtcChBtctrade) hedging(sellSide, buySide account, priceS, pirceB *proto.Price, amount string) (float64, error) {
	if j.huidu {
		log.Debug("huidu on")
		return 0, nil
	}
	//sell
	order, err := j.deal(sellSide.bourse, proto.SELL, amount, fmt.Sprintf("%f", priceS.Buy))
	if err != nil {
		return 0, fmt.Errorf("%s:%s %v", sellSide.name, proto.SELL, err)
	}
	log.Info("sell", sellSide.name, priceS.Buy, amount, order, order.OrderID, order.Status)

	//buy
	order, err = j.deal(buySide.bourse, proto.BUY, amount, fmt.Sprintf("%f", pirceB.Sell))
	var buyprice = pirceB.Sell
	if err != nil {
		log.Error(err)
		buyprice = priceS.Buy*(1-sellSide.fee_etc) - pirceB.Sell*buySide.fee_etc //挂单价格=卖出的价格-手续费
		log.Debug("buyprice", buyprice, "=", priceS.Buy, "*(1-", sellSide.fee_etc, ")-", pirceB.Sell, "*", buySide.fee_etc)
		if order, err = j.retryBuy(buySide.bourse, proto.BUY, amount, fmt.Sprintf("%f", buyprice)); err != nil {
			return 0, err //重试失败
		}
		log.Info(sellSide.name, "buy retry ok")
	}
	log.Info("buy:", buySide.name, buyprice, amount, order, order.OrderID, order.Status)
	return mypro.Earn(priceS.Buy, sellSide.fee_etc, pirceB.Sell, buySide.fee_etc), nil
}

func (j *EtcChBtctrade) retryBuy(bou bourse.Bourse, side, amount, price string) (*proto.Order, error) {
	sec := rand.Intn(10)
	if sec == 0 {
		sec = 1
	}
	for {
		if order, err := j.deal(bou, side, amount, price); err == nil {
			return order, err
		} else {
			log.Errorf("retry err %v", err)
		}
		log.Debug("retryDeal sleep:", sec)
		time.Sleep(time.Duration(sec) * time.Millisecond)
		sec = sec << 1
		if sec > 40 {
			return nil, fmt.Errorf("retry %s err", side)
		}
	}
}

func (j *EtcChBtctrade) deal(bou bourse.Bourse, side, amount, price string) (*proto.Order, error) {
	var order *proto.Order
	var err error
	if side == proto.SELL {
		order, err = bou.Sell(amount, price, proto.ETC_CNY)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	} else if side == proto.BUY {
		order, err = bou.Buy(amount, price, proto.ETC_CNY)
		if err != nil {
			log.Error(err)
			return nil, err
		}
	}
	return j.checkOrder(bou, side, order.OrderID, order.Currency)
}

func (j *EtcChBtctrade) checkOrder(bou bourse.Bourse, side, orderId, currencyPair string) (*proto.Order, error) {
	sec := 50
	for {
		if order, err := bou.GetOneOrder(orderId, currencyPair); err != nil || order.Status == proto.ORDER_UNFINISH {
			log.Error("retry check", side, orderId, order.Status, err)
		} else {
			log.Debugf("retry check %s ok!", side)
			return order, err
		}
		log.Debug("checkOrder sleep:", sec)
		time.Sleep(time.Duration(sec) * time.Millisecond)
		sec = sec << 1
		if sec > 500 { //150 50 150 100 150 200 150 400  = 800ms
			if order, err := j.cancelOrder(bou, orderId, currencyPair); err != nil {
				return nil, fmt.Errorf("%s %s retry check & cancel err", side, orderId)
			} else if order != nil {
				return order, nil
			} else {
				return nil, fmt.Errorf("%s %s retry check err & cancel ok", side, orderId)
			}
		}
	}
}

func (j *EtcChBtctrade) cancelOrder(bou bourse.Bourse, orderId, currencyPair string) (*proto.Order, error) {
	sec := 10
	for {
		if cancel, cerr := bou.CancelOrder(orderId, currencyPair); !cancel || cerr != nil {
			log.Error("cancel err:", cancel, cerr)
			if order, err := bou.GetOneOrder(orderId, currencyPair); err != nil || order.Status == proto.ORDER_UNFINISH {
				//获取失败或者订单没有结束
				log.Error("retry cancel err", orderId, cancel, order.Status, cerr, err)
			} else if order.Status == proto.ORDER_CANCEL { //订单被取消了
				return nil, nil
			} else if order.Status == proto.ORDER_FINISH { //订单结束了
				return order, nil
			}
		} else {
			return nil, nil
		}
		log.Debug("CancelOrder sleep:", sec)
		time.Sleep(time.Duration(sec) * time.Millisecond)
		sec = sec << 1
		if sec > 40 { //300 10 300 20 300 40 = 930ms
			return nil, fmt.Errorf("retry cancel err")
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

func (j *EtcChBtctrade) Stop() error {
	if j.status {
		j.status = false
	} else {
		return fmt.Errorf("%s is already stop", j.name)
	}
	j.stop <- struct{}{}
	log.Infof("stop judge:%s ok", j.name)
	return nil
}

func (j *EtcChBtctrade) SetHuidu(huidu bool) bool {
	j.huidu = huidu
	log.Infof("set judge:%s huidu:%v ok", j.name, huidu)
	return j.huidu
}
func (j *EtcChBtctrade) SetDepth(depth float64) float64 {
	j.depth = depth
	log.Infof("set judge:%s depth:%v ok", j.name, depth)
	return j.depth
}

func (j *EtcChBtctrade) SetAmount(amount float64) float64 {
	j.amount = amount
	log.Infof("set judge:%s amount:%v ok", j.name, amount)
	return j.amount
}

func (j *EtcChBtctrade) SetRightEarn(rightEarn float64) float64 {
	j.rightEarn = rightEarn
	log.Infof("set judge:%s rightEarn:%v ok", j.name, rightEarn)
	return j.rightEarn
}

func (j *EtcChBtctrade) SetLeftEarn(leftEarn float64) float64 {
	j.leftEarn = leftEarn
	log.Infof("set judge:%s leftEarn:%v ok", j.name, leftEarn)
	return j.leftEarn
}

func (j *EtcChBtctrade) SetTicker(ticker int) string {
	j.ticker = ticker
	j.loop = time.NewTicker(time.Second * time.Duration(ticker))
	log.Infof("set judge:%s ticker:%v ok", j.name, ticker)
	return fmt.Sprintf("ticker set %d/s ok", ticker)
}

func (j *EtcChBtctrade) GetConfig() *mypro.ConfigReply {
	return &mypro.ConfigReply{
		Ticker:    j.ticker,
		Huidu:     j.huidu,
		Depth:     j.depth,
		Amount:    j.amount,
		RightEarn: j.rightEarn,
		LeftEarn:  j.leftEarn,
	}
}

func (j *EtcChBtctrade) Status() bool {
	return j.status
}
