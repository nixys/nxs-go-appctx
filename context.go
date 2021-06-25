package appctx

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/sirupsen/logrus"
)

// AppContext contains the dataset to create appctx context
type AppContext struct {

	// Custom context data pointer
	cc CustomContext

	// Pointer to arguments data read from command line
	args interface{}

	// Path to config file
	cfgPath string

	// Application termination signals set
	termSignals []os.Signal

	// Application reload signals set
	reloadSignals []os.Signal

	// Application logrotate signals set
	logrotateSignals []os.Signal

	// cfgData contains
	cfgData CfgData

	// termChanToMain is a channels to send exit codes from appctx to main()
	termChanToMain chan int

	// termChanFromMain is a channels to send exit codes from main() to appctx
	termChanFromMain chan int

	// Channel to notify main() when derived routine done (routine sends exit status via this channel)
	grDoneChan chan int

	// wg is a wait group for application runtime routines
	wg sync.WaitGroup

	routines   []routineElt
	routinesMu sync.Mutex

	log *logrus.Logger
}

// Settings contains setting to init appctx
type Settings struct {

	// Custom context data pointer
	CustomContext CustomContext

	// Pointer to arguments data read from command line
	Args interface{}

	// Path to config file
	CfgPath string

	// Application termination signals set
	TermSignals []os.Signal

	// Application reload signals set
	ReloadSignals []os.Signal

	// Application logrotate signals set
	LogrotateSignals []os.Signal

	// LogFormatter defines log format.
	// If LogFormatter is nil, then will be used nxs-go-appctx` default text log format.
	// To get logs in JSON format use `&logrus.JSONFormatter{}`
	LogFormatter logrus.Formatter
}

// CfgData contains the generic config options, that are used in context processing.
// It also used to receive data from context init and reload functions.
type CfgData struct {
	PidFile  string
	LogFile  string
	LogLevel string
}

// CustomContextFuncOpts contains options to be passed into
// cunstom context `Init`, `Reload` and `Free` functions
type CustomContextFuncOpts struct {
	Args   interface{}
	Config string
	Log    *logrus.Logger
}

// CustomContext is an interface for custom context described by user
type CustomContext interface {
	Init(CustomContextFuncOpts) (CfgData, error)
	Reload(CustomContextFuncOpts) (CfgData, error)
	Free(CustomContextFuncOpts) int
}

// routineElt contains routing element description
type routineElt struct {

	// Routine's cancel function.
	// When program retrieve the termination signal each cancel function will be called to notify routine to done
	cf context.CancelFunc

	// Routine's `context reload channel` which used to send to receivers
	// updated context.
	crc chan interface{}
}

const (
	// ExitStatusSuccess defines success exit status
	ExitStatusSuccess = 0

	// ExitStatusFailure defines failure exit status
	ExitStatusFailure = 1
)

// ContextInit initializes application context and fills global `ctx` and `log` variables
// in accordance with data from conguration file.
func ContextInit(s Settings) (*AppContext, error) {

	var (
		ac  AppContext
		err error
	)

	// Check specified custom context pointer is not nil
	if s.CustomContext == nil {
		return nil, fmt.Errorf("nil custom context")
	}

	// Set context settings
	ac.cc = s.CustomContext
	ac.args = s.Args
	ac.cfgPath = s.CfgPath
	ac.termSignals = s.TermSignals
	ac.reloadSignals = s.ReloadSignals
	ac.logrotateSignals = s.LogrotateSignals

	// Call custom context init function
	ac.cfgData, err = ac.cc.Init(CustomContextFuncOpts{
		Args:   ac.args,
		Config: ac.cfgPath,
		Log: &logrus.Logger{
			Out:       os.Stdout,
			Level:     logrus.DebugLevel,
			Formatter: logFormat(s.LogFormatter),
		},
	})
	if err != nil {
		return nil, err
	}

	// Initialize logging
	l, err := LogfileInit(ac.cfgData.LogFile, ac.cfgData.LogLevel, ac.logrotateSignals, s.LogFormatter)
	if err != nil {
		return nil, err
	}
	ac.log = l

	// Create pidfile if path is specified
	if err := PidfileCreate(ac.cfgData.PidFile); err != nil {
		return nil, err
	}

	// Channel to notify main() when routine done
	ac.grDoneChan = make(chan int)

	// from appctx to main
	ac.termChanToMain = make(chan int)

	// from main to appctx
	ac.termChanFromMain = make(chan int)

	// Set context reload signals processing
	ac.setReloadSignals()

	// Set app termination signals processing
	ac.setTermSignals()

	// Init application context

	return &ac, nil
}

// ContextDone completes the appctx.
// This method must be called before main() function Exit to complete write operations (e.g. log file)
// and close channes.
func (ac *AppContext) ContextDone() {

	// Close log file
	LogfileClose(ac.log)

	// Close the channels
	close(ac.termChanToMain)
	close(ac.termChanFromMain)
}

// ContextTerminate generates the termination signal
//
// This function must ba called from main() to initiate the context free
// and program termination (e.g. after one of the routines done or failed).
//
// After this function is called and exit status sent, the appctx will
// notified all derived routines to terminate, freed application context and return
// exit status back to main() via termChan[chanToMain].
func (ac *AppContext) ContextTerminate(status int) {
	ac.termChanFromMain <- status
}

// RoutineCreate prepares all background data and exec specified runtime function in new routine
// In args:
// - ctx: context derived from main() function
// - runtime: body function for new routine
func (ac *AppContext) RoutineCreate(
	ctx context.Context,
	runtime func(context.Context, *AppContext, chan interface{}),
) {

	if runtime == nil {
		ac.log.Warn("skipping to create new routine: empty runtime function")
		return
	}

	// Create derived context for routine
	ctxRoutine, cf := context.WithCancel(ctx)

	// Add a routine element into appctx
	crc := ac.routineAdd(cf)

	go func() {
		defer ac.routineDone(crc)
		runtime(ctxRoutine, ac, crc)
	}()
}

// ExitWait waits the exit code
// This function must be called in main() to notified program termination
func (ac *AppContext) ExitWait() chan int {
	return ac.termChanToMain
}

// CustomCtx returns appctx custom context
func (ac *AppContext) CustomCtx() interface{} {
	return ac.cc
}

// Log returns appctx logger
func (ac *AppContext) Log() *logrus.Logger {
	return ac.log
}

// RoutineDoneWait returns routine done channel.
// Used in main() for obtain exit status of the derived routine
func (ac *AppContext) RoutineDoneWait() chan int {
	return ac.grDoneChan
}

// RoutineDoneSend sends specified status into routine done channel.
// Used in derived routines for send its exit status to main()
func (ac *AppContext) RoutineDoneSend(status int) {
	ac.grDoneChan <- status
}

// MainBodyGeneric represents generic body used in main().
// If you need to create custom functionality in main() you may use body of this
// funtion as example.
// This function must be call with `defer`
func (ac *AppContext) MainBodyGeneric() {
	for {
		select {

		// Wait for program termination
		case ec := <-ac.ExitWait():

			ac.Log().WithFields(logrus.Fields{
				"exit code": ec,
			}).Info("program terminating")

			// Done the appctx
			ac.ContextDone()

			// Exit from program with `ec` status
			os.Exit(ec)

		// Wait for any of derived routine is done
		case s := <-ac.RoutineDoneWait():

			ac.Log().WithFields(logrus.Fields{
				"exit code": s,
			}).Debug("derived routine done")

			ac.ContextTerminate(s)
		}
	}
}

// routineAdd adds a new routine element into appctx.
// For each element the cancel function is set and new context reload channel creates.
// Important: all routines that are created using RoutineAdd() must call RoutineDone() when completes.
func (ac *AppContext) routineAdd(cf context.CancelFunc) chan interface{} {

	ac.routinesMu.Lock()
	defer ac.routinesMu.Unlock()

	crc := make(chan interface{})

	ac.routines = append(ac.routines, routineElt{
		cf:  cf,
		crc: crc,
	})

	ac.wg.Add(1)

	return crc
}

// routineDone calls the appctx waitGroup Done() function when routine completes.
// Arg `crc` is an context reload channel of the specific routine
func (ac *AppContext) routineDone(crc chan interface{}) {

	ac.routinesMu.Lock()
	defer ac.routinesMu.Unlock()

	for i, e := range ac.routines {
		if e.crc == crc {
			close(e.crc)
			ac.routines = append(ac.routines[:i], ac.routines[i+1:]...)
		}
	}

	ac.wg.Done()
}

// setTermSignals sets the application termination signals processing.
// Also it waits the exit status from main()
func (ac *AppContext) setTermSignals() {

	// Termination signals processing
	sigChan := make(chan os.Signal, 1)

	go func() {
		signal.Notify(sigChan, ac.termSignals...)
		for {
			select {
			case s := <-sigChan:
				ac.log.WithFields(logrus.Fields{
					"signal": s,
				}).Debug("got terminating signal")
				ac.terminate(ExitStatusSuccess)
			case s := <-ac.termChanFromMain:
				ac.terminate(s)
			}
		}
	}()
}

// setReloadSignals sets the application reload signals processing
func (ac *AppContext) setReloadSignals() {

	// Context reload signals processing
	sigChan := make(chan os.Signal, 1)

	go func() {
		signal.Notify(sigChan, ac.reloadSignals...)
		for s := range sigChan {

			ac.log.WithFields(logrus.Fields{
				"signal": s,
			}).Debug("got reloading signal")

			d, err := ac.cc.Reload(CustomContextFuncOpts{
				Args:   ac.args,
				Config: ac.cfgPath,
				Log:    ac.log,
			})
			if err != nil {
				ac.terminate(ExitStatusFailure)
				continue
			}

			// Save old values
			o := ac.cfgData

			// Set new values
			ac.cfgData = d

			if ac.cfgData.LogFile != o.LogFile {

				// If logfile path has been changed

				if err = LogfileChange(ac.log, ac.cfgData.LogFile, ac.cfgData.LogLevel, ac.logrotateSignals); err != nil {
					ac.log.Errorf("context reload error: %v", err)
					ac.terminate(ExitStatusFailure)
					continue
				}
			} else {
				if ac.cfgData.LogLevel != o.LogLevel {

					// If log level has been changed only

					// Validate loglevel
					level, err := logrus.ParseLevel(ac.cfgData.LogLevel)
					if err != nil {
						ac.log.Errorf("context reload error: wrong loglevel value: %s", ac.cfgData.LogLevel)
						ac.terminate(ExitStatusFailure)
						continue
					}
					ac.log.SetLevel(level)
				}
			}

			if err := PidfileChange(o.PidFile, ac.cfgData.PidFile); err != nil {
				ac.log.Errorf("context reload error: %v", err)
				ac.terminate(ExitStatusFailure)
				continue
			}

			// Send updated context into specified channels
			for _, r := range ac.routines {
				r.crc <- nil
			}
		}
	}()
}

// Application termination
func (ac *AppContext) terminate(status int) {

	// Call all context cancel functions
	for _, r := range ac.routines {
		r.cf()
	}

	// Wait for application runtime routines done
	ac.wg.Wait()

	es := ac.cc.Free(CustomContextFuncOpts{
		Args:   ac.args,
		Config: ac.cfgPath,
		Log:    ac.log,
	})

	// If desired status is set
	if status != 0 {
		es = status
	}

	// Remove pid file if necessary
	PidfileRemove(ac.cfgData.PidFile)

	// Notify main function with exit code
	ac.termChanToMain <- es
}
