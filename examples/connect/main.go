package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"
	"tarantool"
	"tarantool/debug"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"

	"golang.org/x/sync/errgroup"
)

func main() {
	opts := struct {
		DebugListen string `long:"debug.listen" env:"EDITOR_DEBUG_LISTEN" default:":6060" description:"Interface for serve debug information(metrics/health/pprof)"`
		Verbose     bool   `short:"v" env:"VERBOSE" description:"Enable verbose log output"`

		TTCluster []string `long:"tt.cluster" env:"TT_CLUSTER" description:"Hosts to tarantool cluster"`
		TTUser    string   `long:"tt.user" env:"TT_USER" description:"Username to auth at tarantool cluster"`
		TTPass    string   `long:"tt.pass" env:"TT_PASS" description:"Password to auth at tarantool cluster"`
	}{}

	_, err := flags.Parse(&opts)
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
		os.Exit(1)
	}

	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARNING", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	if opts.Verbose {
		filter.SetMinLevel(logutils.LogLevel("DEBUG"))
	}

	logger := log.New(filter, "", log.Ldate|log.Ltime|log.LUTC)
	logger.Printf("[INFO] Launching Application with: %+v", opts)

	d := debug.New()
	gr, ctx := errgroup.WithContext(context.Background())
	ctx, cancel := context.WithCancel(ctx)

	gr.Go(func() error {
		return d.Serve(ctx, opts.DebugListen, logger)
	})

	ErrCanceled := errors.New("canceled")
	gr.Go(func() error {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		for {
			select {
			case <-ctx.Done():
				return ErrCanceled
			case <-sigs:
				cancel()
				logger.Printf("[INFO] Caught stop signal. Exiting ...")
			}
		}
	})

	gr.Go(func() error {

		t, err := tarantool.New(opts.TTUser, opts.TTPass)
		if err != nil {
			return err
		}
		err = t.ConnectTo(ctx, opts.TTCluster)
		if err != nil {
			return err
		}

		logger.Printf("[DEBUG] connected: %+v", t)

		err = t.Close()
		if err != nil {
			return err
		}

		cancel()
		return nil
	})

	logger.Printf("[DEBUG] wait group...")
	err = gr.Wait()
	if err != nil {
		logger.Fatalf("[ERROR] shutdown error: %v", err)
	}
	logger.Printf("[DEBUG] goodbye!")
}
