package main

import (
	"flag"
	"fmt"
	stdlog "log"
	"time"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/judge"
	"github.com/Giantmen/hedge/store"

	"github.com/BurntSushi/toml"
	"github.com/golang/glog"
	"github.com/solomoner/gozilla"
)

var (
	cfgPath = flag.String("config", ".config.toml", "config file path")
)

func flushLog() {
	for {
		glog.Flush()
		time.Sleep(2 * time.Second)
	}
}

func main() {
	flag.Parse()
	go flushLog()
	var cfg config.Config
	_, err := toml.DecodeFile(*cfgPath, &cfg)
	if err != nil {
		stdlog.Fatal("DecodeConfigFile error: ", err)
	}

	bourse, err := store.NewService(&cfg)
	if err != nil {
		glog.Errorln("NewService err", err)
	}

	rule, err := judge.NewJudge(&cfg, bourse)
	if err != nil {
		panic(fmt.Sprintf("NewJudge err %v", err))
	}
	gozilla.RegisterService(rule, "judge")
	glog.Infoln("register", "judge")
	rule.Process()
	defer rule.StopAll()

	gozilla.DefaultLogOpt.Format += " {{.Body}}"
	stdlog.Fatal(gozilla.ListenAndServe(cfg.Listen))
}
