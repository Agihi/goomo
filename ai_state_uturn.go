package goomo

/*
Turns without moving forward until two postits of different color are detected within 150cm reach;
the succeeding state is FollowPostits
*/

type UturnState struct {
	ai              *MovementAI
	turningVelocity float32
	oldDirection    int
}

func NewUturnState(ai *MovementAI) *UturnState {
	return &UturnState{
		ai:              ai,
		turningVelocity: 0.2,
	}
}

func (u *UturnState) name() string {
	return "Uturn"
}

func (u *UturnState) id() StateId {
	return Uturn
}

func (u *UturnState) start() {
	u.oldDirection = u.ai.direction
}

func (u *UturnState) stop() {}

func (u *UturnState) handlePostits(postits [][]Feature) (lv, av float32) {

	pits0 := postits[0]
	pits1 := postits[1]

	if len(pits0) == 0 || len(pits1) == 0 {
		return 0, u.turningVelocity
	}

	near0 := nearest(pits0)
	near1 := nearest(pits1)

	if near0.imagePos.Y <= PixelHorizon || near1.imagePos.Y <= PixelHorizon {
		return 0, u.turningVelocity
	}

	d0 := SharedDistanceLookup().EuclidianToLoomoPixel(near0.imagePos)
	d1 := SharedDistanceLookup().EuclidianToLoomoPixel(near1.imagePos)

	if d0 < 150 && d1 < 150 {
		currentDirection := signumInt(near0.imagePos.X - near1.imagePos.X)
		if currentDirection != u.oldDirection {
			// uturn complete
			u.ai.setState(NewFollowPostitsState(u.ai))
			return 0, 0
		}
	}

	return 0, u.turningVelocity
}

func (u *UturnState) handleTrafficSigns(trafficSign TrafficSignFeature) (lv, av float32) {
	return u.ai.oldAv, u.ai.oldLv
}
