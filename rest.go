package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func HomeHandler(writer http.ResponseWriter, request *http.Request) {

	res, err := json.Marshal(map[string]string{
		"/participants": "Get/Set current participant [GET/POST] or set participant data [PATCH]",
		"/trials": "Get/Set current trial [GET/POST] or set trial data [PATCH]",
	})
	if err != nil {
		return
	}
	fmt.Println(writer.Write(res))
}

func ParticipantsHandler(writer http.ResponseWriter, request *http.Request) {
	if  {

	}
	netmgr.Persist.LastParticipant()
}

func TrialsHandler(writer http.ResponseWriter, request *http.Request) {

}