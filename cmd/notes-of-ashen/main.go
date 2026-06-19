package main

import (
	"flag"
	"fmt"
	"os"

	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/handler"
	"notes-of-ashen/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/notes-of-ashen.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// Load .env (if present) so local `go run` picks up remote DB/Redis/MQ
	// addresses without manual shell exports. Real env vars still win.
	if err := config.LoadDotEnv(os.Getenv("APP_ENV_FILE")); err != nil {
		logx.Must(err)
	}
	logx.Must(c.ApplyEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
