# nxs-go-appctx

This Go package provides tools to make Go applications context. You can write application code instead of system kits to allow your daemons work fine. 

## Features

- **Application context control**  
Most applications consist of goroutines that implements program functionality and context data that contains data used at runtime; i.e. database credentials, API settings, etc. `Nxs-go-appctx` allows control of the derived goroutines and keep context data up-to-date via following context actions:
  - *Init*: sets context settings, loads data from config file, creates log and pid files, if necessary
  - *Reload*: reloads config file and updates application context data
  - *Terminate*: frees context data and terminates the aplliaction

- **Logging**  
`Nxs-go-appctx` uses the [logrus](https://github.com/sirupsen/logrus) logger created at the *init* stage and is available in application at runtime. In accordance with context settings, log file can be changed after context reload.

- **Pid files**  
If the *pid file* path is set, the pid file with the program PID will be automatically created at the *init* and removed at the *terminate* stage. In accordance with context settings, pid file can be changed after context reload.

- **Reload signals**  
If the *reload signals* is set, every time they are sent to the application, its context data will be updated in accordance with `ContextReload` function.

- **Termination signals**  
If the *terminate signals* is set, every time they are sent to the application, its context data will be freed in accordance with `ContextFree` function and the application itself will be terminated.

- **Logrotate signals**  
If the *logrotate signals* is set, every time they are sent to the application the log file will be reopened. It is useful for `logrotate` utility.

## Getting started

For better understanding the `nxs-go-appctx` description will be provided with the sample gists. You can find the complete code example in the `example/` directory. Also see the [Example of usage](#example-of-usage) section for more information.

### Setup nxs-go-appctx

At first, you need to define the struct that contains an application context data:

```go
type selfContext struct {
    timeInterval int
}
```

Next, declare the variables:
- Context variable `ctx` as type `selfContext` in main() function
  ```go
  var ctx selfContext
  ```
- Logger variable `log` as type `*logrus.Logger`. For convenience it could be global variable:
  ```go
  var log *logrus.Logger
  ```

Create three functions to manage application context:

- **Context init function** 
This function must read the config file (e.g. with [nxs-go-conf](https://github.com/nixys/nxs-go-conf)), set the application context data and return the `appctx.CfgData` struct:
  ```go
  func contextInit(ctx interface{}, cfgPath string) (appctx.CfgData, error) {
  
  	var cfgData appctx.CfgData
  
  	// Read config file
  	conf, err := confRead(cfgPath)
  	if err != nil {
  		return cfgData, err
  	}
  
  	// Set application context
  	c := ctx.(*selfContext)
  	c.timeInterval = conf.TimeInterval
  
  	// Fill `appctx.CfgData`
  	cfgData.LogFile = conf.LogFile
  	cfgData.LogLevel = conf.LogLevel
  	cfgData.PidFile = conf.PidFile
  
  	return cfgData, nil
  }
  ```

- **Context reload function**  
This function can read config file and change the application context with new data. It also must fill and return the `appctx.CfgData` struct. Usually this function almost the same as `contextInit`:
  ```go
  func contextReload(ctx interface{}, cfgPath string, singnal os.Signal) (appctx.CfgData, error) {
  
  	log.WithFields(logrus.Fields{
  		"signal": singnal,
  	}).Debug("program reload")
  
  	return contextInit(ctx, cfgPath)
  }
  ```

- **Context free function**
This function must clean up the application context if necessary (i.e. database disconnect, etc.) and return the program exit status. In the simle case it can looks like this:
  ```go
  func contextFree(ctx interface{}, signal os.Signal) int {
  
  	log.WithFields(logrus.Fields{
  		"signal": signal,
  	}).Debug("got termination signal")
  
  	return 0
  }
  ```

Then declare and initialize the `appCtx` variable with `appctx` settings in main() function:
```go
appCtx := appctx.AppContext{
    AppCtx:           &ctx,
    CfgPath:          configPath,
    CtxInit:          contextInit,
    CtxReload:        contextReload,
    CtxFree:          contextFree,
    TermSignals:      []os.Signal{syscall.SIGTERM, syscall.SIGINT},
    ReloadSignals:    []os.Signal{syscall.SIGHUP},
    LogrotateSignals: []os.Signal{syscall.SIGUSR1},
}
```

The fields description is as follows:
- `AppCtx`: context application data pointer
- `CfgPath`: config file path
- `CtxInit`: context init function
- `CtxReload`: context reload function
- `CtxFree`: context free function
- `TermSignals`: termination signals array
- `ReloadSignals`: reload signals array
- `LogrotateSignals`: logrotate signals array

After the `appCtx` variable declaration call the `appCtx.ContextInit()` function. It will set the application context, create log and pid files and return the logger:
```go
log, err = appCtx.ContextInit()
if err != nil {
    fmt.Println(err)
    os.Exit(1)
}
```

Next you need to setup the *context* and *cancel* functions (due to [context](https://golang.org/pkg/context/) package) for each goroutines to be executed at runtime:
```go
// Create main context
c := context.Background()

// Create derived context for goroutine
cRuntime, cf := context.WithCancel(c)
```

Add a new goroutine element into `appctx`. This action creates a `context reload channel` for goroutine. This channel is used to send new application context data to goroutine when program is reloaded:
```go
// Add a goroutine element into appctx
crc := appCtx.RoutineAdd(cf)
```

### Goroutines

After goroutine element is added into `appctx` you can start the new goroutine. Note that all goroutines created using `RoutineAdd()` must call `RoutineDone()` when completed:
```go
defer appCtx.RoutineDone(crc)
runtime(cRuntime, ctx, crc)
```

Within the goroutines you should process at least the following channels:
- `context done channel`, to receive notification when program terminates
- `context reload channel`, to receive new application context data

Optionally, you can process other channels in accordance with your application algorithm.

Goroutine `runtime` function may look like this:
```go
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
			// Terminate the program.
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
```

### Program termination

To correctly terminate the program, the main() function should perform the following steps:

- Wait for program termination (e.g. by receiving termination signals):
  ```go
  // Wait for program termination
  ec := appCtx.ExitWait()
  ```

- Done the appctx. It will complete write operations at the log file.
  ```go
  // Done the appctx
  appCtx.ContextDone()
  ```

- Exit from program using `os.Exit()` call with specified status:
  ```go
  // Exit from program with `ec` status
  os.Exit(ec)
  ```

## Install

```
go get github.com/nixys/nxs-go-appctx
```

## Example of usage

The `example` program:

- Use the following config file:
  ```yaml
  time_int: 3
  logfile: /tmp/example.log
  loglevel: debug
  pidfile: /tmp/example.pid
  ```

- Handle the following termination signals:
  - SIGTERM
  - SIGINT

- Handle the following reload signals:
  - SIGHUP

- Handle the following logrotate signals:
  - SIGUSR1

After starting the program you will see in the log file similar like this:
```
[2019-01-22T08:18:53+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:18:54+07:00] INFO: Time to work! [2] (time interval: 4)
[2019-01-22T08:18:56+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:18:58+07:00] INFO: Time to work! [2] (time interval: 4)
[2019-01-22T08:18:59+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:19:02+07:00] INFO: Time to work! [2] (time interval: 4)
[2019-01-22T08:19:02+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:19:05+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:19:06+07:00] INFO: Time to work! [2] (time interval: 4)
[2019-01-22T08:19:08+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:19:10+07:00] INFO: Time to work! [2] (time interval: 4)
[2019-01-22T08:19:11+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:19:14+07:00] INFO: Time to work! [2] (time interval: 4)
[2019-01-22T08:19:14+07:00] INFO: Time to work! (time interval: 3)
[2019-01-22T08:19:15+07:00] INFO: Done [2]
[2019-01-22T08:19:15+07:00] INFO: Done
[2019-01-22T08:19:15+07:00] DEBUG: got termination signal (signal: terminated)
[2019-01-22T08:19:15+07:00] INFO: program terminating (exit code: 0)
```

Then you can play with the config file. After the config file change send the `SIGHUP` to the application with the following command:
```
kill -HUP `cat /tmp/example.pid`
```
and check the log file.

To terminate the program use:
```
kill -TERM `cat /tmp/example.pid`
```
