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
	"github.com/Giantmen/trader/bourse"
	"github.com/Giantmen/trader/proto"

	"github.com/golang/glog"
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
	clearCur  *time.Ticker
	stop      chan struct{}
	status    bool    //程序开关
	first     string  //优先级
	bourseA   account //btctrade
	bourseB   account //chbtc
	income    income  //收入
}

type account struct {
	name   string
	bourse bourse.Bourse
	cny    float64
	coin   float64 //币种
	fee    float64
}

type income struct {
	all float64
	cur float64
}

func NewHedge(cfg *config.Judge, sr *store.Service) (*Hedge, error) {
	Depth, err := strconv.ParseFloat(cfg.Depth, 64)
	if err != nil {
		glog.Infoln("cfg.Depth ParseFloat err", err)
		return nil, err
	}
	Amount, err := strconv.ParseFloat(cfg.Amount, 64)
	if err != nil {
		glog.Infoln("cfg.Amount ParseFloat err", err)
		return nil, err
	}
	RightEarn, err := strconv.ParseFloat(cfg.Rightearn, 64)
	if err != nil {
		glog.Infoln("cfg.RightEarn ParseFloat err", err)
		return nil, err
	}
	LeftEarn, err := strconv.ParseFloat(cfg.Leftearn, 64)
	if err != nil {
		glog.Infoln("cfg.LeftEarn ParseFloat err", err)
		return nil, err
	}
	listI := strings.Split(cfg.Income, "#")
	if err != nil || len(listI) < 2 {
		glog.Infoln("Split Income err", err)
		return nil, err
	}
	all, err := strconv.ParseFloat(listI[0], 64)
	if err != nil {
		glog.Infoln("cfg.Income ParseFloat all err", err)
		return nil, err
	}
	cur, err := strconv.ParseFloat(listI[1], 64)
	if err != nil {
		glog.Infoln("cfg.Income ParseFloat cur err", err)
		return nil, err
	}

	if len(cfg.Bourse) != 2 {
		glog.Infoln("len Bourse not 2", cfg.Bourse)
		return nil, fmt.Errorf("len Bourse not 2: %v", cfg.Bourse)
	}
	bourseNameA = cfg.Bourse[0]
	bourseA, ok := sr.Bourses[strings.ToUpper(bourseNameA)]
	if !ok {
		glog.Infof("err get %s err", bourseNameA)
		return nil, fmt.Errorf("get %s err", bourseNameA)
	}
	bourseNameB = cfg.Bourse[1]
	bourseB, ok := sr.Bourses[strings.ToUpper(bourseNameB)]
	if !ok {
		glog.Infof("err get %s err", bourseNameB)
		return nil, fmt.Errorf("get %s err", bourseNameB)
	}
	listN := strings.Split(cfg.Name, "_") //获取币种
	coin := listN[0]

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
		clearCur:  time.NewTicker(time.Hour * 24),
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
		income: income{
			all: all,
			cur: cur,
		},
	}, nil
}

func (h *Hedge) setTicker() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for range ticker.C {
		if time.Now().Hour() == 0 { //0点
			h.clearCur = time.NewTicker(time.Hour * 24)
			glog.Info("new ticker")
			glog.Infof("income all: %v today: %v", h.income.all, h.income.cur)
			h.income.cur = 0
			return
		}
	}
}

func (h *Hedge) Process() error {
	if !h.status {
		h.status = true
		glog.Infof("%s process start", h.name)
	} else {
		return fmt.Errorf("%s is already start", h.name)
	}
	go h.setTicker()
	h.getAccount()
	glog.Infof("account %s cny:%f %s:%f", h.bourseA.name, h.bourseA.cny, h.name, h.bourseA.coin)
	glog.Infof("account %s cny:%f %s:%f", h.bourseB.name, h.bourseB.cny, h.name, h.bourseB.coin)
	var accounter = time.NewTicker(time.Second * 100)
	defer accounter.Stop()
	for {
		select {
		case <-h.loop.C:
			go h.getAccount()
			h.judge()

		case <-accounter.C:
			glog.Infof("account %s cny:%f %s:%f", h.bourseA.name, h.bourseA.cny, h.name, h.bourseA.coin)
			glog.Infof("account %s cny:%f %s:%f", h.bourseB.name, h.bourseB.cny, h.name, h.bourseB.coin)
			glog.Infof("income_now all: %v today: %v", h.income.all, h.income.cur)
		case <-h.stop:
			glog.Infof("%s process stop!", h.name)
			return nil
		case <-h.clearCur.C:
			glog.Infof("income all: %v today: %v", h.income.all, h.income.cur)
			h.income.cur = 0
		}
	}
}

func (h *Hedge) judge() {
	priceA := h.getDepth(h.bourseA.bourse, h.depth)
	priceB := h.getDepth(h.bourseB.bourse, h.depth)
	// glog.Infof("%s %s: buy:%v sell:%v", strings.ToUpper(h.name), h.bourseA.name, priceA.Buy, priceA.Sell)
	// glog.Infof("%s %s: buy:%v sell:%v", strings.ToUpper(h.name), h.bourseB.name, priceB.Buy, priceB.Sell)
	amount := h.amount
	if profit := mypro.Earn(priceA.Buy, h.bourseA.fee, priceB.Sell, h.bourseB.fee); profit > h.rightEarn {
		if err := h.checkAccount(h.bourseA, h.bourseB, priceB, amount); err != nil {
			if h.right { //左边搬空 且向右开关开的
				glog.Infof("err %s 停止交易: %s -> %s %v", strings.ToUpper(h.name), h.bourseA.name, h.bourseB.name, err)
				h.right = false
				return
			} else { //且向右开关关的
				glog.Infof("%s 禁止交易:%s -> %s %v", strings.ToUpper(h.name), h.bourseA.name, h.bourseB.name, err)
				glog.Infoln(strings.ToUpper(h.name), "profit:", fmt.Sprintf("%0.4f", profit*amount), strings.ToUpper(h.bourseA.name), "sell:", priceA.Buy, strings.ToUpper(h.bourseB.name), "buy:", priceB.Sell)
				return
			}
		} else if !h.right { //仓位正常 且向右开关关的
			h.right = true
			glog.Infof("%s 恢复交易:  %s -> %s", strings.ToUpper(h.name), h.bourseA.name, h.bourseB.name)
		}

		earn, err := h.hedging(h.bourseA, h.bourseB, priceA, priceB, amount)
		if err == nil {
			if h.huidu {
				glog.Infoln(strings.ToUpper(h.name), "profit:", fmt.Sprintf("%0.4f", profit*amount), strings.ToUpper(h.bourseA.name), "sell:", priceA.Buy, strings.ToUpper(h.bourseB.name), "buy:", priceB.Sell)
			} else {
				glog.Infoln(strings.ToUpper(h.name), "earn:", fmt.Sprintf("%0.4f", earn*amount), strings.ToUpper(h.bourseA.name), "sell:", priceA.Buy, strings.ToUpper(h.bourseB.name), "buy:", priceB.Sell)
				h.income.all += earn * amount
				h.income.cur += earn * amount
			}
		} else {
			glog.Errorln(strings.ToUpper(h.name), "hedging err", err)
		}
	} else if profit := mypro.Earn(priceB.Buy, h.bourseB.fee, priceA.Sell, h.bourseA.fee); profit > h.leftEarn {
		if err := h.checkAccount(h.bourseB, h.bourseA, priceA, amount); err != nil {
			if h.left { //右边搬空 且向左开关开的
				glog.Infof("err %s 停止交易: %s -> %s %v", strings.ToUpper(h.name), h.bourseB.name, h.bourseA.name, err)
				h.left = false
				return
			} else { //且向左开关关的
				glog.Infof("%s 禁止交易:%s -> %s %v", strings.ToUpper(h.name), h.bourseB.name, h.bourseA.name, err)
				glog.Infoln(strings.ToUpper(h.name), "profit:", fmt.Sprintf("%0.4f", profit*amount), strings.ToUpper(h.bourseB.name), "sell:", priceB.Buy, strings.ToUpper(h.bourseA.name), "buy:", priceA.Sell)
				return
			}
		} else if h.left { //仓位正常 且向左开关关的
			h.left = true
			glog.Infof("err %s 恢复交易:  %s -> %s", strings.ToUpper(h.name), h.bourseB.name, h.bourseA.name)
		}

		earn, err := h.hedging(h.bourseB, h.bourseA, priceB, priceA, amount)
		if err == nil {
			if h.huidu {
				glog.Infoln(strings.ToUpper(h.name), "profit:", fmt.Sprintf("%0.4f", profit*amount), strings.ToUpper(h.bourseB.name), "sell:", priceB.Buy, strings.ToUpper(h.bourseA.name), "buy:", priceA.Sell)
			} else {
				glog.Infoln(strings.ToUpper(h.name), "earn:", fmt.Sprintf("%0.4f", earn*amount), strings.ToUpper(h.bourseB.name), "sell:", priceB.Buy, strings.ToUpper(h.bourseA.name), "buy:", priceA.Sell)
				h.income.all += earn * amount
				h.income.cur += earn * amount
			}
		} else {
			glog.Errorln(strings.ToUpper(h.name), "hedging err", err)
		}
	}
}

func (h *Hedge) getDepth(bou bourse.Bourse, depth float64) *proto.Price {
	var price *proto.Price
	var err error
	if price, err = bou.GetPriceOfDepth(50, depth, mypro.ConvertCurrencyPair(h.coin)); err != nil {
		glog.Infoln("getdepth err", err)
	}

	for err != nil {
		if price, err = bou.GetPriceOfDepth(50, depth, mypro.ConvertCurrencyPair(h.coin)); err != nil {
			glog.Infoln("getdepth err", err)
		}
	}
	return price
}

func (h *Hedge) getAccount() {
	account, err := h.bourseA.bourse.GetAccount()
	if err != nil {
		glog.Infof("err get account err %s:%v", h.bourseA.name, err)
	} else {
		h.bourseA.cny = account.SubAccounts[mypro.CNY].Available
		h.bourseA.coin = account.SubAccounts[h.coin].Available
	}

	account, err = h.bourseB.bourse.GetAccount()
	if err != nil {
		glog.Infof("err get account err %s:%v", h.bourseB.name, err)
	} else {
		h.bourseB.cny = account.SubAccounts[mypro.CNY].Available
		h.bourseB.coin = account.SubAccounts[h.coin].Available
	}
}

func (h *Hedge) checkAccount(sellSide, buySide account, pirceB *proto.Price, amount float64) error {
	glog.Infof("account %s cny:%f %s:%f", h.bourseA.name, h.bourseA.cny, h.name, h.bourseA.coin) //debug
	glog.Infof("account %s cny:%f %s:%f", h.bourseB.name, h.bourseB.cny, h.name, h.bourseB.coin)
	if sellSide.coin < amount*2 {
		return fmt.Errorf("%s:%s余额不足:%f cny:%f", sellSide.name, h.name, sellSide.coin, sellSide.cny)
	} else if buySide.cny < (pirceB.Sell * (amount * 2)) {
		return fmt.Errorf("%s:cny余额不足:%f %s:%f", buySide.name, buySide.cny, h.name, buySide.coin)
	}
	return nil
}

func (h *Hedge) hedging(sellSide, buySide account, priceA, priceB *proto.Price, amount float64) (float64, error) {
	if h.huidu {
		glog.Infoln(strings.ToUpper(h.name), "huidu on")
		return 0, nil
	}
	if sellSide.name == h.first {
		order, err := h.sell(sellSide.bourse, amount, priceA.Buy, false)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", sellSide.name, proto.SELL, err)
		}
		glog.Infoln("sell", sellSide.name, priceA.Buy, amount, order, order.OrderID, order.Status)

		order, err = h.buy(buySide, amount, priceB.Sell, true)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", buySide.name, proto.BUY, err)
		}
		glog.Infoln("buy:", buySide.name, priceB.Sell, amount, order, order.OrderID, order.Status)
	}

	if buySide.name == h.first {
		order, err := h.buy(buySide, amount, priceB.Sell, false)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", buySide.name, proto.BUY, err)
		}
		glog.Infoln(strings.ToUpper(h.name), "buy:", buySide.name, priceB.Sell, amount, order, order.OrderID, order.Status)

		order, err = h.sell(sellSide.bourse, amount, priceA.Buy, true)
		if err != nil {
			return 0, fmt.Errorf("%s:%s %v", sellSide.name, proto.SELL, err)
		}
		glog.Infoln(strings.ToUpper(h.name), "sell", sellSide.name, priceA.Buy, amount, order, order.OrderID, order.Status)
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
		return nil, fmt.Errorf("err:%v", err)
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
		return nil, fmt.Errorf("err:%v", err)
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
			glog.Infof("err %s retry %s err %v", strings.ToUpper(h.name), side, err)
		}
		glog.Infoln(strings.ToUpper(h.name), "retryDeal sleep:", sec)
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
	glog.Infof("stop judge:%s ok", h.name)
	return nil
}

func (h *Hedge) SetHuidu(huidu bool) bool {
	h.huidu = huidu
	glog.Infof("set judge:%s huidu:%v ok", h.name, huidu)
	return h.huidu
}

func (h *Hedge) SetDepth(depth float64) float64 {
	h.depth = depth
	glog.Infof("set judge:%s depth:%v ok", h.name, depth)
	return h.depth
}

func (h *Hedge) SetAmount(amount float64) float64 {
	h.amount = amount
	glog.Infof("set judge:%s amount:%v ok", h.name, amount)
	return h.amount
}

func (h *Hedge) SetRightEarn(rightEarn float64) float64 {
	h.rightEarn = rightEarn
	glog.Infof("set judge:%s rightEarn:%v ok", h.name, rightEarn)
	return h.rightEarn
}

func (h *Hedge) SetLeftEarn(leftEarn float64) float64 {
	h.leftEarn = leftEarn
	glog.Infof("set judge:%s leftEarn:%v ok", h.name, leftEarn)
	return h.leftEarn
}

func (h *Hedge) SetTicker(ticker int) string {
	h.ticker = ticker
	h.loop = time.NewTicker(time.Second * time.Duration(ticker))
	glog.Infof("set judge:%s ticker:%v ok", h.name, ticker)
	return fmt.Sprintf("ticker set %d/s ok", ticker)
}

func (h *Hedge) SetFirst(first string) string {
	h.first = strings.ToLower(first)
	glog.Infof("set judge:%s first:%v ok", h.name, first)
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

func (h *Hedge) GetIncome() *mypro.Income {
	return &mypro.Income{
		All: h.income.all,
		Cur: h.income.cur,
	}
}
