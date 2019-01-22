package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/nixys/nxs-go-appctx"
	"github.com/sirupsen/logrus"
)

type selfContext struct {
	timeInterval int
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

	defer func() {

		// Wait for program termination
		ec := appCtx.ExitWait()

		log.WithFields(logrus.Fields{
			"exit code": ec,
		}).Info("program terminating")

		// Done the appctx
		appCtx.ContextDone()

		// Exit from program with `ec` status
		os.Exit(ec)
	}()

	// Create main context
	c := context.Background()

	// Create derived context for goroutine
	cRuntime, cf := context.WithCancel(c)

	// Add a goroutine element into appctx
	crc := appCtx.RoutineAdd(cf)

	// Do the same for second goroutine
	cRuntime2, cf2 := context.WithCancel(c)
	crc2 := appCtx.RoutineAdd(cf2)

	go func() {
		defer appCtx.RoutineDone(crc)
		runtime(cRuntime, ctx, crc)
	}()

	go func() {
		defer appCtx.RoutineDone(crc2)
		runtime2(cRuntime2, ctx, crc2)
	}()
}

func runtime(cRuntime context.Context, ctx selfContext, crc chan interface{}) {

	timer := time.NewTimer(time.Duration(ctx.timeInterval) * time.Second)

	for {
		select {
		case <-timer.C:
			// Do the some actions
			log.WithFields(logrus.Fields{
				"time interval": ctx.timeInterval,
			}).Info("Time to work!")
			timer.Reset(time.Duration(ctx.timeInterval) * time.Second)
		case <-cRuntime.Done():
			// Termination the program.
			// Write "Done" to log and complete the current goroutine.
			log.Info("Done")
			return
		case c := <-crc:
			// Updated context application data.
			// Set the new one in current goroutine.
			ctx = *(c.(*selfContext))
		}
	}
}

// Do same actions as `runtime()`, but adds one additional second to timer
func runtime2(cRuntime context.Context, ctx selfContext, crc chan interface{}) {

	timer := time.NewTimer(time.Duration(ctx.timeInterval+1) * time.Second)

	for {
		select {
		case <-timer.C:
			// Do the some actions
			log.WithFields(logrus.Fields{
				"time interval": ctx.timeInterval + 1,
			}).Info("Time to work! [2]")
			timer.Reset(time.Duration(ctx.timeInterval+1) * time.Second)
		case <-cRuntime.Done():
			// Termination the program.
			// Write "Done" to log and complete the current goroutine.
			log.Info("Done [2]")
			return
		case c := <-crc:
			// Updated context application data.
			// Set the new one in current goroutine.
			ctx = *(c.(*selfContext))
		}
	}
}
