package appctx

import (
	"context"
	"os"
	"os/signal"
	"sync"
)

type Signal struct {
	s   os.Signal
	ctx context.Context
	ax  *AppCtx
}

type SignalsParam struct {
	Signals []os.Signal
	Handler SigHandler
}

type SigHandler func(Signal)

type sig struct {
	signals []os.Signal
	handler SigHandler
}

func (s *sig) run(ctx context.Context, wg *sync.WaitGroup, ax *AppCtx) {

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, s.signals...)

	c, f := context.WithCancel(ctx)
	defer f()

	for {
		select {
		case v := <-sc:

			if s.handler == nil {
				break
			}

			wg.Add(1)
			s.handler(Signal{
				s:   v,
				ctx: c,
				ax:  ax,
			})
			wg.Done()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Signal) SignalGet() os.Signal {
	return s.s
}

func (s *Signal) Ctx() context.Context {
	return s.ctx
}

func (s *Signal) CtxDone() <-chan struct{} {
	return s.ctx.Done()
}

func (s *Signal) RoutineState(name string) routineState {
	return s.ax.routineState(name)
}

func (s *Signal) RoutineStart(name string) error {
	return s.ax.routineStart(name)
}

func (s *Signal) RoutineShutdown(name string) {
	s.ax.routineShutdown(name)
}

func (s *Signal) ValueGet() any {
	return s.ax.valueGet()
}

func (s *Signal) ValueSet(v any) {
	s.ax.valueSet(v)
}

func (s *Signal) Shutdown(e error) {
	s.ax.shutdown(e)
}
