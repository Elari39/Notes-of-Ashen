package main

import (
	"flag"
	"fmt"
	"os"

	_ "go.uber.org/automaxprocs"

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
	logx.Must(c.ValidateConfig())

	// go-zero 的内置访问日志会在 5xx 时转储完整请求，其中可能包含认证凭证和 API Key。
	// 关闭内置实现，统一使用项目的安全访问日志中间件。
	c.RestConf.Middlewares.Log = false
	server := rest.MustNewServer(c.RestConf)

	ctx := svc.NewServiceContext(c)
	// defer 为 LIFO：先注册 ctx.Close（后执行），再注册 server.Stop（先执行），
	// 确保优雅停机（处理完在途请求）后再关闭下游 DB/Redis/MQ 资源池。
	defer ctx.Close()
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
