package roundrobin

import (
	"fmt"
	"github.com/apex/log"
	"sync"
	"sync/atomic"
)

type Token struct {
	token string
	valid bool
	lock  sync.RWMutex
}

type RoundRobiner interface {
	Pick() (*Token, error)
}

// The New() function is a constructor used to create a new RoundRobin instance.
func New(tokens []string) RoundRobiner {
	log.Debugf("create round robin with %d tokens", len(tokens))
	if len(tokens) == 0 {
		return &noTokenRoundRobin{}
	}
	result := make([]*Token, 0, len(tokens))
	for _, item := range tokens {
		result = append(result, NewToken(item))
	}
	return &realRoundRobin{tokens: result}
}

type noTokenRoundRobin struct {
}

func (rr *noTokenRoundRobin) Pick() (*Token, error) {
	return nil, nil
}

func NewToken(token string) *Token {
	return &Token{
		token: token,
		valid: true,
	}
}

type realRoundRobin struct {
	tokens []*Token
	next   int64
}

func (rr *realRoundRobin) Pick() (*Token, error) {
	return rr.doPick(0)
}

func (rr *realRoundRobin) doPick(try int) (*Token, error) {
	if try > len(rr.tokens) {
		return nil, fmt.Errorf("no valid token left")
	}
	idx := atomic.LoadInt64(&rr.next)
	atomic.StoreInt64(&rr.next, (idx+1)%int64(len(rr.tokens)))
	if pick := rr.tokens[idx]; pick.OK() {
		log.Debugf("picked %s", pick.Key())
		return pick, nil
	}
	return rr.doPick(try + 1)
}

func (t *Token) OK() bool {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.valid
}

func (t *Token) Invalidate() {
	log.Warnf("invalidated token '...%s'", t)
	t.lock.Lock()
	defer t.lock.Unlock()
	t.valid = false
}

func (t *Token) Key() string {
	return t.token
}
