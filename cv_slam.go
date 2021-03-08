package goomo

// IMPORTANT:
// make sure LD_LIBRARY_PATH is set to ~/GolandProjects/goomo/slam_lib

/*
#cgo LDFLAGS: -L ${SRCDIR}/slam_lib -l ORB_SLAM2
#cgo CFLAGS: -I ${SRCDIR}/slam_lib
#include <stdlib.h>
#include "MonoSLAM.h"
*/
import "C"
import (
	"fmt"
	"gocv.io/x/gocv"
	"math"
	"sort"
	"unsafe"
)

type MonoSLAM struct {
	Inbound                      chan *ManagedMat
	OutboundFeaturePoints        chan []FeaturePoint
	OutboundMatchedFeaturePoints chan []FeaturePoint
	OutboundPose                 chan *gocv.Mat

	numberOfPoses int
	currentState  TrackingState

	c *C.MonoSLAM
}

type FeaturePoint struct {
	c *C.FeaturePoint
}

func NewMonoSLAM(orbvocPath string, settingsPath string, showViewer, semiDense bool) *MonoSLAM {
	c := C.NewMonoSLAM(C.CString(orbvocPath), C.CString(settingsPath), C.bool(showViewer), C.bool(semiDense))
	return &MonoSLAM{
		c:            c,
		currentState: SystemNotReady,
	}
}

func (m *MonoSLAM) StartSlam(Inbound chan *ManagedMat) {
	logger.Debug("Slam started.")
	m.Inbound = Inbound

	for managedMat := range m.Inbound {
		m.Track(managedMat.mat, managedMat.timestamp)

		//if m.PoseDidChange() {
		//	pose, err := m.GetLastPose()
		//	if err == nil {
		//		x, y, z := PoseToPosition(pose)
		//		fmt.Println(x, y, z)
		//	}
		//}

		state := m.GetState()
		if state != m.currentState {
			m.currentState = state
			//fmt.Println("state", state)
		}

		managedMat.Done()
	}

	m.Shutdown()
	m.Close()

	logger.Debug("Slam stopped.")
}

func (m *MonoSLAM) Close() {
	C.FreeMonoSLAM(m.c)
}

func (m *MonoSLAM) Shutdown() {
	C.Shutdown(m.c)
}

func (m *MonoSLAM) GetFeaturePoints() {
	n := m.GetNumberOfFeaturePoints()

	if n == 0 {
		return
	}

	fps := make([]FeaturePoint, n)
	for i := 0; i < n; i++ {
		fps[i] = FeaturePoint{C.GetFeaturePointAt(m.c, C.int(i))}
	}

	if m.OutboundFeaturePoints != nil {
		m.OutboundFeaturePoints <- fps
	}
}

func (m *MonoSLAM) GetNumberOfFeaturePoints() int {
	return int(C.GetNumberOfFeaturePoints(m.c))
}

func (m *MonoSLAM) GetMatchedFeaturePoints() []FeaturePoint {
	n := m.GetNumberOfMatchedFeaturePoints()

	if n == 0 {
		return nil
	}

	fps := make([]FeaturePoint, n)

	for i := 0; i < n; i++ {
		var fp *C.FeaturePoint = C.GetMatchedFeaturePointAt(m.c, C.int(i))
		fps[i] = FeaturePoint{c: fp}
	}

	//if m.OutboundMatchedFeaturePoints != nil {
	//	m.OutboundMatchedFeaturePoints <- fps
	//}

	return fps
}

func (m *MonoSLAM) GetNearestMatchedFeaturePoints(n int) {
	fps := m.GetMatchedFeaturePoints()

	pose, err := m.GetLastPose()
	if err != nil {
		return
	}

	x, y, z := PoseToPosition(pose)

	dist := func(fp FeaturePoint) float32 {
		fx, fy, fz, err := fp.GetPosition()
		if err != nil {
			return 1000000.0
		}

		return (fx-x)*(fx-x) + (fy-y)*(fy-y) + (fz-z)*(fz-z)
	}

	sort.Slice(fps, func(i, j int) bool {
		return dist(fps[i]) < dist(fps[j])
	})

	fmt.Println("Nearest:")
	for i := 0; i < int(math.Min(float64(n), float64(len(fps)))); i++ {
		fmt.Println(dist(fps[i]))
	}
}

func (m *MonoSLAM) GetNumberOfMatchedFeaturePoints() int {
	return int(C.GetNumberOfMatchedFeaturePoints(m.c))
}

func (m *MonoSLAM) GetNumberOfPoses() int {
	return int(C.GetNumberOfPoses(m.c))
}

// poses are unordered
func (m *MonoSLAM) GetPoses() []*gocv.Mat {
	n := m.GetNumberOfPoses()

	mats := make([]*gocv.Mat, n)

	if n == 0 {
		return mats
	}

	for i := 0; i < n; i++ {
		var mat C.Mat = C.GetPoseAt(m.c, C.int(i))
		mats[i] = (*gocv.Mat)(unsafe.Pointer(&mat))
	}

	// TODO: Close mats somewhere ?!
	return mats
}

func (m *MonoSLAM) PoseDidChange() bool {
	return bool(C.PoseDidChange(m.c))
}

func (m *MonoSLAM) GetLastPose() (*gocv.Mat, error) {
	var x C.Mat = C.GetLastPose(m.c)

	if x == nil {
		return nil, fmt.Errorf("C.GetLastPose returned NULL")
	}

	z := (*gocv.Mat)(unsafe.Pointer(&x))

	return z, nil
}

func TestPoseToPosition() {
	// https://math.stackexchange.com/questions/82602/how-to-find-camera-position-and-rotation-from-a-4x4-matrix
	pose := gocv.NewMatWithSize(4, 4, gocv.MatTypeCV32F)

	pose.SetFloatAt(0, 0, 0.211)
	pose.SetFloatAt(1, 0, 0.662)
	pose.SetFloatAt(2, 0, 0.718)
	pose.SetFloatAt(3, 0, 0)

	pose.SetFloatAt(0, 1, -0.306)
	pose.SetFloatAt(1, 1, 0.742)
	pose.SetFloatAt(2, 1, -0.595)
	pose.SetFloatAt(3, 1, 0)

	pose.SetFloatAt(0, 2, -0.928)
	pose.SetFloatAt(1, 2, -0.0947)
	pose.SetFloatAt(2, 2, 0.360)
	pose.SetFloatAt(3, 2, 0)

	pose.SetFloatAt(0, 3, 0.789)
	pose.SetFloatAt(1, 3, 0.147)
	pose.SetFloatAt(2, 3, 3.26)
	pose.SetFloatAt(3, 3, 1)

	Print32FMat(&pose)

	x, y, z := PoseToPosition(&pose)

	fmt.Println(x, y, z)
}

func PoseToPosition(pose *gocv.Mat) (x, y, z float32) {

	if pose == nil {
		return 0, 0, 0
	}

	T0 := pose.GetFloatAt(0, 3)
	T1 := pose.GetFloatAt(1, 3)
	T2 := pose.GetFloatAt(2, 3)

	R00 := pose.GetFloatAt(0, 0)
	R10 := pose.GetFloatAt(1, 0)
	R20 := pose.GetFloatAt(2, 0)

	R01 := pose.GetFloatAt(0, 1)
	R11 := pose.GetFloatAt(1, 1)
	R21 := pose.GetFloatAt(2, 1)

	R02 := pose.GetFloatAt(0, 2)
	R12 := pose.GetFloatAt(1, 2)
	R22 := pose.GetFloatAt(2, 2)

	// https://en.wikipedia.org/wiki/Camera_resectioning -> extrinsic parameters
	x = -(R00*T0 + R10*T1 + R20*T2)
	y = -(R01*T0 + R11*T1 + R21*T2)
	z = -(R02*T0 + R12*T1 + R22*T2)

	//fmt.Println(R00, R01, R02, T0)
	//fmt.Println(R10, R11, R12, T1)
	//fmt.Println(R20, R21, R22, T2)

	// TODO: Figure out units and accuracy
	return x, y, z
}

func (m *MonoSLAM) Track(img *gocv.Mat, timestamp uint64) {
	C.Track(m.c, C.Mat(img.Ptr()), C.double(timestamp))
}

type TrackingState int

const (
	SystemNotReady TrackingState = iota - 1
	NoImagesYet
	NotInitialized
	Ok
	Lost
)

func (m *MonoSLAM) GetState() TrackingState {
	i := int(C.GetState(m.c))
	state := TrackingState(i)
	return state
}

func (fp *FeaturePoint) GetPosition() (x, y, z float32, err error) {
	cmat := C.GetPosition(fp.c)

	if cmat == nil {
		return 0, 0, 0, fmt.Errorf("C.GetPosition returned NULL")
	}

	mat := (*gocv.Mat)(unsafe.Pointer(&cmat))
	x = mat.GetFloatAt(0, 0)
	y = mat.GetFloatAt(1, 0)
	z = mat.GetFloatAt(2, 0)

	mat.Close()

	return x, y, z, nil
}
