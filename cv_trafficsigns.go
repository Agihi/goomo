package goomo

import (
	"errors"
	"fmt"
	tf "github.com/tensorflow/tensorflow/tensorflow/go"
	"gocv.io/x/gocv"
	"gonum.org/v1/plot/vg"
	"image"
	"os"
	"path/filepath"
)

var counter = 0
var size = image.Point{32, 32}

type TrafficSignDescription struct {
	HSVDescription
	outgoing chan TrafficSign
}

type TrafficSignNN struct {
	model             *tf.SavedModel
	predictOperation  *tf.Operation
	logitsOperation   *tf.Operation
	softmaxOperation  *tf.Operation
	imagesPlaceholder tf.Output
	keepProb          tf.Output
	keepProbConv      tf.Output
}

type TrafficSign struct {
	Name     string
	Index    int64
	Position vg.Point
}

const (
	stopSign       = "stop"
	uturnSign      = "uturn"
	sharpRightSign = "sharp_right"
	unknownSign    = "unknown"
)

// loads traffic sign neural net from ./traffic_sign_nn
func NewTrafficSignNN() (*TrafficSignNN, error) {
	return newTrafficSignNN()
}

// image has to be 32 x 32 px and grayscaled with values ranging from 0 to 255
// returns a prediction for a traffic sign and its certainy
func (nn *TrafficSignNN) PredictWithCertainty(image *gocv.Mat) (TrafficSign, float32, error) {
	return nn.predictWithCertainty(image)
}

func newTrafficSignNN() (*TrafficSignNN, error) {
	//exportDir := "/home/markus/PycharmProjects/simple_traffic_sign_detection/traffic_sign_nn"
	exportDir := "traffic_sign_nn/"
	model, err := tf.LoadSavedModel(exportDir, []string{"serve"}, nil)
	if err != nil {
		return nil, err
	}

	predictOp := model.Graph.Operation("predict")
	logitsOp := model.Graph.Operation("logits")
	softmaxOp := model.Graph.Operation("softmax")
	imagesOp := model.Graph.Operation("images_ph")
	keepProbOp := model.Graph.Operation("keep_prob")
	keepProbConvOp := model.Graph.Operation("keep_prob_conv")

	if predictOp == nil {
		return nil, errors.New("predict operation not found")
	}
	if logitsOp == nil {
		return nil, errors.New("logitsOp operation not found")
	}
	if softmaxOp == nil {
		return nil, errors.New("softmaxOp operation not found")
	}
	if imagesOp == nil {
		return nil, errors.New("images_ph operation not found")
	}
	if keepProbOp == nil {
		return nil, errors.New("keepProbOp operation not found")
	}
	if keepProbConvOp == nil {
		return nil, errors.New("keepProbConvOp operation not found")
	}

	return &TrafficSignNN{
		model:             model,
		predictOperation:  predictOp,
		logitsOperation:   logitsOp,
		softmaxOperation:  softmaxOp,
		imagesPlaceholder: imagesOp.Output(0),
		keepProb:          keepProbOp.Output(0),
		keepProbConv:      keepProbConvOp.Output(0),
	}, nil
}

// input: grayscaled 32 x 32 px image
func preprocess(image *gocv.Mat) (*tf.Tensor, error) {
	n := image.Rows()
	m := image.Cols()

	if n != 32 || m != 32 {
		return nil, errors.New("Mat has to be 32 x 32 px")
	}

	arr := [1][32][32][1]float32{}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			arr[0][i][j][0] = float32(image.GetUCharAt(i, j)) / 255.0
		}
	}

	return tf.NewTensor(arr)
}

func parsePrediction(value interface{}) (int64, error) {
	switch i := value.(type) {
	case []int64:
		return i[0], nil
	default:
		return 0, fmt.Errorf("Could not parse value %v to int", value)
	}
}

func parseSoftmax(value interface{}) ([]float32, error) {
	switch i := value.(type) {
	case [][]float32:
		return i[0], nil
	default:
		return []float32{}, fmt.Errorf("Could not parse value %v to int", value)
	}
}

func NewTrafficSign(index int64) (TrafficSign, error) {
	switch index {
	case 0:
		return TrafficSign{sharpRightSign, index, vg.Point{}}, nil
	case 1:
		return TrafficSign{stopSign, index, vg.Point{}}, nil
	case 2:
		return TrafficSign{uturnSign, index, vg.Point{}}, nil
	case 3:
		return TrafficSign{unknownSign, index, vg.Point{}}, nil
	default:
		return TrafficSign{}, fmt.Errorf("Could not create Traffic sign with index %d", index)
	}
}

// images has to be 32 x 32 px and grayscaled with values between 0 and 255
func (nn *TrafficSignNN) Predict(image *gocv.Mat) (TrafficSign, error) {
	tensor, terr := preprocess(image)
	if terr != nil {
		return TrafficSign{}, terr
	}

	result, err := nn.model.Session.Run(
		map[tf.Output]*tf.Tensor{nn.imagesPlaceholder: tensor}, // feeds
		[]tf.Output{nn.predictOperation.Output(0)},             // fetches
		nil,
	)

	if err != nil {
		return TrafficSign{}, terr
	}

	// should not happen
	if len(result) == 0 {
		return TrafficSign{}, fmt.Errorf("Too few returned tensors.")
	}

	t := result[0]
	value := t.Value()
	index, perr := parsePrediction(value)

	if perr != nil {
		return TrafficSign{}, perr
	}

	return NewTrafficSign(index)
}

// images has to be 32 x 32 px and grayscaled with values between 0 and 255
func (nn *TrafficSignNN) predictWithCertainty(image *gocv.Mat) (TrafficSign, float32, error) {
	tensor, terr := preprocess(image)
	if terr != nil {
		return TrafficSign{}, 0.0, terr
	}

	one, _ := tf.NewTensor(float32(1))

	result, err := nn.model.Session.Run(
		// feeds
		map[tf.Output]*tf.Tensor{
			nn.imagesPlaceholder: tensor,
			nn.keepProb:          one,
			nn.keepProbConv:      one,
		},
		// fetches
		[]tf.Output{
			nn.predictOperation.Output(0),
			nn.softmaxOperation.Output(0),
		},
		nil,
	)

	if err != nil {
		return TrafficSign{}, 0.0, terr
	}

	// should not happen
	if len(result) < 1 {
		return TrafficSign{}, 0.0, fmt.Errorf("Too few returned tensors.")
	}

	t := result[0]
	index, perr := parsePrediction(t.Value())

	if perr != nil {
		return TrafficSign{}, 0.0, perr
	}

	s := result[1]
	probs, serr := parseSoftmax(s.Value())

	if serr != nil {
		return TrafficSign{}, 0.0, serr
	}

	certainty := probs[index]
	trafficSign, tferr := NewTrafficSign(index)

	return trafficSign, certainty, tferr
}

func (nn *TrafficSignNN) Close() {
	nn.model.Session.Close()
}

func testData() {
	root := "/home/markus/PycharmProjects/simple_traffic_sign_detection/data_01"

	nn, _ := NewTrafficSignNN()

	correct_count := 0
	total_count := 0

	folderWalk := func(path string, info os.FileInfo, err error) error {
		fmt.Println(path)
		correct_class := filepath.Base(filepath.Dir(path))
		mat := gocv.IMRead(path, gocv.IMReadGrayScale)
		//fmt.Println(mat.Cols(), mat.Rows(), mat.Type())

		ts, _, err := nn.PredictWithCertainty(&mat)
		mat.Close()

		if err != nil {
			return nil
		}

		//fmt.Println(ts.name, correct_class, ts.name == correct_class)

		if ts.Name == correct_class {
			correct_count += 1
		}

		total_count += 1

		return nil
	}

	filepath.Walk(root, folderWalk)

	fmt.Printf("accuracy: %v,  %v, %v", float64(correct_count)/float64(total_count), correct_count, total_count)

}

func writeMat(mat *gocv.Mat) {
	if mat.Cols()*mat.Rows() >= 20*20 {
		dimMat := gocv.NewMat()

		// preprocess 32x32
		gocv.Resize(*mat, &dimMat, size, 0, 0, gocv.InterpolationLinear)

		gocv.IMWrite(fmt.Sprintf("/home/markus/Pictures/data_03/stop/sign%v.jpg", counter), dimMat)
		fmt.Println(counter)
		counter++

		dimMat.Close()
	}
}
