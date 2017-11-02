package dxl

import (
	"errors"
	"fmt"
	"sync"
)

type rpcEntry struct {
	ts      int
	reschan chan<- *ServiceResponse
}

type RpcManager struct {
	sync.Mutex
	mailbox map[string]*rpcEntry
}

func newRpcManager() *RpcManager {
	return &RpcManager{
		mailbox: make(map[string]*rpcEntry),
	}
}

func newRpcEntry(callId string, c chan<- *ServiceResponse) *rpcEntry {
	return &rpcEntry{reschan: c}
}

func (r *RpcManager) getEntry(callId string) *rpcEntry {
	//No lock in this function.  should only call from locked section
	p, ok := r.mailbox[callId]
	if ok {
		return p
	}
	return nil
}

func (r *RpcManager) enqueueRPC(callId string, c chan<- *ServiceResponse) (err error) {
	r.Lock()
	defer r.Unlock()

	entry := r.getEntry(callId)
	if entry == nil {
		entry := newRpcEntry(callId, c)
		r.mailbox[callId] = entry
	}
	return nil
}

func (r *RpcManager) replyHandle(callId string, res *ServiceResponse) (err error) {
	r.Lock()
	defer r.Unlock()

	entry := r.getEntry(callId)
	if entry == nil {
		fmt.Println("Unknown callId: ", callId)
		return errors.New("Unknown callId!")
	}
	entry.reschan <- res

	return nil
}
