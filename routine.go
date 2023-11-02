package appctx

import (
	"context"
	"sync"

	"github.com/cskr/pubsub"
)

type RoutineParam struct {
	Handler RoutineHandler
}

type RoutineHandler func(App) error

const (
	RoutineStateUnknown routineState = "unknown"
	RoutineStateStandby routineState = "standby"
	RoutineStateRun     routineState = "run"
	RoutineStateDone    routineState = "success"
	RoutineStateFailed  routineState = "failed"
)

type routineState string

type routine struct {
	name    string
	handler RoutineHandler
	ctx     context.Context
	cf      context.CancelFunc
	state   routineState
	wg      *sync.WaitGroup
	ps      *pubsub.PubSub
	va      chan any
	mutex   sync.Mutex
}

type routineSettings struct {
	name    string
	wg      *sync.WaitGroup
	ps      *pubsub.PubSub
	handler RoutineHandler
}

func (rs routineState) String() string {
	return string(rs)
}

func routineInit(s routineSettings) *routine {
	return &routine{
		name:    s.name,
		handler: s.handler,
		wg:      s.wg,
		ps:      s.ps,
		state:   RoutineStateStandby,
	}
}

func (r *routine) run(ctx context.Context, ax *AppCtx) {

	if r.handler == nil {
		return
	}

	r.wg.Add(1)

	// Routine runner
	go func() {

		defer r.wg.Done()

		// Prevent double-run routine
		if r.stateGet() != RoutineStateRun {

			r.mutex.Lock()

			r.ctx, r.cf = context.WithCancel(ctx)

			f := r.valueWatcher()
			defer f()

			// Set routine state to `run`
			r.state = RoutineStateRun
			r.mutex.Unlock()

			err := r.handler(App{
				routine: r,
				ax:      ax,
			})

			r.mutex.Lock()

			// Set routine state in appropriate with returned error
			if err == nil {
				r.state = RoutineStateDone
			} else {
				r.state = RoutineStateFailed
			}

			r.mutex.Unlock()
		}
	}()
}

func (r *routine) shutdown() {

	if r.stateGet() != RoutineStateRun {
		return
	}

	// Check that context has not been already canceled
	if r.ctx.Err() == nil {
		r.cf()
	}
}

func (r *routine) valueWatcher() context.CancelFunc {

	r.va = make(chan any, 1)

	c, f := context.WithCancel(r.ctx)

	r.wg.Add(1)

	// Value change channel receiver
	go func() {

		vs := r.ps.Sub(psValue)

		defer r.wg.Done()
		defer func() {
			// Unsubscribe from value update channel when
			// primary routine finished
			go r.ps.Unsub(vs, psValue)
			for range vs {
			}

			close(r.va)
			r.va = nil
		}()

		for {
			select {
			case <-vs:
				if len(r.va) == 0 {
					r.va <- true
				}
			case <-c.Done():
				return
			}
		}
	}()

	return f
}

func (r *routine) nameGet() string {
	return r.name
}

func (r *routine) valueC() <-chan any {
	return r.va
}

func (r *routine) valueCheck() bool {

	if len(r.va) == 0 {
		return false
	}

	// Skip all data from chan
	for range r.va {
	}

	return true
}

func (r *routine) ctxGet() context.Context {
	return r.ctx
}

func (r *routine) stateGet() routineState {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return r.state
}
