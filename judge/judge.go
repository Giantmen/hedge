package judge

import (
	"fmt"
	"strings"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/log"
	"github.com/Giantmen/hedge/proto"
	"github.com/Giantmen/hedge/store"

	"github.com/solomoner/gozilla"
)

type Processer interface {
	Process() error
	Stop() error
	Status() bool
	SetHuidu(huidu bool) bool
	SetDepth(depth float64) float64
	SetAmount(amount float64) float64
	SetRightEarn(rightEarn float64) float64
	SetLeftEarn(leftEarn float64) float64
	SetTicker(ticker int) string
	SetFirst(first string) string
	GetConfig() *proto.ConfigReply
}

type Judge struct {
	judges map[string]Processer
}

func NewJudge(cfg *config.Config, sr *store.Service) (*Judge, error) {
	var judges = make(map[string]Processer)
	for _, c := range cfg.Judge {
		switch c.Name {
		case "etc_btctrade_chbtc":
			if etc_btctrade_chbtc, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = etc_btctrade_chbtc
			}
			// case "eth_btctrade_chbtc": //策略name
			// 	if eth_btctrade_chbtc, err := NewHedge(&c, sr); err != nil {
			// 		return nil, err
			// 	} else {
			// 		judges[strings.ToUpper(c.Name)] = eth_btctrade_chbtc
			// 	}
		}
	}

	return &Judge{
		judges: judges,
	}, nil
}

func (j *Judge) Process() {
	for name, judge := range j.judges {
		go judge.Process()
		log.Info("judge", name, "start", "ok")
	}
}

func (j *Judge) StopAll() {
	for name, judge := range j.judges {
		judge.Stop()
		log.Info("judge", name, "stop", "ok")
	}
}

func (j *Judge) Start(ctx *gozilla.Context, r *proto.JudgeQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	if ju.Status() {
		log.Errorf("%s is already start", r.Judge)
		return "", fmt.Errorf("%s is already start", r.Judge)
	} else {
		go ju.Process()
		return fmt.Sprintf("judge:%s start ok", r.Judge), nil
	}
}

func (j *Judge) Stop(ctx *gozilla.Context, r *proto.JudgeQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	if !ju.Status() {
		log.Errorf("%s is already stop", r.Judge)
		return "", fmt.Errorf("%s is already stop", r.Judge)
	} else {
		go ju.Stop()
		return fmt.Sprintf("judge:%s stop ok", r.Judge), nil
	}
}

func (j *Judge) Status(ctx *gozilla.Context, r *proto.JudgeQuery) (bool, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return false, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.Status(), nil
}

func (j *Judge) SetHuidu(ctx *gozilla.Context, r *proto.HuiduQuery) (bool, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return false, fmt.Errorf("get %s err", r.Judge)
	}
	log.Debug("huidu", r.Judge, r.Value)
	return ju.SetHuidu(r.Value), nil
}

func (j *Judge) SetDepth(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetDepth(r.Value), nil
}

func (j *Judge) SetAmount(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetAmount(r.Value), nil
}

func (j *Judge) SetRightEarn(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetRightEarn(r.Value), nil
}

func (j *Judge) SetLeftEarn(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetLeftEarn(r.Value), nil
}

func (j *Judge) SetTicker(ctx *gozilla.Context, r *proto.ConfigQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	if int(r.Value) < 1 {
		return "", fmt.Errorf("set ticker err %v < 1", r.Value)
	}
	return ju.SetTicker(int(r.Value)), nil
}

func (j *Judge) SetFirst(ctx *gozilla.Context, r *proto.FirstQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetFirst(r.Value), nil
}

func (j *Judge) GetConfig(ctx *gozilla.Context, r *proto.JudgeQuery) (*proto.ConfigReply, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		log.Errorf("get %s err", r.Judge)
		return nil, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.GetConfig(), nil
}
