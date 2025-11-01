// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"flag"
	"fmt"

	"nof0-api/internal/config"
	"nof0-api/internal/handler"
	"nof0-api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/nof0.yaml", "the config file")

func main() {
	flag.Parse()

	cfg := config.MustLoad(*configFile)

	server := rest.MustNewServer(cfg.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(*cfg, *configFile)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", cfg.Host, cfg.Port)
	server.Start()
}
