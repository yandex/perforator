package httpmock

import (
	"io"
	"net/http"
	"reflect"
	"sync/atomic"

	"github.com/jarcoal/httpmock/internal"
)

var (
	GetPackage     = getPackage
	ExtractPackage = extractPackage
	CalledFrom     = calledFrom
)

type (
	MatchResponder  = matchResponder
	MatchResponders = matchResponders
)

func init() {
	atomic.AddInt64(&matcherID, 0xabcdef)
}

func GetIgnorePackages() map[string]bool {
	return ignorePackages
}

// bodyCopyOnRead

func NewBodyCopyOnRead(body io.ReadCloser) *bodyCopyOnRead { //nolint: revive
	return &bodyCopyOnRead{body: body}
}

func (b *bodyCopyOnRead) Body() io.ReadCloser {
	return b.body
}

func (b *bodyCopyOnRead) Rearm() {
	b.rearm()
}

// matchRouteKey

func NewMatchRouteKey(rk internal.RouteKey, name string) matchRouteKey { //nolint: revive
	return matchRouteKey{RouteKey: rk, name: name}
}

// matchResponder

func NewMatchResponder(matcher Matcher, resp Responder) matchResponder { //nolint: revive
	return matchResponder{matcher: matcher, responder: resp}
}

func (mr matchResponder) ResponderPointer() uintptr {
	return reflect.ValueOf(mr.responder).Pointer()
}

func (mr matchResponder) Matcher() Matcher {
	return mr.matcher
}

// matchResponders

func (mrs matchResponders) Add(mr matchResponder) matchResponders {
	return mrs.add(mr)
}

func (mrs matchResponders) Remove(name string) matchResponders {
	return mrs.remove(name)
}

func (mrs matchResponders) FindMatchResponder(req *http.Request) *matchResponder {
	return mrs.findMatchResponder(req)
}

// Matcher

func (m Matcher) FnPointer() uintptr {
	return reflect.ValueOf(m.fn).Pointer()
}
