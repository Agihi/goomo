package goomo

type IdleState struct {
	ai *MovementAI
}

func NewIdleState(ai *MovementAI) *IdleState {
	return &IdleState{ai: ai}
}

func (i *IdleState) name() string {
	return "Idle"
}

func (i *IdleState) id() StateId {
	return Idle
}

func (i *IdleState) handlePostits(postits [][]Feature) (lv, av float32) {
	return 0, 0
}

func (i *IdleState) handleTrafficSigns(trafficSign TrafficSignFeature) (lv, av float32) {
	return 0, 0
}

func (i *IdleState) start() {
	i.ai.setVelocities(0, 0)
}

func (i *IdleState) stop() {}
