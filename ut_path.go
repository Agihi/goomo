package goomo

import (
	"fmt"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
	"image"
	"image/color"
)

type BezierPathThroughKnots struct {
	knots         []vg.Point
	controlPoints [][2]vg.Point
}

// 0 <= t <= 1
func (b BezierPathThroughKnots) Point(t float64) vg.Point {
	n := len(b.knots) - 1

	// ti is new value between 0 and 1 for the i-curve
	i := int(float64(n) * t)
	ti := float64(n)*t - float64(i)

	//log.Printf("len(b.knots) == %v && i == %v", len(b.knots), i)
	ki0 := b.knots[i]
	ki1 := b.knots[i+1]
	ci0 := b.controlPoints[i][0]
	ci1 := b.controlPoints[i][1]

	a0 := vg.Length((1 - ti) * (1 - ti) * (1 - ti))
	a1 := vg.Length(3 * (1 - ti) * (1 - ti) * ti)
	a2 := vg.Length(3 * (1 - ti) * ti * ti)
	a3 := vg.Length(ti * ti * ti)

	p := vg.Point{0, 0}

	return p.Add(ki0.Scale(a0)).Add(ci0.Scale(a1)).Add(ci1.Scale(a2)).Add(ki1.Scale(a3))
}

// https://www.stkent.com/2015/07/03/building-smooth-paths-using-bezier-curves.html
func NewBezierPath(knots []vg.Point) (BezierPathThroughKnots, error) {
	if len(knots) < 2 {
		return BezierPathThroughKnots{}, fmt.Errorf("knots has to be of length >= 2")
	}
	// repeat last knot to "fake" an open bezier path
	// (such that boundary conditions do not hold for last knot)
	knots = append(knots, knots[len(knots)-1])

	n := len(knots) - 1
	a := make([]float64, n)
	b := make([]float64, n)
	y := make([]vg.Point, n)

	// setting up the linear system
	for i := 0; i < n; i++ {
		if i == 0 {
			a[i] = 1
			b[i] = 2
		} else if i == n-1 {
			a[i] = 2
			b[i] = 7
		} else {
			a[i] = 1
			b[i] = 4
		}
	}

	y[0] = knots[0].Add(knots[1].Scale(2))
	for i := 1; i < n-1; i++ {
		y[i] = knots[i].Scale(2).Add(knots[i+1]).Scale(2)
	}
	y[n-1] = knots[n-1].Scale(8).Add(knots[n])

	// computing control points
	controlPoints0 := computeControlPoints(a, b, y)
	controlPoints := make([][2]vg.Point, n)

	for i := 0; i <= n-2; i++ {
		controlPoints[i][0] = controlPoints0[i]
		controlPoints[i][1] = knots[i+1].Scale(2).Sub(controlPoints0[i+1])
	}
	controlPoints[n-1][0] = controlPoints0[n-1]
	controlPoints[n-1][1] = knots[n].Add(controlPoints0[n-1]).Scale(1 / 2)

	knots = knots[:n] // remove repeated knot

	return BezierPathThroughKnots{
		knots:         knots,
		controlPoints: controlPoints,
	}, nil
}

// https://en.wikipedia.org/wiki/Tridiagonal_matrix_algorithm
func computeControlPoints(a, b []float64, d []vg.Point) []vg.Point {
	n := len(a)

	cnew := make([]float64, n)
	dnew := make([]vg.Point, n)

	cnew[0] = 1 / b[0]
	dnew[0] = d[0].Scale(vg.Length(1 / b[0]))

	for i := 1; i < n; i++ {
		cnew[i] = 1 / (b[i] - a[i]*cnew[i-1])
		dnew[i] = d[i].Sub(dnew[i-1].Scale(vg.Length(a[i])))
		dnew[i] = dnew[i].Scale(vg.Length(1 / (b[i] - a[i]*cnew[i-1])))
	}

	x := make([]vg.Point, n)

	x[n-1] = dnew[n-1]
	for i := n - 2; i >= 0; i-- {
		x[i] = dnew[i].Sub(x[i+1].Scale(vg.Length(cnew[i])))
	}

	return x
}

func BezierPath(postits []Feature, startX, startY float64) (BezierPathThroughKnots, error) {
	var current = vg.Point{
		X: vg.Length(startX),
		Y: vg.Length(startY),
	}

	var knots = []vg.Point{current}

	// copy postit group as positions are deleted
	copiedPits := make([]Feature, len(postits))
	copy(copiedPits, postits)

	for len(copiedPits) > 0 {
		nearest, d, err := extractNearest(current, &copiedPits)

		// only consider postits in 200cm distance to current
		if err != nil || d > 200 {
			break
		}

		knots = append(knots, nearest)
		current = nearest
	}
	return NewBezierPath(knots)
}

// gets postit that is nearest to pos and removes it from pits.Positions
func extractNearest(pos vg.Point, pits *[]Feature) (vg.Point, float64, error) {
	var nearest vg.Point
	var index = -1
	var distance = 500.0 // 5m

	for i, q := range *pits {

		if q.imagePos.Y < PixelHorizon {
			continue
		}

		qdx, qdy := SharedDistanceLookup().Distance(q.imagePos.X, q.imagePos.Y)

		d := SharedDistanceLookup().Euclidean(float64(pos.X), float64(pos.Y), qdx, qdy)

		if d < distance {
			nearest = vg.Point{vg.Length(qdx), vg.Length(qdy)}
			index = i
			distance = d
		}
	}

	if index >= 0 {
		remove(pits, index)
		return nearest, distance, nil
	} else {
		return nearest, distance, fmt.Errorf("no nearest point found")
	}

}

// remove position from Features at index i
func remove(pits *[]Feature, i int) {
	(*pits)[i] = (*pits)[len(*pits)-1]
	(*pits) = (*pits)[:len(*pits)-1]
}

func TestBezierPath() {

	p := []Feature{
		{imagePos: image.Point{340, 480}},
		{imagePos: image.Point{360, 470}},
		{imagePos: image.Point{360, 400}},
		{imagePos: image.Point{370, 450}},
	}

	path, _ := BezierPath(p, 40, 0)
	fmt.Println(path)
}

func TestPath() {

	p1 := vg.Point{X: 0, Y: 0}
	p2 := vg.Point{X: 0.5, Y: 1.5}
	p3 := vg.Point{X: 1, Y: 1}
	p4 := vg.Point{1.5, 0.5}
	p5 := vg.Point{2, 0.5}

	ps := []vg.Point{p1, p2, p3, p4, p5}

	curve, _ := NewBezierPath(ps)

	plt, _ := plot.New()

	n := 100
	points := make(plotter.XYs, n+3)
	for i := 0; i < n; i++ {
		p := curve.Point(float64(i) / float64(n))
		points[i] = plotter.XY{float64(p.X), float64(p.Y)}
	}

	scatter, _ := plotter.NewScatter(points)
	scatter.GlyphStyle.Color = color.RGBA{R: 255, B: 128, A: 255}

	knots := make(plotter.XYs, len(ps))

	for i, p := range ps {
		knots[i] = plotter.XY{float64(p.X), float64(p.Y)}
	}

	scatterKnots, _ := plotter.NewScatter(knots)

	lines, _ := plotter.NewLine(points)

	// do not close graph
	j := 0
	for i := len(lines.XYs) - 1; i >= 0; i-- {
		if lines.XYs[i].X == 0 && lines.XYs[i].Y == 0 {
			j += 1
		} else {
			break
		}
	}

	lines.XYs = lines.XYs[:len(lines.XYs)-j]

	//plt.Add(scatter)
	plt.Add(lines)
	plt.Add(scatterKnots)

	plt.Save(4*vg.Inch, 4*vg.Inch, "/home/markus/points.png")
}
