package pmtilr

import "context"

type Role int

const (
	Leader Role = iota
	Waiter
)

type Response struct {
	Body []byte
	Err  error
}

type Ticket struct {
	done <-chan struct{}
	res  *Response
}

func (t Ticket) Done() <-chan struct{} { return t.done }
func (t Ticket) Result() Response      { return *t.res }

type acquireReq struct {
	key string
	ch  chan acquireResp
}

type acquireResp struct {
	role   Role
	ticket Ticket
}

type completeReq struct {
	key string
	res Response
}

type entry struct {
	done    chan struct{}
	res     Response
	waiters int
}

type InflightActor struct {
	acquireCh  chan acquireReq
	completeCh chan completeReq
	maxWaiters int
	m          map[string]*entry
}

func NewInflightActor(maxWaiters int, acquireBuffer int) *InflightActor {
	a := &InflightActor{
		acquireCh:  make(chan acquireReq, acquireBuffer),
		completeCh: make(chan completeReq, 64),
		maxWaiters: maxWaiters,
		m:          make(map[string]*entry),
	}
	go a.run()
	return a
}

func (a *InflightActor) run() {
	for {
		// Priority: drain all completions first, they unblock waiters.
		select {
		case m := <-a.completeCh:
			a.onComplete(m)
			continue
		default:
		}

		select {
		case m := <-a.completeCh:
			a.onComplete(m)
		case m := <-a.acquireCh:
			a.onAcquire(m)
		}
	}
}

func (a *InflightActor) Acquire(ctx context.Context, key string) (Role, Ticket, error) {
	reply := make(chan acquireResp, 1)
	req := acquireReq{key: key, ch: reply}

	select {
	case a.acquireCh <- req:
	case <-ctx.Done():
		return 0, Ticket{}, ctx.Err()
	}

	select {
	case resp := <-reply:
		return resp.role, resp.ticket, nil
	case <-ctx.Done():
		return 0, Ticket{}, ctx.Err()
	}
}

func (a *InflightActor) Complete(ctx context.Context, key string, res Response) error {
	req := completeReq{key: key, res: res}
	select {
	case a.completeCh <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *InflightActor) onAcquire(req acquireReq) {
	if e, ok := a.m[req.key]; ok {
		e.waiters++
		req.ch <- acquireResp{
			role:   Waiter,
			ticket: Ticket{done: e.done, res: &e.res},
		}
		return
	}

	e := &entry{done: make(chan struct{})}
	a.m[req.key] = e
	req.ch <- acquireResp{
		role:   Leader,
		ticket: Ticket{done: e.done, res: &e.res},
	}
}

func (a *InflightActor) onComplete(req completeReq) {
	e, ok := a.m[req.key]
	if !ok {
		return
	}
	e.res = req.res
	close(e.done)
	delete(a.m, req.key)
}
