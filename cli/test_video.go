package main

import (
	"gocv.io/x/gocv"
	"image/color"
	goomo2 "iteragit.iteratec.de/go_loomo_go/goomo"
	"log"
	"sync"
)

var white = color.RGBA{255, 255, 255, 1}
var red = color.RGBA{255, 0, 0, 1}
var blackScalar = gocv.NewScalar(0, 0, 0, 1)
var kernel = gocv.NewMatWithSizeFromScalar(gocv.NewScalar(1, 1, 1, 1), 3, 3, gocv.MatTypeCV8U)
var whiteMat = gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255, 255, 255, 1), 480, 640, gocv.MatTypeCV8UC3)

const I = 255

func main() {
	findGrid4()
}

func testVideo() {
	vc, err := gocv.VideoCaptureFile("/home/wedl/Pictures/test2.h264")
	if err != nil {
		log.Fatal(err)
	}

	in := make(chan *gocv.Mat)
	defer close(in)

	chanPostits := make(chan goomo2.Postits)
	defer close(chanPostits)

	mat := gocv.NewMat()
	defer mat.Close()

	descriptions := make(goomo2.Descriptions, 3)
	descriptions[0] = goomo2.PostitDescription{
		goomo2.HSV{
			H: 34,
			S: 71,
			V: 69,
		},
		goomo2.HSVB{
			HB: 4,
			SB: 20,
			VB: 50,
		},
	}
	descriptions[1] = goomo2.PostitDescription{
		goomo2.HSV{
			H: 86,
			S: 52,
			V: 76,
		},
		goomo2.HSVB{
			HB: 4,
			SB: 20,
			VB: 50,
		},
	}
	descriptions[2] = goomo2.PostitDescription{
		goomo2.HSV{
			H: 356,
			S: 37,
			V: 84,
		},
		goomo2.HSVB{
			HB: 4,
			SB: 20,
			VB: 50,
		},
	}
	go descriptions.FindPostits(in, chanPostits)

	for vc.Read(&mat) {
		in <- &mat
		<-chanPostits
	}
}

func canny(in chan *gocv.Mat, out chan *gocv.Mat) {
	res := gocv.NewMat()
	defer res.Close()
	for mat := range in {
		gocv.Canny(*mat, &res, 50, 200)
		out <- &res
	}
}

func findGrid3(in chan *gocv.Mat, out chan *gocv.Mat) {
	res := gocv.NewMat()
	defer res.Close()

	blobDetector := gocv.NewSimpleBlobDetector()
	defer blobDetector.Close()

	color := color.RGBA{255, 255, 255, 1}
	for mat := range in {
		// gocv.Canny(*mat, &res, 50, 50)
		keyPoints := blobDetector.Detect(*mat)
		gocv.DrawKeyPoints(*mat, keyPoints, mat, color, gocv.DrawRichKeyPoints)
		out <- mat
	}
}

func findGrid4(in chan *gocv.Mat, out chan *gocv.Mat) {
	res := gocv.NewMat()
	defer res.Close()
	labels := gocv.NewMat()
	defer labels.Close()

	boundary := float64(80)
	lb := gocv.NewScalar(150-boundary, 150-boundary, 150-boundary, 1)
	ub := gocv.NewScalar(150+boundary, 150+boundary, 150+boundary, 1)

	color := color.RGBA{255, 255, 255, 1}
	for mat := range in {
		gocv.InRangeWithScalar(*mat, lb, ub, &res)
		contours := gocv.FindContours(res, gocv.RetrievalTree, gocv.ChainApproxSimple)
		// gocv.DrawContours(mat, contours, -1 , color, 1)
		// log.Printf("contour length: %v", len(contours))
		gocv.FillPoly(&res, contours, color)
		x := gocv.ConnectedComponents(res, &labels) // rest of the arguments for ConnectedComponentsWithParams: , 8, gocv.MatTypeCV16U, gocv.CCL_DEFAULT)
		log.Printf("x: %v\n", x)
		out <- &res
	}
}

func connectedComponents(in chan *gocv.Mat, out chan *gocv.Mat) {
	res := gocv.NewMat()
	defer res.Close()
	labeledMat := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV32S)
	defer labeledMat.Close()
	copyMat := gocv.NewMatWithSize(labeledMat.Rows(), labeledMat.Cols(), gocv.MatTypeCV8U)
	defer copyMat.Close()

	boundary := float64(80)
	lb := gocv.NewScalar(150-boundary, 150-boundary, 150-boundary, 1)
	ub := gocv.NewScalar(150+boundary, 150+boundary, 150+boundary, 1)

	for mat := range in {
		gocv.InRangeWithScalar(*mat, lb, ub, &res)
		// contours := gocv.FindContours(res, gocv.RetrievalTree, gocv.ChainApproxSimple)
		// gocv.DrawContours(mat, contours, -1 , color, 1)
		// log.Printf("contour length: %v", len(contours))
		// gocv.FillPoly(&res, contours, white)
		// nLabels :=
		erode(mat)
		dilation(mat)
		dilation(mat)
		erode(mat)
		gocv.ConnectedComponents(res, &labeledMat) // rest of the arguments for ConnectedComponentsWithParams: , 8, gocv.MatTypeCV16U, gocv.CCL_DEFAULT)
		//colors := make(nL)
		for row := 0; row < labeledMat.Rows(); row++ {
			for col := 0; col < labeledMat.Cols(); col++ {
				val := labeledMat.GetIntAt(row, col)
				if val >= 2 && val <= 5 {
					copyMat.SetUCharAt(row, col, uint8(255))
				} else {
					copyMat.SetUCharAt(row, col, uint8(0))
				}
			}
		}
		for i := 0; i < 5; i++ {
			zhangSuenThinning(&copyMat)
		}

		// gocv.Threshold(rgbMat, &rgbMat, float32(i-1), 255, gocv.ThresholdBinary)
		// PrintMat(&labeledMat)
		out <- &copyMat
	}
}

func betterThinning(mat *gocv.Mat) {
	p := make([]uint8, 9)

	b := func(x []uint8, i int) uint8 {
		return x[2*i-2] * (x[2*i-1]/255 + x[2*i]/255)
	}

	logicOr := func(x, y uint8) uint8 {
		if x/255+y/255 >= 1 {
			return 1
		} else {
			return 0
		}
	}

	n1 := func(x []uint8) uint8 {
		var sum uint8
		for k := 1; k <= 4; k++ {
			sum += logicOr(x[2*k-2], x[2*k-1])
		}
		return sum
	}

	n2 := func(x []uint8) uint8 {
		var sum uint8
		for k := 1; k <= 4; k++ {
			sum += logicOr(x[2*k-1], x[2*k])
		}
		return sum
	}

	g1 := func(x []uint8) bool {
		return (b(x, 1) + b(x, 2) + b(x, 3) + b(x, 4)) == 1
	}

	g2 := func(x []uint8) bool {
		n1p := n1(x)
		n2p := n2(x)
		if n1p <= n2p {
			return n1p >= 2 && n1p <= 3
		} else {
			return n2p >= 2 && n2p <= 3
		}
	}

	g3 := func(x []uint8) bool {
		var disjunction uint8
		if x[1]/255+x[2]/255+x[7]/255 >= 1 {
			disjunction = 1
		} else {
			disjunction = 0
		}
		return disjunction*x[0] == 0
	}

	g3prime := func(x []uint8) bool {
		var disjunction uint8
		if x[5]/255+x[6]/255+x[3]/255 >= 1 {
			disjunction = 1
		} else {
			disjunction = 0
		}
		return disjunction*x[4] == 0
	}

	for row := 1; row < mat.Rows()-1; row++ {
		for col := 1; col < mat.Cols()-1; col++ {
			if mat.GetUCharAt(row, col) == 0 {
				continue
			}
			p[0] = mat.GetUCharAt(row, col+1)
			p[1] = mat.GetUCharAt(row-1, col+1)
			p[2] = mat.GetUCharAt(row-1, col)
			p[3] = mat.GetUCharAt(row-1, col-1)
			p[4] = mat.GetUCharAt(row, col-1)
			p[5] = mat.GetUCharAt(row+1, col-1)
			p[6] = mat.GetUCharAt(row+1, col)
			p[7] = mat.GetUCharAt(row+1, col+1)
			p[8] = mat.GetUCharAt(row, col+1)
			if g1(p) && g2(p) && g3(p) {
				mat.SetUCharAt(row, col, 0)
			}
			if g1(p) && g2(p) && g3prime(p) {
				mat.SetUCharAt(row, col, 0)
			}
		}
	}
}

func zhangSuenThinning(mat *gocv.Mat) {
	var waitgroup sync.WaitGroup
	for row := 1; row < mat.Rows()-1; row++ {
		for col := 1; col < mat.Cols()-1; col++ {
			waitgroup.Add(1)
			go zhangSuenEval(mat, row, col, &waitgroup)
		}
	}
	waitgroup.Wait()
}

func zhangSuenEval(mat *gocv.Mat, row, col int, group *sync.WaitGroup) {
	defer group.Done()
	p := make([]uint8, 9)
	if mat.GetUCharAt(row, col) == 0 {
		return
	}

	p[0] = mat.GetUCharAt(row-1, col)
	p[2] = mat.GetUCharAt(row, col+1)
	p[4] = mat.GetUCharAt(row+1, col)
	if p[0]*p[2]*p[4] != 0 {
		return
	}
	p[6] = mat.GetUCharAt(row, col-1)
	if p[2]*p[4]*p[6] != 0 {
		return
	}
	p[1] = mat.GetUCharAt(row-1, col+1)
	p[3] = mat.GetUCharAt(row+1, col+1)
	p[5] = mat.GetUCharAt(row+1, col-1)
	p[7] = mat.GetUCharAt(row-1, col-1)
	p[8] = mat.GetUCharAt(row-1, col)
	count := 0
	countNeighbors := 0
	for i := 1; i < len(p); i++ {
		if p[i] > p[i-1] {
			count++
		}
		if p[i] != 0 {
			countNeighbors++
		}
	}
	if count != 1 || countNeighbors > 6 || countNeighbors < 2 {
		return
	}
	mat.SetUCharAt(row, col, 0)
}

func dilation(mat *gocv.Mat) {
	gocv.Dilate(*mat, mat, kernel)
}

func erode(mat *gocv.Mat) {
	gocv.Erode(*mat, mat, kernel)
}

// var kernel = gocv.NewMatWithSizeFromScalar(gocv.NewScalar(1, 1, 1, 1), 3, 3, gocv.MatTypeCV8U)

func thinning(mat *gocv.Mat) {
	for i := 0; i < 3; i++ {
		erode(mat)
	}
}

// func thinning(mat *gocv.Mat)
