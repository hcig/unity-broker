package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"io"
	"net/http"
	"strconv"
	"strings"
	"viveSyncBroker/lib"
	messages "viveSyncBroker/pb"
	"viveSyncBroker/persistence"
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
		if err != nil && !strings.EqualFold(err.Error(), "no rows in result set") {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Println(writer.Write([]byte(strconv.Itoa(part))))
	case http.MethodPost: // Set current participant number
		body, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		bodyMap := make(map[string]any)
		if err = json.Unmarshal(body, &bodyMap); err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err = netmgr.Persist.SetParticipant(int(bodyMap["participant"].(float64))); err != nil {
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
		if err = netmgr.Persist.AddParticipantData(data); err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		writer.WriteHeader(http.StatusCreated)
	default:
		_, _ = writer.Write([]byte("Method not implemented"))
		writer.WriteHeader(http.StatusNotImplemented)
		return
	}
}

func TrialsHandler(writer http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet: // Get current number
		// If participant ID was sent; set it
		if request.URL.Query().Has("participant") {
			participant, err := strconv.Atoi(request.URL.Query().Get("participant"))
			if err != nil {
				fmt.Println(err)
				writer.WriteHeader(http.StatusInternalServerError)
			}
			if err = netmgr.Persist.SetParticipant(participant); err != nil {
				fmt.Println(err)
				writer.WriteHeader(http.StatusInternalServerError)
			}
		}
		trial, err := netmgr.Persist.LastTrial()
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		fmt.Println(trial)
		_, _ = writer.Write([]byte(strconv.Itoa(trial)))
	case http.MethodPost: // Set current trial number
		body, err := io.ReadAll(request.Body)
		if err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		bodyMap := make(map[string]any)
		if err = json.Unmarshal(body, &bodyMap); err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = netmgr.Persist.SetTrial(int(bodyMap["trial"].(float64)))
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
		if err = netmgr.Persist.AddTrialData(data); err != nil {
			fmt.Println(err)
			writer.WriteHeader(http.StatusInternalServerError)
		}
		writer.WriteHeader(http.StatusCreated)
	default:
		_, _ = writer.Write([]byte("Method not implemented"))
		writer.WriteHeader(http.StatusNotImplemented)
		return
	}
}

func QuestionnairesHandler(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	err = netmgr.Persist.SaveQuestionnaire(request.URL.Query().Get("kind"), body)
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
}

func (nm *NetworkMgr) GesturesOverrideHandler(writer http.ResponseWriter, request *http.Request) {
	body, err := io.ReadAll(request.Body)
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	bodyMap := make(map[string]bool)
	if err = json.Unmarshal(body, &bodyMap); err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	hasGesture := "0"
	if bodyMap["g"] {
		hasGesture = "1"
	}
	value, err := structpb.NewStruct(map[string]interface{}{
		"g": hasGesture,
		"c": 1,
	})
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	msg := &messages.Command{
		Timestamp: timestamppb.Now(),
		Command:   messages.CommandType_MsgCommand,
		Payload: &messages.Payload{
			PayloadTypes: &messages.Payload_Msg{
				Msg: &messages.Payload_MsgPayload{
					Receiver: "PoseReceiver",
					Method:   "Override",
					Value:    structpb.NewStructValue(value),
				},
			},
		},
	}
	err = netmgr.Persist.AddEntry("manual", msg)
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	nm.Pubsub.Publish(PubSubTopicBasic, msg)
	writer.WriteHeader(http.StatusOK)
}

/***
 * Endpoints delivering R fragments
 **/

func RConnectionHandler(writer http.ResponseWriter, request *http.Request) {
	connVar := lib.CoalesceString(request.URL.Query().Get("name"), persistence.RDefaultConnectionVarName)
	_, _ = writer.Write([]byte(fmt.Sprintf("library(DBI)\n%s <- %s", connVar, persistence.TSConfig.AsRConnection())))
}

func RTimeseriesQueryHandler(writer http.ResponseWriter, request *http.Request) {
	connVar := lib.CoalesceString(request.URL.Query().Get("name"), persistence.RDefaultConnectionVarName)
	queryVar := lib.CoalesceString(request.URL.Query().Get("queryName"), persistence.RDefaultDataQueryVarName)
	vars := mux.Vars(request)
	part, err := strconv.Atoi(vars["part"])
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	trial, err := strconv.Atoi(vars["trial"])
	if err != nil {
		fmt.Println(err)
		writer.WriteHeader(http.StatusInternalServerError)
	}
	_, _ = writer.Write([]byte(fmt.Sprintf(
		`%s <- dbGetQuery(%s, "%s", param = list(%d, %d))`,
		queryVar, connVar, netmgr.Persist.(*persistence.TimescaleHandler).GetTrialQuery(), part, trial,
	)))
}
