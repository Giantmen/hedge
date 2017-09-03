package judge

import (
	"fmt"
	"strings"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/proto"
	"github.com/Giantmen/hedge/store"

	"github.com/golang/glog"
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
	GetIncome() *proto.Income
}

type Judge struct {
	judges map[string]Processer
}

func NewJudge(cfg *config.Config, sr *store.Service) (*Judge, error) {
	var judges = make(map[string]Processer)
	for _, c := range cfg.Judge {
		switch c.Name {
		case "etc_chbtc_huobiN":
			if etc_chbtc_huobiN, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = etc_chbtc_huobiN
			}

		case "eth_chbtc_huobiN":
			if eth_chbtc_huobiN, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = eth_chbtc_huobiN
			}

		case "snt_yunbi_bter":
			if snt_yunbi_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = snt_yunbi_bter
			}

		case "omg_yunbi_bter":
			if omg_yunbi_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = omg_yunbi_bter
			}

		case "pay_yunbi_bter":
			if pay_yunbi_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = pay_yunbi_bter
			}

			//etc
		case "etc_chbtc_bter":
			if etc_chbtc_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = etc_chbtc_bter
			}

		case "etc_yunbi_chbtc":
			if etc_yunbi_chbtc, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = etc_yunbi_chbtc
			}
		case "etc_yunbi_bter":
			if etc_yunbi_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = etc_yunbi_bter
			}

			//eos
		case "eos_chbtc_bter":
			if eos_chbtc_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = eos_chbtc_bter
			}

		case "eos_yunbi_chbtc":
			if eos_yunbi_chbtc, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = eos_yunbi_chbtc
			}

		case "eos_yunbi_bter":
			if eos_yunbi_bter, err := NewHedge(&c, sr); err != nil {
				return nil, err
			} else {
				judges[strings.ToUpper(c.Name)] = eos_yunbi_bter
			}
		}
	}

	return &Judge{
		judges: judges,
	}, nil
}

func (j *Judge) Process() {
	for name, judge := range j.judges {
		go judge.Process()
		glog.Infoln("judge", name, "start", "ok")
	}
}

func (j *Judge) StopAll() {
	for name, judge := range j.judges {
		judge.Stop()
		glog.Infoln("judge", name, "stop", "ok")
	}
}

func (j *Judge) Start(ctx *gozilla.Context, r *proto.JudgeQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	if ju.Status() {
		glog.Errorf("%s is already start", r.Judge)
		return "", fmt.Errorf("%s is already start", r.Judge)
	} else {
		go ju.Process()
		return fmt.Sprintf("judge:%s start ok", r.Judge), nil
	}
}

func (j *Judge) Stop(ctx *gozilla.Context, r *proto.JudgeQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	if !ju.Status() {
		glog.Errorf("%s is already stop", r.Judge)
		return "", fmt.Errorf("%s is already stop", r.Judge)
	} else {
		go ju.Stop()
		return fmt.Sprintf("judge:%s stop ok", r.Judge), nil
	}
}

func (j *Judge) Status(ctx *gozilla.Context, r *proto.JudgeQuery) (bool, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return false, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.Status(), nil
}

func (j *Judge) SetHuidu(ctx *gozilla.Context, r *proto.HuiduQuery) (bool, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return false, fmt.Errorf("get %s err", r.Judge)
	}
	glog.Infoln("huidu", r.Judge, r.Value)
	return ju.SetHuidu(r.Value), nil
}

func (j *Judge) SetDepth(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetDepth(r.Value), nil
}

func (j *Judge) SetAmount(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetAmount(r.Value), nil
}

func (j *Judge) SetRightEarn(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetRightEarn(r.Value), nil
}

func (j *Judge) SetLeftEarn(ctx *gozilla.Context, r *proto.ConfigQuery) (float64, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return 0, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetLeftEarn(r.Value), nil
}

func (j *Judge) SetTicker(ctx *gozilla.Context, r *proto.ConfigQuery) (string, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
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
		glog.Errorf("get %s err", r.Judge)
		return "", fmt.Errorf("get %s err", r.Judge)
	}
	return ju.SetFirst(r.Value), nil
}

func (j *Judge) GetConfig(ctx *gozilla.Context, r *proto.JudgeQuery) (*proto.ConfigReply, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return nil, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.GetConfig(), nil
}

func (j *Judge) GetIncome(ctx *gozilla.Context, r *proto.JudgeQuery) (*proto.Income, error) {
	ju, ok := j.judges[strings.ToUpper(r.Judge)]
	if !ok {
		glog.Errorf("get %s err", r.Judge)
		return nil, fmt.Errorf("get %s err", r.Judge)
	}
	return ju.GetIncome(), nil
}
