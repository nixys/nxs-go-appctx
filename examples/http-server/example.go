package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/nixys/nxs-go-appctx"
	"github.com/sirupsen/logrus"
)

type selfContext struct {
	conf confOpts
}

type httpServerContext struct {
	http.Server
	done chan interface{}
}

var log *logrus.Logger

func main() {

	var (
		ctx selfContext
		err error
	)

	// Read command line arguments
	args := argsRead()

	appCtx := appctx.AppContext{
		AppCtx:           &ctx,
		CfgPath:          args.configPath,
		CtxInit:          contextInit,
		CtxReload:        contextReload,
		CtxFree:          contextFree,
		TermSignals:      []os.Signal{syscall.SIGTERM, syscall.SIGINT},
		ReloadSignals:    []os.Signal{syscall.SIGHUP},
		LogrotateSignals: []os.Signal{syscall.SIGUSR1},
	}

	log, err = appCtx.ContextInit()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	log.Info("program started")

	// Channel to notify main() when goroutine done
	grChan := make(chan int)

	defer func() {

		for {
			select {

			// Wait for program termination
			case ec := <-appCtx.ExitWait():

				log.WithFields(logrus.Fields{
					"exit code": ec,
				}).Info("program terminating")

				// Done the appctx
				appCtx.ContextDone()

				// Exit from program with `ec` status
				os.Exit(ec)

			// Wait for goroutine is done
			case s := <-grChan:

				log.WithFields(logrus.Fields{
					"goroutine exit code": s,
				}).Debug("goroutine done")

				appCtx.ContextTerminate(s)
			}
		}
	}()

	// Create main context
	c := context.Background()

	// Create derived context for goroutine
	cRuntime, cf := context.WithCancel(c)

	// Add a goroutine element into appctx
	crc := appCtx.RoutineAdd(cf)

	go func() {
		defer appCtx.RoutineDone(crc)
		runtime(cRuntime, ctx, crc, grChan)
	}()
}

func servStart(ctx selfContext) *httpServerContext {

	s := &httpServerContext{
		Server: http.Server{
			Addr:         ctx.conf.Bind,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			Handler:      epRoutesSet(ctx),
		},
		done: make(chan interface{}),
	}

	go func() {
		log.Debugf("server status: starting")
		if err := s.ListenAndServe(); err != nil {
			log.Debugf("server status: %v", err)
		}
		s.done <- true
	}()

	return s
}

func servShutdown(s *httpServerContext) error {

	//Create shutdown context with 10 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	//Shutdown the server
	if err := s.Shutdown(ctx); err != nil {
		return err
	}

	<-s.done
	return nil
}

func servRestart(ctx selfContext, s *httpServerContext) (*httpServerContext, error) {

	if err := servShutdown(s); err != nil {
		return nil, err
	}

	return servStart(ctx), nil
}

func runtime(cRuntime context.Context, ctx selfContext, crc chan interface{}, grChan chan int) {

	var err error

	s := servStart(ctx)

	for {
		select {
		case <-cRuntime.Done():
			// Program termination.
			err = servShutdown(s)
			if err != nil {
				log.Errorf("http server shutdown error: %v", err)
			}
			return
		case c := <-crc:
			// Updated context application data.
			// Set the new one in current goroutine.
			ctx = *(c.(*selfContext))
			s, err = servRestart(ctx, s)
			if err != nil {
				log.Errorf("http server reload error: %v", err)
				grChan <- 1
				return
			}
		}
	}
}
