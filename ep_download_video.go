package goomo

import (
	"io/ioutil"
	"net/http"
)

type DownloadVideo struct {
}

func (s *DownloadVideo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	video, err := ioutil.ReadFile(videofilePath)
	if err != nil {
		logger.Error(err)
	}
	w.Header().Set("Content-Disposition", "attachment; filename=loomo_video.h264")
	w.Header().Set("Content-Type", "application/octet-stream")

	_, err = w.Write(video)
	if err != nil {
		logger.Error(err)
	}
}
