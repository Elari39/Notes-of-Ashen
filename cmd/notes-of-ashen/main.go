package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	_ "go.uber.org/automaxprocs"

	"notes-of-ashen/internal/config"
	"notes-of-ashen/internal/handler"
	backuplogic "notes-of-ashen/internal/logic/backup"
	"notes-of-ashen/internal/security"
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
	defer ctx.Close()
	// 在开始接受请求前，完成已提交恢复的媒体目录发布，或回滚尚未提交的暂存目录。
	// 这使 MySQL 中的恢复标记和媒体文件系统在进程异常退出后重新收敛。
	logx.Must(recoverPendingRestore(ctx))
	// defer 为 LIFO：先注册 ctx.Close（后执行），再注册 server.Stop（先执行），
	// 确保优雅停机（处理完在途请求）后再关闭下游 DB/Redis/MQ 资源池。
	defer server.Stop()

	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

func recoverPendingRestore(svcCtx *svc.ServiceContext) error {
	lease, restoreCtx, err := security.AcquireRestoreLease(context.Background(), svcCtx.Redis)
	if err != nil {
		if errors.Is(err, security.ErrRestoreLeaseHeld) {
			// 另一实例仍持有恢复租约时，其 staging 目录可能仍在写入；不能由
			// 新启动实例清理。维护中间件会继续依据 Redis 标记拦截写请求。
			logx.Info("[startup] backup restore lease is held; skip pending restore recovery")
			return nil
		}
		return fmt.Errorf("acquire backup restore lease for startup recovery: %w", err)
	}
	defer func() {
		if releaseErr := lease.Release(); releaseErr != nil {
			logx.Errorf("release startup restore lease failed: %v", releaseErr)
		}
	}()
	return backuplogic.RecoverPendingRestore(restoreCtx, svcCtx)
}
