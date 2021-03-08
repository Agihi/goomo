package goomo

import (
	"gonum.org/v1/plot/vg"
	"math"
)

/*
Assumes that there are two lanes of postits, calculates a bezierpath for each of the lanes
and follows the middlepath between them.
If a trafficsign is detected 10 times in a row, the corresponding state will be set.
If no postits are detected a uturn will be initiated.
*/

type FollowPostitsState struct {
	ai           *MovementAI
	signCounter  int
	signTreshold int
	signIndex    int64
}

func NewFollowPostitsState(ai *MovementAI) *FollowPostitsState {
	return &FollowPostitsState{
		ai:           ai,
		signCounter:  0,
		signTreshold: 10,
		signIndex:    -1,
	}
}

func (f *FollowPostitsState) name() string {
	return "FollowPostits"
}

func (f *FollowPostitsState) id() StateId {
	return FollowPostits
}

func (f *FollowPostitsState) start() {}

func (f *FollowPostitsState) stop() {}

func (f *FollowPostitsState) handlePostits(postits [][]Feature) (lv, av float32) {
	maxAv := f.ai.maxAv
	maxLv := f.ai.maxLv

	var pits0 = postits[0] // orange
	var pits1 = postits[1] // green

	if len(pits0) == 0 || len(pits1) == 0 {
		// direction 1 	-> pits0 on the right, pits1 on the left
		// direction -1	-> pits0 on the left, pits1 on the right
		if len(pits0) > 0 {
			av = 0.1 * float32(f.ai.direction)
			lv = 0.1
		} else if len(pits1) > 0 {
			av = -0.1 * float32(f.ai.direction)
			lv = 0.1
		} else {
			av = 0.0
			lv = 0.0

			// no postits on screen -> uturn
			f.ai.setState(NewUturnState(f.ai))
		}
	} else {
		var err error
		left, err := BezierPath(pits0, -40, 0)
		if err != nil {
			return f.ai.oldLv, f.ai.oldAv
		}
		right, err := BezierPath(pits1, 40, 0)
		if err != nil {
			return f.ai.oldLv, f.ai.oldAv
		}
		// offset 8 cm to left because camera is not in center
		disposition := calculateDisposition(left, right, 10, 0.5, -8)
		if math.Abs(disposition) > 50 {
			av = maxAv * float32(-signumF64(disposition))
		} else {
			// disposition/40 rounded to 2 decimal points
			av = maxAv * float32(math.Round(-disposition*2.5)/100)
		}
		lv = maxLv - float32(math.Abs(float64(av)))
	}

	return lv, av
}

func (f *FollowPostitsState) handleTrafficSigns(trafficSign TrafficSignFeature) (lv, av float32) {
	if trafficSign.Index == f.signIndex {
		// avoid false positives
		f.signCounter++
		distance := SharedDistanceLookup().EuclideanToLoomDistance(trafficSign.realPos)

		if f.signCounter > f.signTreshold && distance < 100 {
			state, err := NewMovementAIState(f.ai, &trafficSign)
			if err != nil {
				return
			}
			f.ai.setState(state)
		}
	} else {
		f.signCounter = 0
		f.signIndex = trafficSign.Index
	}

	return f.ai.oldLv, f.ai.oldAv
}

func calculateDisposition(left, right BezierPathThroughKnots, sampleSize int, visionDistance float64, offsetX vg.Length) float64 {
	sample := make([]vg.Point, 0, sampleSize)
	for i := 0; i < sampleSize; i++ {
		var selection = visionDistance * (float64(i+1) / float64(sampleSize))
		leftPoint := left.Point(selection)
		rightPoint := right.Point(selection)

		// TODO parametrize this:
		sampledPoint := vg.Point{X: (leftPoint.X+rightPoint.X)/2 + offsetX, Y: (leftPoint.Y + rightPoint.Y) / 2}
		sample = append(sample, sampledPoint)
	}
	return float64(pointAvg(sample...).X)
}

func nearest(postits []Feature) Feature {
	maxY := 0
	f := Feature{}

	for _, p := range postits {
		if p.imagePos.Y > maxY {
			f = p
			maxY = p.imagePos.Y
		}
	}

	return f
}

func pointAvg(points ...vg.Point) vg.Point {
	avg := vg.Point{}
	for _, point := range points {
		avg.X += point.X
		avg.Y += point.Y
	}
	avg.X /= vg.Length(len(points))
	avg.Y /= vg.Length(len(points))
	return avg
}
