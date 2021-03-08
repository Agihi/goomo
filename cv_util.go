package goomo

import (
	"gocv.io/x/gocv"
	"image"
	"math"
)

func Area(r *image.Rectangle) int {
	size := r.Size()
	return size.X * size.Y
}

func distance(r1 *image.Rectangle, r2 *image.Rectangle) float64 {
	/*
	 *  dist is the euclidean distance between points
	 *  rect. 1 is formed by points (x1, y1) and (x1b, y1b)
	 *  rect. 2 is formed by points (x2, y2) and (x2b, y2b)
	 */
	x1 := r1.Min.X
	y1 := r1.Min.Y
	x1b := r1.Max.X
	y1b := r1.Max.Y

	x2 := r2.Min.X
	y2 := r2.Min.Y
	x2b := r2.Max.X
	y2b := r2.Max.Y

	left := x2b < x1
	right := x1b < x2
	bottom := y2b < y1
	top := y1b < y2

	if top && left {
		return dist(x1, y1b, x2b, y2)
	} else if left && bottom {
		return dist(x1, y1, x2b, y2b)
	} else if bottom && right {
		return dist(x1b, y1, x2, y2b)
	} else if right && top {
		return dist(x1b, y1b, x2, y2)
	} else if left {
		return float64(x1 - x2b)
	} else if right {
		return float64(x2 - x1b)
	} else if bottom {
		return float64(y1 - y2b)
	} else if top {
		return float64(y2 - y1b)
	} else {
		return 0.
	}
}

func dist(i int, i2 int, i3 int, i4 int) float64 {
	dx := i - i3
	dy := i2 - i4
	return math.Sqrt(float64(dx*dx + dy*dy))
}

func findMiddlePoint(rect *image.Rectangle) image.Point {
	x := (rect.Min.X + rect.Max.X) / 2
	y := (rect.Min.Y + rect.Max.Y) / 2
	return image.Point{X: x, Y: y}
}

func FindTrue(row, col *int, mat *gocv.Mat) bool {
	// log.Printf("row: %v, col: %v mat.Rows()/2: %v", *row, *col, mat.Rows()/2)
	for ; *row >= mat.Rows()/2; *row-- {
		// log.Printf("row: %v, col: %v ", *row, *col)
		for ; *col >= 0; *col-- {
			if mat.GetUCharAt(*row, *col) != 0 {
				return true
			}
		}
		*col = mat.Cols()
	}
	return false
}
func dilation(mat *gocv.Mat) {
	gocv.Dilate(*mat, mat, kernel)
}

func erode(mat *gocv.Mat) {
	gocv.Erode(*mat, mat, kernel)
}
