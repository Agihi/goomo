# goomo

## Demo

[![thumbnail](readme/goomo_demo_thumbnail_with_play.png)](https://www.youtube.com/watch?v=Xmw1vgQwXIA)

## Project Overview 
The goomo project is structured modularly.
In the main struct `goomo` many modules, such as `PostitAI`, can be added or removed dynamically via its `Activate()` and `Deactivate()` functions.
A module has inbound channels for data input and outbound channels for data output.  
The `goomo` itself is started with `goomo.Start()` in the `newserv.go/main` function, after activating the desired modules one has to call `goomo.Wait()`.
This keeps the main go routine alive, otherwise all routines would be stopped alongside the main go routine.

As to avoid circular dependencies there exists only one go-package `goomo`.
In order to keep the files organized prefixes (such as `ep_` for endpoints) are added to the filenames.

![diagram](readme/goomo_diagram.png)

### Modules
#### LommoCommunicator - lc
This module is responsible for establishing a TCP connection to the Loomo and for sending commands.
With the `RegisterHandler` method `StreamDataHandler` like `DataProcessor` can be added to receive `SensorStream` data.
The LoomoCommunicator holds the `Cmds` channel, which is for example passed to the `MovementAI` as an outbound channel.

After connecting, `lc.Connect()`, and starting, `lc.Start()`, everytime a `Command` is sent to the `Cmds` channel, it is automatically sent to the Loomo. 

#### DataProcessor - dp
This module is registered as a `StreamDataHandler` to the `DataProcessor`.

It decodes the stream data to `gocv.Mat`, wraps it in the `ManagedMat` struct for memory management and writes to the `OutboundMat` channel, which is used by the `MatMux`.
Furthermore, it writes the stream data in the `OutboundJPG` channel, used by the `JPGMux`.

#### Mat- and JPG-Multiplexer - matMux, jpgMux
These modules use the outbound channels of the `DataProcessor` as inbound and distribute the incoming mats/jpgs amongst its receivers.

Receivers can be added and removed via the `Add` and `Remove` methods.

#### PostitTracker - pt
This module "inherits" from `ColorTracker`. It has to be registered in `matMux` to receive the mats to be processed.
It outputs a two dimensional slice of `Feature`.

The colors of the post-its are specified as `HSVDescription`, in which the `HSV` field stands for the color of the post-it in the HSV color space and the `HSVB` field specifies the accepted boundaries.  
For each incoming mat and for each color the `findColorFeature` algorithm is executed and outputs `[]Feature` (all features with the same color in one frame).
It performs color matching, contour finding and merges the bounding rectangles if they are too close to each other.
Finally it puts a drawing function for the debug-screen as finish function to the `managedMat`, which is called in `managedMat.Finish()` right before the memory is freed.  

#### TrafficSignTracker - tt
Like `PostitTracker` this module has to be registered in `matMux`, but it outputs `TrafficSignFeature`.

As of now, the `TrafficSignTracker` relies on the traffic signs being printed on magenta-colored paper.
The color tracking functions precisely the same as in `PostitTracker`, but the mats need further processing before being fed into the `TrafficSignNN`.
This is done in the loop of `StartTrafficSignTracker()`.

#### MJPGStream
This module registers in `jpgMux` and serves as a http-Handler for `/stream`.  
It is created and started in `ActivateHTTPEndpoints` in `goomo`.

#### MovementAI - ai
This module takes in the outbound channels of the Postit- and TrafficSignTracker.
It consists of the submodules `PostitAI` and `TrafficSignAI` which can be toggled independently.

The `MovementAI` has an internal state, which can be changed by calling `m.setState(state)`.
The feature-processing and movement calculations are delegated to the state via `state.handlePostits(postits)` and `state.handleTrafficSign(trafficSign)`.
In addition, the state has the possibility to execute logic at the beginning and ending of its lifecycle with `state.start()` and `state.stop()`.

Currently, there are four states:

- **Idle**
- **FollowPostits**: Assumes that there are two lanes of postits, calculates a bezierpath for each of the lanes
                     and follows the middlepath between them.  
                     If a trafficsign is detected 10 times in a row, the corresponding state will be set.
                     If no postits are detected a uturn will be initiated.
- **Uturn**: Turns without moving forward until two postits of different color are detected within 150cm reach;  
             the succeeding state is FollowPostits
- **Stop**: Loomo stops for 5 seconds, then follows postits for 5 seconds;  
        the succeeding state is FollowPostits

After processing the incoming data and calculating the linear `lv` and angular `av` velocities it outputs the command into the `Cmds` channel of the `LoomoCommunicator`.

#### TrafficSignNN
This module uses tensorflow for go and loads an already trained neural net (`traffic_sign_nn/*`).
This model is built in [pyNet](https://iteragit.iteratec.de/go_loomo_go/pynet) based on [this](https://github.com/mohamedameen93/German-Traffic-Sign-Classification-Using-TensorFlow) and trained with our own [data set](https://iteragit.iteratec.de/go_loomo_go/pynet/tree/master/data_04).  
To export and import the model has to be saved with the tag "serve":
```
python: builder.add_meta_graph_and_variables(session, ["serve"])
go:     model, err := tf.LoadSavedModel(exportDir, []string{"serve"}, nil)
```
Also, the individual operations have to exported with a name:
```
python: self.predict_operation = tf.argmax(self.logits, 1, name="predict")
go:     predictOp := model.Graph.Operation("predict")
```
For traffic sign classification the method `predictWithCertainty(mat)` is used.

#### DistanceLookup
This module is a shared instance, which creates a pixel-distance mapping on init.
The math behind this mapping require the viewing angles (`verticalAlpha`, `horizontalBeta`) and the height of the Loomo (`eyeHeight`), which were estimated empirically.
It is also assumed that the camera is parallel to the floor (though it is corrected with `transform`) and that the floor is flat.  
The lookup-table only stores distances for pixel coordinates `(px, py)`, where `py > pixelHorizon`, as it would be to inaccurate for pixels further atop.  
To get the estimated distance in two dimensions for a pixel, call `Distance(x, y int) (dx, dy float64)`.  
It is also possible to estimate the pixel for a distance input with `Pixel(x, y float64) (px, py int)`, which uses binary search.  
In addition, there are many helper functions, e.g. to compute the distance between to pixel.

![distance_v](readme/distance_v.png) 
![distance_h](readme/distance_h.png)

#### MonoSLAM - slam
This module provides go bindings for the C++ Library [ORB_SLAM2](https://iteragit.iteratec.de/go_loomo_go/orb_slam).
`slam_lib/MonoSLAM.h` is a C-Wrapper for the most important parts of the library and `slam_lib/libORB_SLAM2.so` is the shared object library, which is built with the above repository.
Both of this files have to be linked/included:
```
#cgo LDFLAGS: -L ${SRCDIR}/slam_lib -l ORB_SLAM2
#cgo CFLAGS: -I ${SRCDIR}/slam_lib
#include "MonoSLAM.h
```
In addition, the following environment variable has to be set in the run configuration:
```
LD_LIBRARY_PATH=/path/to/goomo/slam_lib
```
Furthermore, all C++ dependencies have to be [installed](#dependencies).

In the `slam_lib/settings.yaml` file the camera distortion and calibration parameters can be set, feature point (ORB) extraction can be tweaked and the pangolin viewer can be configured.
The camera parameters were [estimated with opencv](https://docs.opencv.org/2.4/doc/tutorials/calib3d/camera_calibration/camera_calibration.html).

Currently, the following bindings exist:

- ` Track(img *gocv.Mat, timestamp uint64)`: with this method the video stream is fed into the SLAM algorithm
- `GetLastPose() (*gocv.Mat, error)`: which returns the last pose as a [4x4 extrinsic parameter matrix](https://en.wikipedia.org/wiki/Camera_resectioning), **not scaled** to real world distances
- `GetFeaturePoints()`: which gets all points that are visible on the map (red and black)
- `GetMatchedFeaturePoints()`: which gets all points that are currently tracked (green points on the debug screen)
- `GetState() TrackingState`: which returns the current state of the SLAM system

_Honorable mention_:  
The [VINS-Monocular System](https://github.com/HeYijia/VINS-Course) was also tested with the Loomo and performs considerably better at feature point tracking in an office environment.
However, its 3D mapping relies on gyroscope and accelerometer input, which is not implemented yet.

#### VideoMaker - vm
This module simply registers in `matMux`, captures a video of the Loomo's camera and safes it to a specified directory. 

### Endpoints

#### /stream
Endpoint for streaming the Loomo camera video stream. For usage [see](https://iteragit.iteratec.de/go_loomo_go/anglo/blob/master/src/app/mjpeg-stream/mjpeg-stream.component.html).

#### /motion
Method: PUT  
Body:
```
{
type: "linear" | "angular",
value: float
}
```
This endpoint receives motion commands and forwards them to the `LoomoCommunicator`.
#### /settings
Method: GET  
Response:
```
[
settings-key: bool,
...
]
```


Method: PUT  
Body:
```
[
settings-key_toBeToggled: bool,
...
]
```
Response:
```
[
settings-key_wasToggled: bool,
...
]
```
Currently allowed settings-keys:
- "debug-screen"
- "postit-ai"
- "trafficsign-ai"
- "slam"
- "video-capture" 

This endpoint talks directly to the `goomo` struct and calls `IsActive()`, `Activate()` and `Deactivate()` functions.

#### /video
Method: GET  
Response: BinaryData

When toggling "video-capture", a video is saved to `/video/tmp.h264`. With this endpoint the video can be downloaded.

## Installation

### Go Dependencies

#### Gorilla
```
go get -u github.com/gorilla/mux
go get -u github.com/gorilla/handlers
```

#### gocv
https://gocv.io/getting-started/linux/
```
gocv version: 0.20.0
opencv lib version: 4.1.0
```

#### tensorflow
https://www.tensorflow.org/install/lang_go

#### logger
```
go get -u go.uber.org/zap
```

#### Optional: gonum/plot
```
go get gonum.org/v1/plot/
```
<a name="dependencies"></a>
### C++ Dependencies
Related packages:
```
sudo apt-get install build-essential
sudo apt-get install git cmake
sudo apt-get install freeglut3-dev libglu-dev libglew-dev
sudo apt-get install ffmpeg libavcodec-dev libavutil-dev libavformat-dev libswscale-dev
```
Eigen3
```
sudo apt-get install libeigen3-dev
```
Boost
```
sudo apt-get install libboost-all-dev
```
OpenCV should be already installed from `gocv`.

Pangolin
```
cd MY_EXTERNAL_LIBRARIES_DIRECTORY
git clone https://github.com/stevenlovegrove/Pangolin.git
cd Pangolin
mkdir BUILD
cd BUILD
cmake ..
make -j4
sudo make install
```

Set environment variable in the run configuration:
```
LD_LIBRARY_PATH=/path/to/goomo/slam_lib
```