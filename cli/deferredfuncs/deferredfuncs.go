package deferredfuncs

import "sync"

type DeferredFuncs struct {
	mutex sync.RWMutex
	funcs []func()
}

func New(funcs ...func()) *DeferredFuncs {
	return &DeferredFuncs{
		funcs: funcs,
	}
}

func (d *DeferredFuncs) Register(funcs ...func()) {
	d.mutex.Lock()
	d.funcs = append(d.funcs, funcs...)
	d.mutex.Unlock()
}

func (d *DeferredFuncs) Run() {
	d.mutex.RLock()
	for _, f := range d.funcs {
		f()
	}
	d.mutex.RUnlock()
}
