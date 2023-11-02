package appctx

import "context"

type App struct {
	routine *routine

	ax *AppCtx
}

const psValue = "value"

func (a *App) SelfNameGet() string {
	return a.routine.nameGet()
}

func (a *App) SelfCtx() context.Context {
	return a.routine.ctxGet()
}

func (a *App) SelfCtxDone() <-chan struct{} {
	return a.routine.ctxGet().Done()
}

func (a *App) RoutineState(name string) routineState {
	return a.ax.routineState(name)
}

func (a *App) RoutineStart(name string) error {
	return a.ax.routineStart(name)
}

func (a *App) RoutineShutdown(name string) {
	a.ax.routineShutdown(name)
}

func (a *App) ValueGet() any {
	return a.ax.valueGet()
}

func (a *App) ValueSet(v any) {
	a.ax.valueSet(v)
}

func (a *App) ValueC() <-chan any {
	return a.routine.valueC()
}

func (a *App) Shutdown(e error) {
	a.ax.shutdown(e)
}
