package goomo

import (
	"time"
)

/*
stops for 5 seconds, then follows postits for 5 seconds;
the succeeding state is FollowPostits
*/

type StopState struct {
	ai                 *MovementAI
	stopActive         bool
	coolDownTimer      *time.Timer
	followPostitsState *FollowPostitsState
}

func NewStopState(ai *MovementAI) *StopState {
	return &StopState{
		ai:                 ai,
		stopActive:         false,
		coolDownTimer:      nil,
		followPostitsState: NewFollowPostitsState(ai),
	}
}

func (s *StopState) name() string {
	return "Stop"
}

func (s *StopState) id() StateId {
	return Stop
}

func (s *StopState) start() {
	s.stopActive = true

	stopTime := 5 * time.Second
	stopTimer := time.NewTimer(stopTime)
	go func() {
		<-stopTimer.C
		s.stopActive = false
	}()

	s.coolDownTimer = time.NewTimer(5*time.Second + stopTime)
	go func() {
		<-s.coolDownTimer.C

		// stop complete
		s.ai.setState(NewFollowPostitsState(s.ai))
	}()
}

func (s *StopState) stop() {
	s.coolDownTimer.Stop()
}

func (s *StopState) handlePostits(postits [][]Feature) (lv, av float32) {
	if s.stopActive {
		return 0.0, 0.0
	} else {
		// coolDownTimerDuration - stopActiveDuration time to drive over stop sign
		lv, av := s.followPostitsState.handlePostits(postits)
		return lv, av
	}
}

func (s *StopState) handleTrafficSigns(trafficSign TrafficSignFeature) (lv, av float32) {
	if s.stopActive {
		return 0.0, 0.0
	} else {
		return s.ai.oldLv, s.ai.oldAv
	}
}
