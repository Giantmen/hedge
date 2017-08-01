package main

import (
	"flag"
	"fmt"
	stdlog "log"

	"github.com/Giantmen/hedge/config"
	"github.com/Giantmen/hedge/judge"
	"github.com/Giantmen/hedge/log"
	"github.com/Giantmen/hedge/store"

	"github.com/BurntSushi/toml"
	"github.com/solomoner/gozilla"
)

var (
	cfgPath = flag.String("config", "config.toml", "config file path")
)

func initLog(cfg *config.Config) {
	log.SetLevelByString(cfg.LogLevel)
	if !cfg.Debug {
		log.SetHighlighting(false)
		err := log.SetOutputByName(cfg.LogPath)
		if err != nil {
			log.Fatal(err)
		}
		log.SetRotateByDay()
	}
}

func main() {
	flag.Parse()
	var cfg config.Config
	_, err := toml.DecodeFile(*cfgPath, &cfg)
	if err != nil {
		stdlog.Fatal("DecodeConfigFile error: ", err)
	}
	initLog(&cfg)

	bourse, err := store.NewService(&cfg)
	if err != nil {
		log.Error("NewService err", err)
	}

	rule, err := judge.NewJudge(&cfg, bourse)
	if err != nil {
		panic(fmt.Sprintf("NewJudge err %v", err))
	}
	gozilla.RegisterService(rule, "judge")
	log.Debug("register", "judge")
	rule.Process()
	defer rule.StopAll()

	gozilla.DefaultLogOpt.Format += " {{.Body}}"
	stdlog.Fatal(gozilla.ListenAndServe(cfg.Listen))
}
