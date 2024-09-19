package main

import (
	"github.com/burp_junior/internal/rest/routers"
)

func main() {
	go func() {
		routers.MountProxyRouter()
	}()

	routers.MountAPIRouter()
}
