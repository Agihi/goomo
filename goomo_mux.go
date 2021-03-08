package goomo

import (
	"sync"
)

type Multiplexer interface {
	Multiplex()
}

type MatMultiplexer struct {
	Inbound       chan *ManagedMat
	outboundMutex *sync.Mutex
	outbounds     map[string]chan *ManagedMat
}

func (m *MatMultiplexer) Multiplex() {
	logger.Debug("MatMultiplexer started.")
	for managed := range m.Inbound {
		m.outboundMutex.Lock()
		for _, outbound := range m.outbounds {
			managed.Assign()
			select {
			case outbound <- managed:
			default:
				// nothing happens
				managed.Done()
			}
		}
		managed.Done()
		m.outboundMutex.Unlock()
	}
	//Take Mat and copy it to all outbounds
	logger.Debug("MatMultiplexer stopped.")
}

func (m *MatMultiplexer) Add(id string, receiver chan *ManagedMat) {
	m.outboundMutex.Lock()
	m.outbounds[id] = receiver
	m.outboundMutex.Unlock()
}

func (m *MatMultiplexer) Has(id string) bool {
	m.outboundMutex.Lock()
	_, ok := m.outbounds[id]
	m.outboundMutex.Unlock()
	return ok
}

func (m *MatMultiplexer) Remove(id string) {
	m.outboundMutex.Lock()
	delete(m.outbounds, id)
	m.outboundMutex.Unlock()
}

type JPGMultiplexer struct {
	Inbound       chan JPG
	outboundMutex *sync.Mutex
	outbounds     map[string]chan JPG
}

func (j *JPGMultiplexer) Multiplex() {
	logger.Debug("JPGMultiplexer started.")
	for jpg := range j.Inbound {
		j.outboundMutex.Lock()
		for _, outbound := range j.outbounds {
			select {
			case outbound <- jpg:
			default:
				// nothing happens
			}
		}
		j.outboundMutex.Unlock()
	}

	logger.Debug("JPGMultiplexer stopped.")
}

func (j *JPGMultiplexer) Add(id string, receiver chan JPG) {
	j.outboundMutex.Lock()
	j.outbounds[id] = receiver
	j.outboundMutex.Unlock()
}

func (j *JPGMultiplexer) Has(id string) bool {
	j.outboundMutex.Lock()
	_, ok := j.outbounds[id]
	j.outboundMutex.Unlock()
	return ok
}

func (j *JPGMultiplexer) Remove(id string) {
	j.outboundMutex.Lock()
	delete(j.outbounds, id)
	j.outboundMutex.Unlock()
}
