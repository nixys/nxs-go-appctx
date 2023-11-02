# nxs-go-appctx

## Introduction

The library helps you create a microservices, Kubernetes operators and CLI utilities in Go.

### Features

- Used approaches make you able to create an application with a flexibility you need to solve your business or engineer issues
- Great for building REST API servers and Kubernetes operators
- Multi-routines application arch. You are able to design an app with logic divided between independent routines
- Unified application values (with any data you need) and methods to manage it (including notification to every app routine if application context has been changed)
- OS signal (Linux) handlers. You can define an OS signals you need to handle. E.g. with this you may create a graceful shutdown for your application
- Suggested application structure answers most questions about location of any element you need to add into your program

### Who can use the tool

Developers and development teams who need to create a lot of applications, microservices and CLI utilities with unified structure to reduce the cost of developing and supporting.

## Quickstart

### Import

```go
import "github.com/nixys/nxs-go-appctx/v3"
```

### Initialize

You need to do following initialization steps to be ready to use `nxs-go-appctx`:

| Method | Required | Description |
| --- | :---: | --- |
| `Init(context.Context)` | Yes | Initializes `nxs-go-appctx` context. You may specify a parent context for `nxs-go-appctx` or set `nil` to use `context.Background()` |
| `ValueInitHandlerSet(appctx.ValueInitHandler)` | No | Sets a handler to initialize application `value` See [Value](#value) for details |
| `RoutinesSet(map[string]appctx.RoutineParam)` | No | Sets a routines (workers) for your app. See [Routine](#routine) for details |
| `SignalsSet([]appctx.SignalsParam)` | No | Sets an OS signals (at the moment only Linux is supported) and handlers for each of them. See [Signal handler](#signal-handler) for details |
| `Run()` | Yes | Runs the `nxs-go-appctx` context that you have been initialized. See [Runtime](#runtime) for details |

#### Logging

`nxs-go-appctx` has no logging within the library so you are able to use any logger you want.

For your convenience `nxs-go-appctx` contains a function `appctx.DefaultLogInit()` to help you initialize logger ([Logrus](github.com/sirupsen/logrus) is used), so you may use it in your application.

### Runtime

After `appctx.Run()` method has been called the following actions will be executed:
- Call `appctx.ValueInitHandler` if it was set at initialize stage
- Launch specified signal handlers
- Launch all specified routines

`nxs-go-appctx` will finished in one of following cases:
- All routines and signal handlers are finished
- If `appctx.App.Shutdown()` or `appctx.Signal.Shutdown()` methods is called (see below for details)

#### Value

Value in `nxs-go-appctx` contains any data you need within your application (e.g. config file options, database connections, etc) and available to operate with in each routine and signal handler. You are able to get this `value` and set a new one. After the `value` is changed a notification will be sent to every routine in your application (it's not require to catch this notification and may be ignored).

#### Routine

Routines it's a named worker functions in `nxs-go-appctx` implements certain piece of isolated functionality within your app (e.g. API server, periodically updates of cache, health checker, etc).

In each point of time routine stay in one of following states (also you are able to manage a state of one routine from another):

| State | Description |
| :---: | --- |
| Standby | Routine has this state after set via `RoutinesSet()` and till its first launch |
| Run | On execution the routine |
| Success | If routine has been finished without error |
| Failed | If routine has been finished with error |

In the routine at runtime you are able to operate with `nxs-go-appctx` via `appctx.App` methods:

| Method | Description |
| --- | --- |
| `SelfNameGet()` | Get name of the current routine |
| `SelfCtx()` | Get `context.Context` for current routine. Useful to create a derived context |
| `SelfCtxDone()` | Get Done channel for context of current routine. The routine must be able to handle this signal in order to exit correctly when it required |
| `RoutineState(string)` | Get state of specified (by name) routine |
| `RoutineStart(string)` | Start specified (by name) routine. Applies only to not running routines (i.e. has status `Success` or `Failed`) |
| `RoutineShutdown(string)` | Shutdown (via context cancel function call) specified (by name) routine. Applies only to running routines (i.e. has status `Run`). Note that `nxs-go-appctx` will be finished with the last finished routine |
| `ValueGet()` | Get a `nxs-go-appctx` application `value` |
| `ValueSet(any)` | Set a new `nxs-go-appctx` application `value`. After new `value` has been specifed a notification will be sent to each routine in `Run` state |
| `ValueC()` | Get a channel to obtain a notifications of updated `value` |
| `Shutdown(error)` | Shutdown `nxs-go-appctx`. After this method is called `nxs-go-appctx` will be shutdown via Go context. When all routines and signal handlers will finished `appctx.Run()` function return an error you specified in `Shutdown(error)` call |

#### Signal handler

Signal handlers is used for processing OS signals sent to your application. Useful for graceful shutdown your application or reload a config files. 

You may specify a handler for every set of signals you need to processing. A signal handlers only runs when your application receives an appropriate OS signal and need to be finished after processing.

At runtime signal handler able to operate with `nxs-go-appctx` via `appctx.Signal` methods:

| Method | Description |
| --- | --- |
| `SignalGet()` | Get the signal that triggered current handler |
| `Ctx()` | Get `context.Context` for current handler. Useful to create a derived context |
| `CtxDone()` | Get Done channel for context of current handler |
| `RoutineState(string)` | Get state of specified (by name) routine |
| `RoutineStart(string)` | Start specified (by name) routine. Applies only to not running routines (i.e. has status `Success` or `Failed`) |
| `RoutineShutdown(string)` | Shutdown (via context cancel function call) specified (by name) routine. Applies only to running routines (i.e. has status `Run`). Note that `nxs-go-appctx` will be finished with the last finished routine |
| `ValueGet()` | Get a `nxs-go-appctx` application `value` |
| `ValueSet(any)` | Set a new `nxs-go-appctx` application `value`. After new `value` has been specifed a notification will be sent to each routine in `Run` state |
| `Shutdown(error)` | Shutdown `nxs-go-appctx`. After this method is called `nxs-go-appctx` will be shutdown via Go context. When all routines and signal handlers will finished `appctx.Run()` function return an error you specified in `Shutdown(error)` call |

## App structure

You may use with `nxs-go-appctx` any application structure you want. But we invite you consider to using the structure based on our experience of using this library.

Graphic diagram of the main elements interaction:

![nxs-go-appctx structure](docs/images/nxs-go-appctx.png)

General directory structure:
- [main.go](#main)
- [ctx/](#ctx)
- [routines/](#routines)
- [modules/](#modules)
- [ds/](#datasources)
- [misc/](#misc)
- [api/](#api)

### Main

The `main.go` contains only a `main()` with initialize and run `nxs-go-appctx` and also an error handler to return appropriate exit code.

### Ctx

This element do two main things:
- Describes `values` used through the application and available in every `routine` and `signal handler`
- Creates `values` with context from CLI args, config files, connections to a databases you need to use in your app, etc 

The `ctx/` directory may contains following files:
- `ctx/args.go`: defines how to read and processing command line arguments
- `ctx/conf.go`: defines how to read and processing config file. It's useful to use [nxs-go-conf](https://github.com/nixys/nxs-go-conf) package to work with config files
- `ctx/context.go`: contains `appctx.ValueInitHandler()` used to create a `values` and calles all need methods from `ctx/args.go` and `ctx/conf.go`

### Routines

Each `nxs-go-appctx` `routine` locates in separate subdirectory within the `routines/` directory and consist of `Runtime()` called from `nxs-go-appctx` and may contain other helper functions if it's necessary.

*Routines* must interact only with the following application elements:
- Ctx
- Modules
- Misc
- API

### Modules

Modules are application units perform a certain logical isolated tasks. Each module has with any structure you need for your app and locates in separate subdirectory in `modules/` directory.

For example, if your app has a `user` table in database with raw dataset such as `id`, `name`, `password` and you need to do complex perform of this data (e.g. find all user whose names match specifed regex), you need to create a *module* `modules/user/` with appropriate code.

*Module* must interact only with the following application elements:
- Other modules
- Datasources
- Misc

### Datasources

Datasources locates in a some subdirectories in `ds/` directory and serve to interaction with any external datasources you need to use such as DBs (eg. Redis, MongoDB, PostgreSQL, MySQL, etc), external APIs (e.g. GitHub API, Kubernetes API, etc), etc.

*Datasources* must interact only with the following application elements:
- Misc

### Misc

Misc contains helper functions, global errors, structures, and other objects you need to use in any part of the application.

*Misc* must interact only with following application elements:
- Misc

### API

This part of application describes an API and contains all elements required for processing requests to server (such as endpoints, methods and handlers).

*API* must interact only with the following application elements:
- Modules
- Misc

## Example

You can see more practical information about `nxs-go-appctx` in following examples:

- [nxs-go-appctx-example-restapi](https://github.com/nixys/nxs-go-appctx-example-restapi)

## Roadmap

- Unified health checker
- Default OS signal handlers

## Feedback

For support and feedback please contact me:
- [GitHub issues](https://github.com/nixys/nxs-go-appctx/issues)
- telegram: [@borisershov](https://t.me/borisershov)
- e-mail: b.ershov@nixys.io

## License

nxs-go-appctx is released under the [MIT License](LICENSE).
