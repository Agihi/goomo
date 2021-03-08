package goomo

import (
	"gocv.io/x/gocv"
	"gonum.org/v1/plot/vg"
	"image"
	"log"
	"math"
	"sync"
)

const (
	verticalAlpha  = 0.7086 // 40.6 degrees
	horizontalBeta = 0.7854 // 45 degrees

	eyeHeight = 60 // cm

	PixelWidth   = 640
	PixelHeight  = 480
	PixelHorizon = 270
)

type DistanceLookup struct {
	xDistances gocv.Mat
	yDistances gocv.Mat
}

func NewDistanceLookup() *DistanceLookup {
	dl := DistanceLookup{}

	dl.xDistances = gocv.NewMatWithSize(PixelWidth, PixelHeight-PixelHorizon, gocv.MatTypeCV32F)
	dl.yDistances = gocv.NewMatWithSize(PixelWidth, PixelHeight-PixelHorizon, gocv.MatTypeCV32F)

	dl.init()
	return &dl
}

var distInstance *DistanceLookup
var distOnce sync.Once

func SharedDistanceLookup() *DistanceLookup {
	distOnce.Do(func() {
		distInstance = NewDistanceLookup()
	})
	return distInstance
}

func (d *DistanceLookup) init() {
	for row := PixelHorizon; row < PixelHeight; row++ {
		for col := 0; col < PixelWidth; col++ {
			d.calcDistance(col, row)
		}
	}
}

// pixel coordinates have to be transformed,
// such that y-distances match the image
// the transformation was calculated manually with GIMP
func transform(x, y int) (float64, float64) {
	// "squishes" picture down along the y-axis
	// (1.0055 factor of x omitted)
	return float64(x), 0.005*float64(x) + 0.9*float64(y) + 49.0
}

func (d *DistanceLookup) calcDistance(x, y int) {
	tX, tY := transform(x, y)
	v := PixelHeight/2 - (PixelHeight - tY)
	vTanGamma := v * math.Tan(verticalAlpha) / (PixelHeight / 2)
	dy := eyeHeight / vTanGamma
	d.yDistances.SetFloatAt(x, y-PixelHorizon, float32(dy))

	h := tX - PixelWidth/2
	width := math.Tan(horizontalBeta) * eyeHeight / vTanGamma
	dx := h / (PixelWidth / 2) * width
	d.xDistances.SetFloatAt(x, y-PixelHorizon, float32(dx))
}

func (d *DistanceLookup) Distance(x, y int) (dx, dy float64) {
	if y < PixelHorizon {
		log.Printf("(%d, %d) is out ouf bounds.", x, y)
	}

	dx = float64(d.xDistances.GetFloatAt(x, y-PixelHorizon))
	dy = float64(d.yDistances.GetFloatAt(x, y-PixelHorizon))

	return dx, dy
}

func (d *DistanceLookup) DistanceWithErrorBounds(x, y int) (dX, dY [3]float64) {
	if y < PixelHorizon {
		log.Printf("(%d, %d) is out ouf bounds.", x, y)
	}

	dX[1] = float64(d.xDistances.GetFloatAt(x, y-PixelHorizon))
	dY[1] = float64(d.yDistances.GetFloatAt(x, y-PixelHorizon))

	minError := 0.0
	maxError := 5.0 // cm
	k := -(maxError - minError) / (PixelHeight - PixelHorizon)
	c := maxError - k*float64(PixelHorizon)

	estError := k*float64(y) + c
	dX[0] = dX[1] - estError
	dX[2] = dX[1] + estError
	dY[0] = dY[1] - estError
	dY[2] = dY[1] + estError
	return
}

// x, y in cm
func (d *DistanceLookup) Pixel(x, y float64) (px, py int) {
	py = d.searchY(y, PixelHorizon, PixelHeight)
	px = d.searchX(x, py, 0, PixelWidth)
	return
}

// returns the euclidean distance between two pixel points in cm
func (d *DistanceLookup) EuclideanBetweenPixels(x1, y1, x2, y2 int) float64 {
	dx1, dy1 := d.Distance(x1, y1)
	dx2, dy2 := d.Distance(x2, y2)

	diffx := dx1 - dx2
	diffy := dy1 - dy2

	return math.Sqrt(diffx*diffx + diffy*diffy)
}

// returns the euclidean between (x1,y1) and (x2, y2)
func (d *DistanceLookup) Euclidean(x1, y1, x2, y2 float64) float64 {
	diffx := x1 - x2
	diffy := y1 - y2

	return math.Sqrt(diffx*diffx + diffy*diffy)
}

// returns the euclidean distance in cm from (0,0) (Loomo) to p, where p is a real point (in cm)
func (d *DistanceLookup) EuclideanToLoomDistance(p vg.Point) float64 {
	return d.Euclidean(0, 0, float64(p.X), float64(p.Y))
}

// returns the euclidean distance in cm from (0,0) (Loomo) to p, where p is a pixel point
func (d *DistanceLookup) EuclidianToLoomoPixel(p image.Point) float64 {
	dx, dy := d.Distance(p.X, p.Y)
	return d.Euclidean(0, 0, dx, dy)
}

// binary search
func (d *DistanceLookup) searchX(x float64, py, start, end int) int {
	if end-start <= 1 {
		return start
	}

	mid := (start + end) / 2
	comp, _ := d.Distance(mid, py)

	if x > comp {
		return d.searchX(x, py, mid, end)
	} else {
		return d.searchX(x, py, start, mid)
	}
}

// binary search
func (d *DistanceLookup) searchY(y float64, start, end int) int {
	if end-start <= 1 {
		return start
	}

	mid := (start + end) / 2
	_, comp := d.Distance(0, mid)

	if y > comp {
		return d.searchY(y, start, mid)
	} else {
		return d.searchY(y, mid, end)
	}

}
