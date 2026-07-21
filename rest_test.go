package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	messages "viveSyncBroker/pb"
	"viveSyncBroker/persistence"
)

func TestHomeHandler(t *testing.T) {
	rr := httptest.NewRecorder()
	HomeHandler(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "/participants") {
		t.Fatalf("unexpected body: %s", rr.Body.String())
	}
}

func TestParticipantsHandlerGetPostPatchAndErrors(t *testing.T) {
	fake := &fakeHandler{lastParticipant: 7}
	nm := &NetworkMgr{Persist: fake}
	restore := withNetMgr(nm)
	defer restore()

	rr := httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodGet, "/participants", nil))
	if rr.Body.String() != "7" {
		t.Fatalf("GET body = %q", rr.Body.String())
	}

	fake.lastParticipantErr = assertErr("no rows in result set")
	rr = httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodGet, "/participants", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET no rows status = %d", rr.Code)
	}

	fake.lastParticipantErr = assertErr("boom")
	rr = httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodGet, "/participants", nil))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("GET error status = %d", rr.Code)
	}
	fake.lastParticipantErr = nil

	rr = httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodPost, "/participants", bytes.NewBufferString(`{"participant":3}`)))
	if rr.Code != http.StatusOK || len(fake.setParticipantCalls) != 1 || fake.setParticipantCalls[0] != 3 {
		t.Fatalf("POST participant not recorded: status=%d calls=%#v", rr.Code, fake.setParticipantCalls)
	}

	rr = httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodPost, "/participants", bytes.NewBufferString(`{`)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("POST malformed status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodPatch, "/participants", bytes.NewBufferString(`{"note":"x"}`)))
	if rr.Code != http.StatusCreated || len(fake.addParticipantData) == 0 {
		t.Fatalf("PATCH participant status=%d data=%#v", rr.Code, fake.addParticipantData)
	}

	rr = httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodDelete, "/participants", nil))
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("DELETE status = %d", rr.Code)
	}
}

func TestTrialsHandlerGetPostPatchAndErrors(t *testing.T) {
	fake := &fakeHandler{lastTrial: 9}
	nm := &NetworkMgr{Persist: fake}
	restore := withNetMgr(nm)
	defer restore()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/trials?participant=4", nil)
	TrialsHandler(rr, req)
	if len(fake.setParticipantCalls) == 0 || fake.setParticipantCalls[0] != 4 {
		t.Fatalf("participant query not applied: %#v", fake.setParticipantCalls)
	}
	if rr.Body.String() != "9" {
		t.Fatalf("GET body = %q", rr.Body.String())
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/trials?participant=bad", nil)
	TrialsHandler(rr, req)
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("invalid participant status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	TrialsHandler(rr, httptest.NewRequest(http.MethodPost, "/trials", bytes.NewBufferString(`{"trial":2}`)))
	if rr.Code != http.StatusOK || len(fake.setTrialCalls) != 1 || fake.setTrialCalls[0] != 2 {
		t.Fatalf("POST trial not recorded: status=%d calls=%#v", rr.Code, fake.setTrialCalls)
	}

	rr = httptest.NewRecorder()
	TrialsHandler(rr, httptest.NewRequest(http.MethodPatch, "/trials", bytes.NewBufferString(`{"trial":"x"`)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("PATCH malformed status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	TrialsHandler(rr, httptest.NewRequest(http.MethodDelete, "/trials", nil))
	if rr.Code != http.StatusNotImplemented {
		t.Fatalf("DELETE status = %d", rr.Code)
	}
}

func TestQuestionnairesGesturesAndRHandlers(t *testing.T) {
	fake := &fakeHandler{lastTrial: 1}
	nm := &NetworkMgr{Pubsub: NewPubsub(nil), Persist: fake}
	restore := withNetMgr(nm)
	defer restore()

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/questionnaires?kind=demo", bytes.NewBufferString("body"))
	QuestionnairesHandler(rr, req)
	if got := string(fake.questionnaires["demo"]); got != "body" {
		t.Fatalf("questionnaire body = %q", got)
	}

	client := newMemoryConn("client-a", nil)
	nm.Pubsub.Subscribe(PubSubTopicBasic, client)
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/override-gestures", bytes.NewBufferString(`{"g":true}`))
	nm.GesturesOverrideHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("gesture override status = %d", rr.Code)
	}
	value, _ := nm.Pubsub.subs[PubSubTopicBasic].Load(client.RemoteAddr().String())
	msg := <-value.(*RemoteClient).Chan
	command := msg.(*messages.Command)
	payloadStruct := command.Payload.GetMsg().GetValue().GetStructValue()
	if payloadStruct.Fields["g"].GetStringValue() != "1" {
		t.Fatalf("gesture override payload = %#v", payloadStruct)
	}

	persistence.TSConfig = &persistence.TsConnection{
		Host:     "localhost",
		Port:     5432,
		DBName:   "study",
		User:     "user",
		Password: "pass",
	}
	rr = httptest.NewRecorder()
	RConnectionHandler(rr, httptest.NewRequest(http.MethodGet, "/r/connection?name=conn", nil))
	if !strings.Contains(rr.Body.String(), "conn <-") {
		t.Fatalf("RConnectionHandler output = %q", rr.Body.String())
	}

	ts := &persistence.TimescaleHandler{}
	if err := ts.SetPrefix("study"); err != nil {
		t.Fatalf("SetPrefix: %v", err)
	}
	nm.Persist = ts
	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/r/timeseries/1/2?name=conn&queryName=q", nil)
	req = mux.SetURLVars(req, map[string]string{"part": "1", "trial": "2"})
	RTimeseriesQueryHandler(rr, req)
	if !strings.Contains(rr.Body.String(), `q <- dbGetQuery(conn,`) {
		t.Fatalf("RTimeseriesQueryHandler output = %q", rr.Body.String())
	}
}

func TestRestHandlerNegativeBranches(t *testing.T) {
	fake := &fakeHandler{addParticipantErr: assertErr("boom"), addTrialErr: assertErr("boom"), saveQuestionnaireErr: assertErr("boom")}
	nm := &NetworkMgr{Pubsub: NewPubsub(nil), Persist: fake}
	restore := withNetMgr(nm)
	defer restore()

	rr := httptest.NewRecorder()
	ParticipantsHandler(rr, httptest.NewRequest(http.MethodPatch, "/participants", bytes.NewBufferString(`{"n":1}`)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("participant patch error status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	TrialsHandler(rr, httptest.NewRequest(http.MethodPatch, "/trials", bytes.NewBufferString(`{"n":1}`)))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("trial patch error status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	QuestionnairesHandler(rr, httptest.NewRequest(http.MethodPost, "/questionnaires?kind=demo", bytes.NewBufferString("body")))
	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("questionnaire error status = %d", rr.Code)
	}
}

func assertErr(msg string) error { return errors.New(msg) }
