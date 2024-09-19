package main

import (
	"github.com/burp_junior/internal/rest/routers"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		return
	}

	go func() {
		routers.MountProxyRouter(logger)
	}()

	routers.MountAPIRouter(logger)
}
