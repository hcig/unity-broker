package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func HomeHandler(writer http.ResponseWriter, request *http.Request) {

	res, err := json.Marshal(map[string]string{
		"/participants": "Get/Set current participant [GET/POST] or set participant data [PATCH]",
		"/trials":       "Get/Set current trial [GET/POST] or set trial data [PATCH]",
	})
	if err != nil {
		return
	}
	fmt.Println(writer.Write(res))
}

func ParticipantsHandler(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet: // Get current number
		part, err := netmgr.Persist.LastParticipant()
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println(writer.Write([]byte(strconv.Itoa(part))))
	case http.MethodPost: // Set current participant number
		body, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		part, err := strconv.Atoi(string(body))
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		err = netmgr.Persist.SetParticipant(part)
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		writer.WriteHeader(http.StatusOK)
	case http.MethodPatch: // Add participant data set
		data := make(map[string]any)
		body, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		if err = json.Unmarshal(body, &data); err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}

	default:
		_, _ = writer.Write([]byte("Method not implemented"))
		writer.WriteHeader(http.StatusNotImplemented)
		return
	}
}

func TrialsHandler(writer http.ResponseWriter, request *http.Request) {

}
