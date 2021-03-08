package goomo

import (
	"gocv.io/x/gocv"
	"image"
	"log"
)

type HSV struct {
	H float64
	S float64
	V float64
}

type HSVB struct {
	HB float64
	SB float64
	VB float64
}

type HSVDescription struct {
	HSV
	HSVB
}

var lowerImageRect = image.Rect(0, 270, 640, 480)

func (description HSVDescription) findColorFeature(inbound chan *ManagedMat, outbound chan []Feature) {
	hsvMat := gocv.NewMat()
	defer hsvMat.Close()

	mask := gocv.NewMatWithSize(480, 640, gocv.MatTypeCV8U)
	defer mask.Close()

	for managedMat := range inbound {
		mat := managedMat.mat
		gocv.CvtColor(*mat, &hsvMat, gocv.ColorBGRToHSV)
		hue := description.H / 2
		saturation := description.S * 2.55
		value := description.V * 2.55

		hueBoundary := description.HB / 2
		saturationBoundary := description.SB * 2.55
		valueBoundary := description.VB * 2.55

		lb := gocv.NewScalar(hue-hueBoundary, saturation-saturationBoundary, value-valueBoundary, 1)
		ub := gocv.NewScalar(hue+hueBoundary, saturation+saturationBoundary, value+valueBoundary, 1)

		lowerMat := hsvMat.Region(lowerImageRect)
		lowerMask := mask.Region(lowerImageRect)
		gocv.InRangeWithScalar(lowerMat, lb, ub, &lowerMask)

		contours := gocv.FindContours(mask, gocv.RetrievalList, gocv.ChainApproxSimple)

		features := make([]Feature, 0, len(contours))
		if len(contours) != 0 {
			oldRect := gocv.BoundingRect(contours[0])
			for _, contour := range contours[1:] {
				newRect := gocv.BoundingRect(contour)
				if distance(&newRect, &oldRect) > 5 { // 10
					if Area(&oldRect) >= 100 {
						features = append(features, Feature{
							imagePos:    findMiddlePoint(&oldRect),
							imageBounds: oldRect,
						})
					}
					oldRect = newRect
				} else {
					oldRect = oldRect.Union(newRect)
				}
			}
			if Area(&oldRect) >= 100 {
				features = append(features, Feature{
					imagePos:    findMiddlePoint(&oldRect),
					imageBounds: oldRect,
				})
			}
		}
		outbound <- features

		err := lowerMat.Close()
		if err != nil {
			log.Fatal(err)
		}
		err = lowerMask.Close()
		if err != nil {
			log.Fatal(err)
		}
		managedMat.put(func(mat *gocv.Mat) {
			if len(contours) != 0 {
				gocv.FillPoly(mat, contours, white)
				for _, feature := range features {
					gocv.Rectangle(mat, feature.imageBounds, green, 1)
					gocv.ArrowedLine(mat, image.Point{mat.Cols() / 2, mat.Rows()}, feature.imagePos, red, 1)
				}
			}
		})
		managedMat.Done()
	}
}
