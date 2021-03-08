//
// Created by markus on 09.09.19.
// C Wrapper for System.h
//

#ifndef ORB_SLAM2_MONOSLAM_H
#define ORB_SLAM2_MONOSLAM_H

#include <stdbool.h>

#ifdef __cplusplus
#include <opencv2/opencv.hpp>
typedef cv::Mat* Mat;
#else
typedef void* Mat;
#endif

#ifdef __cplusplus
extern "C" {
#endif

// Wrapper for MapPoint
typedef void* FeaturePoint;
Mat GetPosition(FeaturePoint* fp);
double GetTimestamp(FeaturePoint* fp);

// Wrapper for Keyframe
typedef void* KeyframeC;

// Wrapper for System
typedef void* MonoSLAM;

MonoSLAM* NewMonoSLAM(char* strVocFile, char* strSettingsFile, bool bUseViewer, bool semiDense);
void FreeMonoSLAM(MonoSLAM*);
void Shutdown(MonoSLAM*);

void Track(MonoSLAM* m, Mat img, double timestamp);

int GetState(MonoSLAM* m);

//FeaturePoint** GetFeaturePoints(MonoSLAM* m);
FeaturePoint* GetFeaturePointAt(MonoSLAM* m, int i);
int GetNumberOfFeaturePoints(MonoSLAM* m);
//FeaturePoint** GetMatchedFeaturePoints(MonoSLAM* m);
FeaturePoint* GetMatchedFeaturePointAt(MonoSLAM* m, int i);
int GetNumberOfMatchedFeaturePoints(MonoSLAM* m);

//Mat* GetPoses(MonoSLAM* m);
Mat GetPoseAt(MonoSLAM* m, int i);
Mat GetLastPose(MonoSLAM* m);
int GetNumberOfPoses(MonoSLAM* m);
bool PoseDidChange(MonoSLAM * m);

FeaturePoint* GetFirstFeaturePoint(MonoSLAM* m);


#ifdef __cplusplus
};
#endif

#endif //ORB_SLAM2_MONOSLAM_H
