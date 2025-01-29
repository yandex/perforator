package graceful

import "context"

////////////////////////////////////////////////////////////////////////////////

type shutdownState struct {
	requested chan bool
	finished  chan bool
}

type ShutdownSource struct {
	*shutdownState
}

type ShutdownCookie struct {
	*shutdownState
}

////////////////////////////////////////////////////////////////////////////////

func NewShutdownCookie() ShutdownCookie {
	state := &shutdownState{
		requested: make(chan bool),
		finished:  make(chan bool),
	}

	return ShutdownCookie{state}
}

////////////////////////////////////////////////////////////////////////////////

func (c *ShutdownCookie) Stop(ctx context.Context) error {
	c.Signal()
	return c.Wait(ctx)
}

func (c *ShutdownCookie) Signal() {
	close(c.requested)
}

func (c *ShutdownCookie) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.finished:
		return nil
	}
}

func (c *ShutdownCookie) GetSource() ShutdownSource {
	return ShutdownSource{c.shutdownState}
}

////////////////////////////////////////////////////////////////////////////////

func (c *ShutdownSource) Done() chan bool {
	return c.requested
}

func (c *ShutdownSource) IsDone() bool {
	select {
	case <-c.requested:
		return true
	default:
		return false
	}
}

func (c *ShutdownSource) Finish() {
	close(c.finished)
}

////////////////////////////////////////////////////////////////////////////////
