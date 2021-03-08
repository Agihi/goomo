package goomo

func NewMovementAI(outboundCmds chan Command) *MovementAI {
	mov := MovementAI{
		OutboundCmds: outboundCmds,
		maxAv:        0.4,
		maxLv:        0.4,
		direction:    0,
	}

	mov.setState(NewIdleState(&mov))

	return &mov
}

func (m *MovementAI) StartPostitAI(inboundPostits chan [][]Feature) {
	m.InboundPostits = inboundPostits
	m.setState(NewFollowPostitsState(m))
	m.BigBrainPostitAI()
}

func (m *MovementAI) StartTrafficSignAI(inboundTrafficSigns chan *TrafficSignFeature) {
	m.InboundTrafficSigns = inboundTrafficSigns
	m.BigBrainTrafficSignAI()
}
