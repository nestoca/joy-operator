package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/davidmdm/x/xsync"
	joy "github.com/nestoca/joy/pkg"
	"github.com/nestoca/joy/pkg/helm"
)

type ChartPuller struct {
	Logger  *slog.Logger
	Mutexes *xsync.Map[string, *sync.Mutex]
}

func MakeChartPuller(logger *slog.Logger) ChartPuller {
	return ChartPuller{
		Logger:  logger,
		Mutexes: new(xsync.Map[string, *sync.Mutex]),
	}
}

func (puller ChartPuller) Pull(ctx context.Context, opts helm.PullOptions) error {
	var buffer bytes.Buffer

	cli := helm.CLI{IO: joy.IO{Out: &buffer, Err: &buffer}}

	url, _ := opts.Chart.ToURL()
	mutex, _ := puller.Mutexes.LoadOrStore(url.String(), new(sync.Mutex))

	mutex.Lock()
	defer mutex.Unlock()

	if entries, err := os.ReadDir(opts.OutputDir); err == nil && len(entries) > 0 {
		// If the output directory exists and has content in it,
		// then it has been pulled by another goroutine: no need to pull the chart
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to stat chart cache: %w", err)
	}

	if err := cli.Pull(ctx, opts); err != nil {
		return fmt.Errorf("%w: %q", err, &buffer)
	}

	puller.Logger.Info("successfully pulled chart", "url", url.String())

	return nil
}
