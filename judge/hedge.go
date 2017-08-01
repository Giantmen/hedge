package judge

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Giantmen/hedge/config"
	mypro "github.com/Giantmen/hedge/proto"
	"github.com/Giantmen/hedge/store"
	"github.com/Giantmen/my_trader/log"

	"github.com/Giantmen/trader/bourse"
	"github.com/Giantmen/trader/proto"
)

var (
	bourseNameA = ""
	bourseNameB = ""
)

type Hedge struct {
	name      string  //策略名称
	coin      string  //币种
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
	status    bool    //程序开关
	first     string  //优先级
	bourseA   account //btctrade //////////////////
	bourseB   account //chbtc
}

type account struct {
	name   string
	bourse bourse.Bourse
	cny    float64
	coin   float64 //币种
	fee    float64
}

func NewHedge(cfg *config.Judge, sr *store.Service) (*Hedge, error) {
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

	if len(cfg.Bourse) != 2 {
		log.Error("len Bourse not 2", cfg.Bourse)
		return nil, fmt.Errorf("len Bourse not 2: %v", cfg.Bourse)
	}
	bourseNameA = cfg.Bourse[0]
	bourseA, ok := sr.Bourses[strings.ToUpper(bourseNameA)]
	if !ok {
		log.Errorf("get %s err", bourseNameA)
		return nil, fmt.Errorf("get %s err", bourseNameA)
	}
	bourseNameB = cfg.Bourse[1]
	bourseB, ok := sr.Bourses[strings.ToUpper(bourseNameB)]
	if !ok {
		log.Errorf("get %s err", bourseNameB)
		return nil, fmt.Errorf("get %s err", bourseNameB)
	}
	list := strings.Split(cfg.Name, "_") //获取币种
	coin := list[0]

	return &Hedge{
		name:      cfg.Name,
		coin:      coin,
		huidu:     cfg.Huidu,
		depth:     Depth,
		amount:    Amount,
		rightEarn: RightEarn, //利润差
		leftEarn:  LeftEarn,
		ticker:    cfg.Ticker,
		loop:      time.NewTicker(time.Second * time.Duration(cfg.Ticker)),
		stop:      make(chan struct{}),
		first:     bourseNameA,
		bourseA: account{
			name:   bourseNameA,
			bourse: bourseA,
			fee:    proto.ConvertFee(fmt.Sprintf("%s_%s", bourseNameA, coin)),
		},
		bourseB: account{
			name:   bourseNameB,
			bourse: bourseB,
			fee:    proto.ConvertFee(fmt.Sprintf("%s_%s", bourseNameB, coin)),
		},
	}, nil
}

func (h *Hedge) Process() error {
	if !h.status {
		h.status = true
		log.Infof("%s process start", h.name)
	} else {
		return fmt.Errorf("%s is already start", h.name)
	}

	h.getAccount()
	log.Infof("account %s cny:%f %s:%f", h.bourseA.name, h.bourseA.cny, h.coin, h.bourseA.coin)
	log.Infof("account %s cny:%f %s:%f", h.bourseB.name, h.bourseB.cny, h.coin, h.bourseB.coin)
	var accounter = time.NewTicker(time.Second * 100)
	defer accounter.Stop()
	for {
		select {
		case <-h.loop.C:
			go h.getAccount() //检查账户
			h.judge()

		case <-accounter.C:
			log.Infof("account %s cny:%f %s:%f", h.bourseA.name, h.bourseA.cny, h.coin, h.bourseA.coin)
			log.Infof("account %s cny:%f %s:%f", h.bourseB.name, h.bourseB.cny, h.coin, h.bourseB.coin)
		case <-h.stop:
			log.Infof("%s process stop!", h.name)
			return nil
		}
	}
}

func (h *Hedge) judge() {
	priceA := h.getDepth(h.bourseA.bourse, h.depth) //btctrade.Buy Sell
	priceB := h.getDepth(h.bourseB.bourse, h.depth) //chbtc

	log.Debugf("%s %s: buy:%v sell:%v", strings.ToUpper(h.coin), h.bourseA.name, priceA.Buy, priceA.Sell)
	log.Debugf("%s %s: buy:%v sell:%v", strings.ToUpper(h.coin), h.bourseB.name, priceB.Buy, priceB.Sell)
	if profit := mypro.Earn(priceA.Buy, h.bourseA.fee, priceB.Sell, h.bourseB.fee); profit > h.rightEarn {
		if err := h.checkAccount(h.bourseA, h.bourseB, priceB, h.amount); err != nil {
			if h.right { //左边搬空 且向右开关开的
				log.Errorf("%s 停止交易: %s -> %s %v", strings.ToUpper(h.coin), h.bourseA.name, h.bourseB.name, err)
				h.right = false
				return
			} else { //且向右开关关的
				log.Debugf("%s 禁止交易:%s -> %s %v", strings.ToUpper(h.coin), h.bourseA.name, h.bourseB.name, err)
				log.Info("profit:", fmt.Sprintf("%0.4f", profit*h.amount), strings.ToUpper(h.bourseA.name), "sell:", priceA.Buy, strings.ToUpper(h.bourseB.name), "buy:", priceB.Sell)
				return
			}
		} else if !h.right { //仓位正常 且向右开关关的
			h.right = true
			log.Infof("%s 恢复交易:  %s -> %s", strings.ToUpper(h.coin), h.bourseA.name, h.bourseB.name)
		}

		earn, err := h.hedging(h.bourseA, h.bourseB, priceA, priceB, h.amount)
		if err == nil {
			if h.huidu {
				log.Info(strings.ToUpper(h.coin), "profit:", fmt.Sprintf("%0.4f", profit*h.amount), strings.ToUpper(h.bourseA.name), "sell:", priceA.Buy, strings.ToUpper(h.bourseB.name), "buy:", priceB.Sell)
			} else {
				log.Info(strings.ToUpper(h.coin), "earn:", fmt.Sprintf("%0.4f", earn*h.amount), strings.ToUpper(h.bourseA.name), "sell:", priceA.Buy, strings.ToUpper(h.bourseB.name), "buy:", priceB.Sell)
			}
		} else {
			log.Error(strings.ToUpper(h.coin), "hedging err", err)
		}
	} else if profit := mypro.Earn(priceB.Buy, h.bourseB.fee, priceA.Sell, h.bourseA.fee); profit > h.leftEarn {
		if err := h.checkAccount(h.bourseB, h.bourseA, priceA, h.amount); err != nil {
			if h.left { //右边搬空 且向左开关开的
				log.Errorf("%s 停止交易: %s -> %s %v", strings.ToUpper(h.coin), h.bourseB.name, h.bourseA.name, err)
				h.left = false
				return
			} else { //且向左开关关的
				log.Debugf("%s 禁止交易:%s -> %s %v", strings.ToUpper(h.coin), h.bourseB.name, h.bourseA.name, err)
				log.Info(strings.ToUpper(h.coin), "profit:", fmt.Sprintf("%0.4f", profit*h.amount), h.bourseB.name, "sell:", priceB.Buy, h.bourseA.name, "buy:", priceA.Sell)
				return
			}
		} else if h.left { //仓位正常 且向左开关关的
			h.left = true
			log.Errorf("%s 恢复交易:  %s -> %s", strings.ToUpper(h.coin), h.bourseB.name, h.bourseA.name)
		}

		earn, err := h.hedging(h.bourseB, h.bourseA, priceB, priceA, h.amount)
		if err == nil {
			if h.huidu {
				log.Info(strings.ToUpper(h.coin), "profit:", fmt.Sprintf("%0.4f", profit*h.amount), strings.ToUpper(h.bourseB.name), "sell:", priceB.Buy, strings.ToUpper(h.bourseA.name), "buy:", priceA.Sell)
			} else {
				log.Info(strings.ToUpper(h.coin), "earn:", fmt.Sprintf("%0.4f", earn*h.amount), strings.ToUpper(h.bourseB.name), "sell:", priceB.Buy, strings.ToUpper(h.bourseA.name), "buy:", priceA.Sell)
			}
		} else {
			log.Error(strings.ToUpper(h.coin), "hedging err", err)
		}
	}
}

func (h *Hedge) getDepth(bou bourse.Bourse, depth float64) *proto.Price {
	var price *proto.Price
	var err error
	if price, err = bou.GetPriceOfDepth(50, depth, mypro.ConvertCurrencyPair(h.coin)); err != nil {
		log.Error("getdepth err", err)
	}

	for err != nil {
		if price, err = bou.GetPriceOfDepth(50, depth, mypro.ConvertCurrencyPair(h.coin)); err != nil {
			log.Error("getdepth err", err)
		}
	}
	return price
}

func (h *Hedge) getAccount() {
	account, err := h.bourseA.bourse.GetAccount()
	if err != nil {
		log.Errorf("get account err %s:%v", h.bourseA.name, err)
	} else {
		h.bourseA.cny = account.SubAccounts[mypro.CNY].Available
		h.bourseA.coin = account.SubAccounts[h.coin].Available
	}

	account, err = h.bourseB.bourse.GetAccount()
	if err != nil {
		log.Errorf("get account err %s:%v", h.bourseB.name, err)
	} else {
		h.bourseB.cny = account.SubAccounts[mypro.CNY].Available
		h.bourseB.coin = account.SubAccounts[h.coin].Available
	}
}

func (h *Hedge) checkAccount(sellSide, buySide account, pirceB *proto.Price, amount float64) error {
	if sellSide.coin < amount*2 {
		return fmt.Errorf("%s:%s余额不足:%f cny:%f", sellSide.name, h.coin, sellSide.coin, sellSide.cny)
	} else if buySide.cny < (pirceB.Sell * (amount * 2)) {
		return fmt.Errorf("%s:cny余额不足:%f %s:%f", buySide.name, buySide.cny, h.coin, buySide.coin)
	}
	return nil
}

func (h *Hedge) hedging(sellSide, buySide account, priceA, priceB *proto.Price, amount float64) (float64, error) {
	if h.huidu {
		log.Debug(strings.ToUpper(h.coin), "huidu on")
		return 0, nil
	}
	if sellSide.name == h.first {
		order, err := h.sell(sellSide.bourse, amount, priceA.Buy, false)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", sellSide.name, proto.SELL, err)
		}
		log.Info("sell", sellSide.name, priceA.Buy, amount, order, order.OrderID, order.Status)

		order, err = h.buy(buySide, amount, priceB.Sell, true)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", buySide.name, proto.BUY, err)
		}
		log.Info("buy:", buySide.name, priceB.Sell, amount, order, order.OrderID, order.Status)
	}

	if buySide.name == h.first {
		order, err := h.buy(buySide, amount, priceB.Sell, false)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", buySide.name, proto.BUY, err)
		}
		log.Info(strings.ToUpper(h.coin), "buy:", buySide.name, priceB.Sell, amount, order, order.OrderID, order.Status)

		order, err = h.sell(sellSide.bourse, amount, priceA.Buy, true)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", sellSide.name, proto.SELL, err)
		}
		log.Info(strings.ToUpper(h.coin), "sell", sellSide.name, priceA.Buy, amount, order, order.OrderID, order.Status)
	}
	return mypro.Earn(priceA.Buy, sellSide.fee, priceB.Sell, buySide.fee), nil
}

func (h *Hedge) buy(bou account, amount, price float64, isRetry bool) (*proto.Order, error) {
	var amountRate float64
	if strings.ToLower(bou.name) == strings.ToLower(proto.Bter) { //针对bter
		amountRate = amount * (1 + bou.fee) / 1.5
	} else {
		amountRate = amount * (1 + bou.fee)
	}
	order, err := h.deal(bou.bourse, proto.BUY, fmt.Sprintf("%v", amountRate), fmt.Sprintf("%v", price*1.5))
	if err == nil {
		return order, err
	} else {
		if isRetry {
			return h.retryDeal(bou.bourse, proto.BUY, fmt.Sprintf("%v", amountRate), fmt.Sprintf("%v", price*1.5))
		}
		return nil, fmt.Errorf("sell err")
	}
}

func (h *Hedge) sell(bou bourse.Bourse, amount, price float64, isRetry bool) (*proto.Order, error) {
	order, err := h.deal(bou, proto.SELL, fmt.Sprintf("%v", amount), fmt.Sprintf("%v", price*0.5))
	if err == nil {
		return order, err
	} else {
		if isRetry {
			return h.retryDeal(bou, proto.SELL, fmt.Sprintf("%v", amount), fmt.Sprintf("%v", price*0.5))
		}
		return nil, fmt.Errorf("sell err")
	}
}

func (h *Hedge) retryDeal(bou bourse.Bourse, side, amount, price string) (*proto.Order, error) {
	sec := rand.Intn(10)
	if sec == 0 {
		sec = 1
	}
	for {
		if order, err := h.deal(bou, side, amount, price); err == nil {
			return order, err
		} else {
			log.Errorf("%s retry %s err %v", strings.ToUpper(h.coin), side, err)
		}
		log.Debug(strings.ToUpper(h.coin), "retryDeal sleep:", sec)
		time.Sleep(time.Duration(sec) * time.Millisecond)
		sec = sec << 1
		if sec > 40 {
			return nil, fmt.Errorf("retry %s err", side)
		}
	}
}

func (h *Hedge) deal(bou bourse.Bourse, side, amount, price string) (*proto.Order, error) {
	if side == proto.SELL {
		return bou.Sell(amount, price, mypro.ConvertCurrencyPair(h.coin))
	} else if side == proto.BUY {
		return bou.Buy(amount, price, mypro.ConvertCurrencyPair(h.coin))
	}
	return nil, fmt.Errorf("err side:%s", side)
}

func (h *Hedge) Stop() error {
	if h.status {
		h.status = false
	} else {
		return fmt.Errorf("%s is already stop", h.name)
	}
	h.stop <- struct{}{}
	log.Infof("stop judge:%s ok", h.name)
	return nil
}

func (h *Hedge) SetHuidu(huidu bool) bool {
	h.huidu = huidu
	log.Infof("set judge:%s huidu:%v ok", h.name, huidu)
	return h.huidu
}

func (h *Hedge) SetDepth(depth float64) float64 {
	h.depth = depth
	log.Infof("set judge:%s depth:%v ok", h.name, depth)
	return h.depth
}

func (h *Hedge) SetAmount(amount float64) float64 {
	h.amount = amount
	log.Infof("set judge:%s amount:%v ok", h.name, amount)
	return h.amount
}

func (h *Hedge) SetRightEarn(rightEarn float64) float64 {
	h.rightEarn = rightEarn
	log.Infof("set judge:%s rightEarn:%v ok", h.name, rightEarn)
	return h.rightEarn
}

func (h *Hedge) SetLeftEarn(leftEarn float64) float64 {
	h.leftEarn = leftEarn
	log.Infof("set judge:%s leftEarn:%v ok", h.name, leftEarn)
	return h.leftEarn
}

func (h *Hedge) SetTicker(ticker int) string {
	h.ticker = ticker
	h.loop = time.NewTicker(time.Second * time.Duration(ticker))
	log.Infof("set judge:%s ticker:%v ok", h.name, ticker)
	return fmt.Sprintf("ticker set %d/s ok", ticker)
}

func (h *Hedge) SetFirst(first string) string {
	h.first = strings.ToLower(first)
	log.Infof("set judge:%s first:%v ok", h.name, first)
	return fmt.Sprintf("first set %s ok", first)
}

func (h *Hedge) GetConfig() *mypro.ConfigReply {
	return &mypro.ConfigReply{
		Ticker:    h.ticker,
		First:     h.first,
		Huidu:     h.huidu,
		Depth:     h.depth,
		Amount:    h.amount,
		RightEarn: h.rightEarn,
		LeftEarn:  h.leftEarn,
	}
}

func (h *Hedge) Status() bool {
	return h.status
}
