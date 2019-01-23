package appctx

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	chanToMain   = 0
	chanFromMain = 1

	exitStatusSuccess = 0
	exitStatusFailure = 1
)

// ContextInit it is a function type that will be called on application context init
type ContextInit func(ctx interface{}, cfgPath string) (CfgData, error)

// ContextFree it is a function type that will be called on application context free
type ContextFree func(ctx interface{}) int

// ContextReload it is a function type that will be called on application context reload
type ContextReload func(ctx interface{}, cfgPath string) (CfgData, error)

// AppContext contains the dataset to create appctx context
type AppContext struct {

	// Application context data pointer
	AppCtx interface{}

	// Function to init application context data
	CtxInit ContextInit

	// Function to reload application context data
	CtxReload ContextReload

	// Function to free application context data
	CtxFree ContextFree

	// Path to config file
	CfgPath string

	// Application termination signals set
	TermSignals []os.Signal

	// Application reload signals set
	ReloadSignals []os.Signal

	// Application logrotate signals set
	LogrotateSignals []os.Signal

	// cfgData contains
	cfgData CfgData

	// termChan is a channels array to send exit codes from appctx to main()
	// and vice versa
	termChan [2]chan int

	// wg is a wait group for application runtime goroutines
	wg sync.WaitGroup

	routines   []routineElt
	routinesMu sync.Mutex

	log *logrus.Logger
}

// CfgData contains the generic config options, that are used in context processing.
// It also used to receive data from context init and reload functions.
type CfgData struct {
	PidFile  string
	LogFile  string
	LogLevel string
}

type routineElt struct {

	// Routine's cancel function.
	// When program retrieve the termination signal each cancel function will be called to notify goroutine to done
	cf context.CancelFunc

	// Routine's `context reload channel` which used to send to receivers
	// updated context.
	crc chan interface{}
}

// ContextInit initializes application context and fills global `ctx` and `log` variables
// in accordance with data from conguration file.
func (c *AppContext) ContextInit() (*logrus.Logger, error) {

	// If CtxInit is set
	if c.CtxInit != nil {

		d, err := c.CtxInit(c.AppCtx, c.CfgPath)
		if err != nil {
			return nil, err
		}

		c.cfgData = d
	}

	// Initialize logging
	l, err := LogfileInit(c.cfgData.LogFile, c.cfgData.LogLevel, c.LogrotateSignals)
	if err != nil {
		return nil, err
	}
	c.log = l

	// Create pidfile if path is specified
	if err := PidfileCreate(c.cfgData.PidFile); err != nil {
		return nil, err
	}

	// from appctx to main
	c.termChan[chanToMain] = make(chan int)

	// from main to appctx
	c.termChan[chanFromMain] = make(chan int)

	// Set context reload signals processing
	c.setReloadSignals()

	// Set app termination signals processing
	c.setTermSignals()

	// Init application context

	return c.log, nil
}

// ContextDone completes the appctx.
// This method must be called before main() function Exit to complete write operations (e.g. log file)
// and close channes.
func (c *AppContext) ContextDone() {

	// Close log file
	LogfileClose(c.log)

	// Close the channels
	close(c.termChan[chanToMain])
	close(c.termChan[chanFromMain])
}

// ContextTerminate generates the termination signal
//
// This function must ba called from main() to initiate the context free
// and program termination (e.g. after one of the goroutines done or failed).
//
// After this function is called and exit status sent, the appctx will
// notified all derived goroutines to terminate, freed application context and return
// exit status back to main() via termChan[chanToMain].
func (c *AppContext) ContextTerminate(status int) {
	c.termChan[chanFromMain] <- status
}

// RoutineAdd adds a new routine element into appctx.
// For each element the cancel function is set and new context reload channel creates.
// Important: all goroutines that are created using RoutineAdd() must call RoutineDone() when completes.
func (c *AppContext) RoutineAdd(cf context.CancelFunc) chan interface{} {

	c.routinesMu.Lock()
	defer c.routinesMu.Unlock()

	crc := make(chan interface{})

	c.routines = append(c.routines, routineElt{
		cf:  cf,
		crc: crc,
	})

	c.wg.Add(1)

	return crc
}

// RoutineDone calls the appctx waitGroup Done() function when goroutine completes
func (c *AppContext) RoutineDone(crc chan interface{}) {

	c.routinesMu.Lock()
	defer c.routinesMu.Unlock()

	for i, e := range c.routines {
		if e.crc == crc {
			close(e.crc)
			c.routines = append(c.routines[:i], c.routines[i+1:]...)
		}
	}

	c.wg.Done()
}

// ExitWait waits the exit code
// This function must be called in main() to notified program termination
func (c *AppContext) ExitWait() chan int {
	return c.termChan[chanToMain]
}

// setTermSignals sets the application termination signals processing.
// Also it waits the exit status from main()
func (c *AppContext) setTermSignals() {

	// Termination signals processing
	sigChan := make(chan os.Signal, 1)

	go func() {
		signal.Notify(sigChan, c.TermSignals...)
		for {
			select {
			case s := <-sigChan:
				c.log.WithFields(logrus.Fields{
					"signal": s,
				}).Debug("got terminating signal")
				c.terminate(exitStatusSuccess)
			case s := <-c.termChan[chanFromMain]:
				c.terminate(s)
			}
		}
	}()
}

// setReloadSignals sets the application reload signals processing
func (c *AppContext) setReloadSignals() {

	// Context reload signals processing
	sigChan := make(chan os.Signal, 1)

	go func() {
		signal.Notify(sigChan, c.ReloadSignals...)
		for s := range sigChan {

			c.log.WithFields(logrus.Fields{
				"signal": s,
			}).Debug("got reloading signal")

			if c.CtxReload == nil {
				continue
			}

			d, err := c.CtxReload(c.AppCtx, c.CfgPath)
			if err != nil {
				c.terminate(exitStatusFailure)
				continue
			}

			// Save old values
			o := c.cfgData

			// Set new values
			c.cfgData = d

			if c.cfgData.LogFile != o.LogFile {

				// If logfile path has been changed

				if err = LogfileChange(c.log, c.cfgData.LogFile, c.cfgData.LogLevel, c.LogrotateSignals); err != nil {
					c.log.Errorf("context reload error: %v", err)
					c.terminate(exitStatusFailure)
					continue
				}
			} else {
				if c.cfgData.LogLevel != o.LogLevel {

					// If log level has been changed only

					// Validate loglevel
					level, err := logrus.ParseLevel(c.cfgData.LogLevel)
					if err != nil {
						c.log.Errorf("context reload error: wrong loglevel value: %s", c.cfgData.LogLevel)
						c.terminate(exitStatusFailure)
						continue
					}
					c.log.SetLevel(level)
				}
			}

			if err := PidfileChange(o.PidFile, c.cfgData.PidFile); err != nil {
				c.log.Errorf("context reload error: %v", err)
				c.terminate(exitStatusFailure)
				continue
			}

			// Send updated context into specified channels
			for _, r := range c.routines {
				r.crc <- c.AppCtx
			}
		}
	}()
}

// Application termination
func (c *AppContext) terminate(status int) {

	var es int

	// Call all context cancel functions
	for _, r := range c.routines {
		r.cf()
	}

	// Wait for application runtime goroutines done
	c.wg.Wait()

	if c.CtxFree != nil {
		es = c.CtxFree(c.AppCtx)
	} else {
		es = 0
	}

	// If desired status is set
	if status != 0 {
		es = status
	}

	// Remove pid file if necessary
	PidfileRemove(c.cfgData.PidFile)

	// Notify main function with exit code
	c.termChan[chanToMain] <- es
}
