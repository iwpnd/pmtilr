package pmtilr

import (
	"context"
)

type Role int

const (
	Leader = iota
	Waiter
)

// Response is the concatenated response that is fetched
// once from the Leader.
type Response struct {
	Body []byte
	Err  error
}

type Ticket struct {
	done <-chan struct{}
	res  *Response
}

func (t Ticket) Done() <-chan struct{} {
	return t.done
}

func (t Ticket) Result() Response {
	return *t.res
}

func spanKey(offset, length uint64) string {
	return buildCacheKey("span", offset, length)
}

type acquireReq struct {
	key string
	ch  chan acquireResp
}

type acquireResp struct {
	role   Role
	ticket Ticket
	err    error
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
	inbox      chan any
	maxWaiters int
	m          map[string]*entry
}

func NewInflightActor(maxWaiters, inboxBuffer int) *InflightActor {
	a := &InflightActor{
		inbox:      make(chan any, inboxBuffer),
		maxWaiters: maxWaiters,
		m:          make(map[string]*entry),
	}

	go a.run()

	return a
}

func (a *InflightActor) run() {
	for msg := range a.inbox {
		switch m := msg.(type) {
		case acquireReq:
			a.onAcquire(m)
		case completeReq:
			a.onComplete(m)
		}
	}
}

func (a *InflightActor) Acquire(ctx context.Context, key string) (Role, Ticket, error) {
	reply := make(chan acquireResp, 1)
	req := acquireReq{key: key, ch: reply}

	select {
	case a.inbox <- req:
	case <-ctx.Done():
		return 0, Ticket{}, ctx.Err()
	}

	select {
	case resp := <-reply:
		return resp.role, resp.ticket, resp.err
	case <-ctx.Done():
		return 0, Ticket{}, ctx.Err()
	}
}

func (a *InflightActor) Complete(ctx context.Context, key string, res Response) error {
	req := completeReq{key: key, res: res}
	select {
	case a.inbox <- req:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (a *InflightActor) onAcquire(req acquireReq) {
	// waiter path
	if entry, ok := a.m[req.key]; ok {
		entry.waiters++
		req.ch <- acquireResp{
			role: Waiter,
			ticket: Ticket{
				done: entry.done,
				res:  &entry.res,
			},
		}
		return
	}

	// first req, leader path
	e := &entry{done: make(chan struct{})}
	a.m[req.key] = e
	req.ch <- acquireResp{
		role: Leader,
		ticket: Ticket{
			done: e.done,
			res:  &e.res,
		},
	}
}

func (a *InflightActor) onComplete(req completeReq) {
	e, ok := a.m[req.key]
	if !ok {
		// already completed/deleted or not a request
		return
	}
	e.res = req.res
	close(e.done)
	delete(a.m, req.key)
}
