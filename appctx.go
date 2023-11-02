package appctx

import (
	"context"
	"fmt"
	"sync"

	"github.com/cskr/pubsub"
)

type AppCtx struct {
	ctx              context.Context
	cf               context.CancelCauseFunc
	value            any
	signals          []sig
	routines         map[string]*routine
	wg               *sync.WaitGroup
	valueInitHandler ValueInitHandler
	ps               *pubsub.PubSub
	mutex            sync.Mutex
}

type ValueInitHandler func() (any, error)

func Init(ctx context.Context) *AppCtx {

	if ctx == nil {
		ctx = context.Background()
	}

	c, f := context.WithCancelCause(ctx)

	return &AppCtx{
		ctx:      c,
		cf:       f,
		routines: make(map[string]*routine),
		ps:       pubsub.New(0),
		wg:       &sync.WaitGroup{},
	}
}

func (ax *AppCtx) RoutinesSet(routines map[string]RoutineParam) *AppCtx {

	// TODO: destroy previous routines
	// make new map

	for n := range routines {
		ax.routines[n] = routineInit(routineSettings{
			name:    n,
			wg:      ax.wg,
			ps:      ax.ps,
			handler: routines[n].Handler,
		})
	}

	return ax
}

func (ax *AppCtx) Run() error {

	var err error

	// Read args, conf files and make user values
	if ax.valueInitHandler != nil {
		ax.value, err = ax.valueInitHandler()
		if err != nil {
			return fmt.Errorf("app ctx: conf handler: %w", err)
		}
	}

	for i := range ax.signals {

		c, f := context.WithCancel(ax.ctx)
		defer f()

		s := ax.signals[i]
		go s.run(c, ax.wg, ax)
	}

	for n := range ax.routines {
		r := ax.routines[n]
		r.run(ax.ctx, ax)
	}

	ax.wg.Wait()

	return context.Cause(ax.ctx)
}

func (ax *AppCtx) SignalsSet(sigs []SignalsParam) *AppCtx {

	ax.signals = []sig{}

	for _, s := range sigs {
		ax.signals = append(
			ax.signals,
			sig{
				signals: s.Signals,
				handler: s.Handler,
			},
		)
	}

	return ax
}

func (appctx *AppCtx) ValueInitHandlerSet(vh ValueInitHandler) *AppCtx {

	appctx.valueInitHandler = nil

	if vh != nil {
		appctx.valueInitHandler = vh
	}

	return appctx
}

func (appctx *AppCtx) routineState(name string) routineState {
	r, b := appctx.routines[name]
	if b == false {
		return RoutineStateUnknown
	}
	return r.stateGet()
}

func (ax *AppCtx) routineStart(name string) error {

	r, b := ax.routines[name]
	if b == false {
		return fmt.Errorf("app ctx: routine start: %w", ErrNotFound)
	}

	r.run(ax.ctx, ax)

	return nil
}

func (ax *AppCtx) routineShutdown(name string) {

	r, b := ax.routines[name]
	if b == false {
		return
	}

	r.shutdown()
}

func (ax *AppCtx) valueGet() any {
	ax.mutex.Lock()
	defer ax.mutex.Unlock()
	return ax.value
}

func (ax *AppCtx) valueSet(v any) {
	ax.mutex.Lock()
	defer ax.mutex.Unlock()
	ax.value = v
	ax.ps.Pub(true, psValue)
}

func (ax *AppCtx) shutdown(e error) {
	ax.cf(e)
}
