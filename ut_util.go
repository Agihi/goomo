package goomo

import (
	"fmt"
	"gocv.io/x/gocv"
	"runtime"
	"strings"
)

func flattenData(dataMap map[uint32][]byte) []byte {
	totalBytes := (len(dataMap)-1)*(maxPacketSize-headerSize) + len(dataMap[uint32(len(dataMap))-1])
	fullData := make([]byte, totalBytes)
	for i := uint32(0); i < uint32(len(dataMap)); i++ {
		copy(fullData[i*(maxPacketSize-headerSize):], dataMap[i])
	}
	return fullData
}

func signumF32(val float32) int {
	switch {
	case val < 0:
		return -1
	case val > 0:
		return 1
	default:
		return 0
	}
}

func signumF64(val float64) int {
	switch {
	case val < 0:
		return -1
	case val > 0:
		return 1
	default:
		return 0
	}
}

func signumInt(val int) int {
	switch {
	case val < 0:
		return -1
	case val > 0:
		return 1
	default:
		return 0
	}
}

func getFrame(skipFrames int) runtime.Frame {
	// We need the frame at index skipFrames+2, since we never want runtime.Callers and getFrame
	targetFrameIndex := skipFrames + 2

	// Set size to targetFrameIndex+2 to ensure we have room for one more caller than we need
	programCounters := make([]uintptr, targetFrameIndex+2)
	n := runtime.Callers(0, programCounters)

	frame := runtime.Frame{Function: "unknown"}
	if n > 0 {
		frames := runtime.CallersFrames(programCounters[:n])
		for more, frameIndex := true, 0; more && frameIndex <= targetFrameIndex; frameIndex++ {
			var frameCandidate runtime.Frame
			frameCandidate, more = frames.Next()
			if frameIndex == targetFrameIndex {
				frame = frameCandidate
			}
		}
	}

	return frame
}

// MyCaller returns the caller of the function that called it :)
func MyCaller() string {
	// Skip GetCallerFunctionName and the function to get the caller of
	return getFrame(2).Function
}

func Print32FMat(mat *gocv.Mat) {
	var strbuilder strings.Builder
	header := fmt.Sprintf("%v x %v mat\n", mat.Rows(), mat.Cols())
	strbuilder.WriteString(header)

	for row := 0; row < mat.Rows(); row++ {
		var rowstrbuilder strings.Builder
		for col := 0; col < mat.Cols(); col++ {
			v := mat.GetFloatAt(row, col)
			if v < 0 {
				rowstrbuilder.WriteString(fmt.Sprintf("%.5f\t", v))
			} else {
				rowstrbuilder.WriteString(fmt.Sprintf(" %.5f\t", v))
			}

		}
		rowstrbuilder.WriteString("\n")
		strbuilder.WriteString(rowstrbuilder.String())
	}

	fmt.Println(strbuilder.String())
}
