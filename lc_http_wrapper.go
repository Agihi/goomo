package goomo

import "net/http"

type HTTPLoomoCommunicator struct {
	LoomoCommunicator
}

/*
Command JSON Structure:
{

}
*/

func (hlc *HTTPLoomoCommunicator) executeCommandViaHTTP(w http.ResponseWriter, r *http.Request) {

}
