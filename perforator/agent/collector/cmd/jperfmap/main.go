package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmattach"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/pidfd"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func main() {

	err := mainImpl(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func mainImpl(ctx context.Context) error {
	pid := flag.Int("pid", -1, "Target")
	flag.Parse()
	if *pid == -1 {
		return fmt.Errorf("pid is required")
	}
	logger := xlog.New(zap.Must(zap.ConsoleConfig(log.DebugLevel)))

	pfd, err := pidfd.Open(linux.ProcessID(*pid))
	if err != nil {
		return fmt.Errorf("failed to open pidfd: %w", err)
	}

	d := &jvmattach.Dialer{
		Logger: logger,
	}
	dialCtx, cancelDialCtx := context.WithTimeoutCause(ctx, 3*time.Second, fmt.Errorf("dial timeout (3s) exceeded"))
	defer cancelDialCtx()
	// TODO: we assume that nspid == pid, i.e. target process is not namespaced
	conn, err := d.Dial(dialCtx, jvmattach.Target{
		ProcessFD:     pfd,
		PID:           linux.ProcessID(*pid),
		NamespacedPID: linux.ProcessID(*pid),
	})
	if err != nil {
		return fmt.Errorf("failed to dial: %w", err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			logger.Warn(ctx, "Cleanup error", log.Error(err))
		}
	}()
	resp, err := conn.Execute(ctx, [4]string{"jcmd", "Compiler.perfmap"})
	if err != nil {
		return fmt.Errorf("failed to dump perfmap: %w", err)
	}
	fmt.Println(resp)

	return nil
}
