package goomo

import (
	"fmt"
	"log"
	"math"
)

type MovementAI struct {
	OutboundCmds        chan Command
	InboundPostits      chan [][]Feature
	InboundTrafficSigns chan *TrafficSignFeature
	oldAv               float32
	oldLv               float32
	state               MovementAIState
	maxAv               float32
	maxLv               float32
	direction           int
}

type StateId uint8

const (
	Idle StateId = iota
	FollowPostits
	Uturn
	Stop
)

type MovementAIState interface {
	name() string
	id() StateId
	handlePostits(postits [][]Feature) (lv, av float32)
	handleTrafficSigns(trafficSign TrafficSignFeature) (lv, av float32)
	start()
	stop()
}

func (m *MovementAI) setState(state MovementAIState) {
	if state == nil {
		log.Println("Could not set state.")
		return
	}

	// stop old state
	if m.state != nil {
		m.state.stop()
	}

	log.Printf("MovementAI in state %v", state.name())

	// start new state
	m.state = state
	m.state.start()
}

func NewMovementAIState(ai *MovementAI, trafficSign *TrafficSignFeature) (MovementAIState, error) {
	switch trafficSign.Name {
	case stopSign:
		return NewStopState(ai), nil
	case uturnSign:
		return NewUturnState(ai), nil
	default:
		return nil, fmt.Errorf("Could not find State for %v", trafficSign.Name)
	}
}

func (m *MovementAI) BigBrainPostitAI() {
	logger.Debug("PostitAI started.")

	av := float32(0)
	lv := float32(0)

	for ps := range m.InboundPostits {
		m.trackDirection(ps)
		lv, av = m.state.handlePostits(ps)
		m.setVelocities(lv, av)
	}

	if m.state.id() == Uturn || m.state.id() == FollowPostits {
		m.setState(NewIdleState(m))
	}

	logger.Debug("PostitAI stopped.")
}

func (m *MovementAI) BigBrainTrafficSignAI() {
	logger.Debug("TrafficSignAI started.")

	av := float32(0)
	lv := float32(0)

	for ts := range m.InboundTrafficSigns {
		lv, av = m.state.handleTrafficSigns(*ts)
		m.setVelocities(lv, av)
	}

	if m.state.id() == Stop {
		m.setState(NewIdleState(m))
	}

	logger.Debug("TrafficSignAI stopped.")
}

func (m *MovementAI) trackDirection(ps [][]Feature) {
	if len(ps[0]) == 0 || len(ps[1]) == 0 {
		return
	}
	near0 := nearest(ps[0])
	near1 := nearest(ps[1])

	m.direction = signumInt(near0.imagePos.X - near1.imagePos.X)
}

func (m *MovementAI) setVelocities(lv, av float32) {
	if math.Abs(float64(lv)) > float64(m.maxAv) || math.Abs(float64(av)) > float64(m.maxAv) {
		//log.Println("velocities out of bounds", lv, av)
		return
	}

	if m.oldLv != lv {
		//log.Printf("Set linear velocity: %v", lv)
		m.OutboundCmds <- &CLVLCommand{Lv: lv}
		m.oldLv = lv
	}
	if m.oldAv != av {
		//log.Printf("Set angular velocity: %v", av)
		m.OutboundCmds <- &CAVLCommand{Av: av}
		m.oldAv = av
	}
}
