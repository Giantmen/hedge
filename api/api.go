package api

import (
	"github.com/Giantmen/hedge/log"
	"github.com/Giantmen/hedge/store"
	"github.com/solomoner/gozilla"
)

func Register(qs *store.Service) {
	gozilla.RegisterService(qs, "trader")
	log.Debug("register", "qs")
}
