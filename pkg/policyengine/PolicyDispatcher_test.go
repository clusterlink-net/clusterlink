package policyengine_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	event "github.com/clusterlink-net/clusterlink/pkg/controlplane/eventmanager"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine"
	"github.com/clusterlink-net/clusterlink/pkg/policyengine/policytypes"
)

const (
	svcName    = "svc"
	badSvcName = "sv"
)

var (
	selectAllSelector = metav1.LabelSelector{}
	simpleSelector    = metav1.LabelSelector{MatchLabels: policytypes.WorkloadAttrs{policyengine.ServiceNameLabel: svcName}}
	simpleWorkloadSet = policytypes.WorkloadSetOrSelector{WorkloadSelector: &simpleSelector}
	policy            = policytypes.ConnectivityPolicy{
		Name:       "test-policy",
		Privileged: false,
		Action:     policytypes.PolicyActionAllow,
		From:       []policytypes.WorkloadSetOrSelector{simpleWorkloadSet},
		To:         []policytypes.WorkloadSetOrSelector{simpleWorkloadSet},
	}

	server *httptest.Server
	client *http.Client
)

func TestMain(m *testing.M) {
	router := chi.NewRouter()
	policyengine.StartPolicyDispatcher(router)

	server = httptest.NewServer(router)
	client = server.Client()

	exitVal := m.Run()

	server.Close()
	os.Exit(exitVal)
}

const (
	jsonEncoding = "application/json"
)

func TestAddAndGetConnectivityPolicy(t *testing.T) {
	policyBuf, err := json.Marshal(policy)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+policyengine.AccessRoute+policyengine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = client.Get(server.URL + policyengine.AccessRoute)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	expectedBody := fmt.Sprintf("[%s]\n", policyBuf)
	if !bytes.Equal(body, []byte(expectedBody)) {
		t.Fatalf("response should be ok, was: %q", string(body))
	}
}

func TestAddAndDeleteConnectivityPolicy(t *testing.T) {
	policyBuf, err := json.Marshal(policy)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+policyengine.AccessRoute+policyengine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = client.Post(server.URL+policyengine.AccessRoute+policyengine.DelRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// deleting the same policy again should result in a not-found error
	resp, err = client.Post(server.URL+policyengine.AccessRoute+policyengine.DelRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAddBadPolicy(t *testing.T) {
	badPolicy := policytypes.ConnectivityPolicy{Name: "bad-policy"}
	policyBuf, err := json.Marshal(badPolicy)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+policyengine.AccessRoute+policyengine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	notEvenAPolicy := []byte{'{'} // a malformed json
	resp, err = client.Post(server.URL+policyengine.AccessRoute+policyengine.AddRoute, jsonEncoding, bytes.NewReader(notEvenAPolicy))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteMalformedPolicy(t *testing.T) {
	notEvenAPolicy := []byte{'{'}
	resp, err := client.Post(server.URL+policyengine.AccessRoute+policyengine.DelRoute, jsonEncoding, bytes.NewReader(notEvenAPolicy))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIncomingConnectionRequests(t *testing.T) {
	policy2 := policy
	policy2.To = []policytypes.WorkloadSetOrSelector{{WorkloadSelector: &selectAllSelector}}
	addPolicy(t, policy2)

	requestAttr := event.ConnectionRequestAttr{SrcService: svcName, Direction: event.Incoming}
	connReqResp := sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Allow, connReqResp.Action)

	requestAttr = event.ConnectionRequestAttr{SrcService: badSvcName, Direction: event.Incoming}
	connReqResp = sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Deny, connReqResp.Action)
}

func TestOutgoingConnectionRequests(t *testing.T) {
	const (
		mbg1 = "mbg1"
		mbg2 = "mbg2"
	)

	simpleSelector2 := metav1.LabelSelector{MatchLabels: policytypes.WorkloadAttrs{
		policyengine.ServiceNameLabel: svcName,
		policyengine.MbgNameLabel:     mbg2}}
	simpleWorkloadSet2 := policytypes.WorkloadSetOrSelector{WorkloadSelector: &simpleSelector2}
	policy2 := policy
	policy2.To = []policytypes.WorkloadSetOrSelector{simpleWorkloadSet2}
	addPolicy(t, policy2)
	addRemoteSvc(t, svcName, mbg1)
	addRemoteSvc(t, svcName, mbg2)

	// Should choose between mbg1 and mbg2, but only mbg2 is allowed by the single access policy
	requestAttr := event.ConnectionRequestAttr{SrcService: svcName, DstService: svcName, Direction: event.Outgoing}
	connReqResp := sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Allow, connReqResp.Action)
	require.Equal(t, mbg2, connReqResp.TargetMbg)

	// Src service does not match the spec of the single access policy
	requestAttr = event.ConnectionRequestAttr{SrcService: badSvcName, DstService: svcName, Direction: event.Outgoing}
	connReqResp = sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Deny, connReqResp.Action)

	// Dst service does not match the spec of the single access policy
	requestAttr = event.ConnectionRequestAttr{SrcService: svcName, DstService: badSvcName, Direction: event.Outgoing}
	connReqResp = sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Deny, connReqResp.Action)

	// mbg2 is removed as a remote for the requested service, so now the single allow policy does not allow the remaining mbgs
	removeRemoteSvc(t, svcName, mbg2)
	requestAttr = event.ConnectionRequestAttr{SrcService: svcName, DstService: svcName, Direction: event.Outgoing}
	connReqResp = sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Deny, connReqResp.Action)
}

func addRemoteSvc(t *testing.T, svc, mbg string) {
	remoteServiceAttr := event.NewRemoteServiceAttr{Service: svc, Mbg: mbg}
	jsonReq, err := json.Marshal(remoteServiceAttr)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+"/"+event.NewRemoteService, jsonEncoding, bytes.NewReader(jsonReq))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func removeRemoteSvc(t *testing.T, svc, mbg string) {
	remoteServiceAttr := event.RemoveRemoteServiceAttr{Service: svc, Mbg: mbg}
	jsonReq, err := json.Marshal(remoteServiceAttr)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+"/"+event.RemoveRemoteService, jsonEncoding, bytes.NewReader(jsonReq))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func addPolicy(t *testing.T, policy policytypes.ConnectivityPolicy) {
	policyBuf, err := json.Marshal(policy)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+policyengine.AccessRoute+policyengine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func sendConnectionRequest(t *testing.T, req *event.ConnectionRequestAttr) event.ConnectionRequestResp {
	requestBuf, err := json.Marshal(req)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+"/"+event.NewConnectionRequest, jsonEncoding, bytes.NewReader(requestBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	var connReqResp event.ConnectionRequestResp
	err = json.Unmarshal(body, &connReqResp)
	require.Nil(t, err)
	return connReqResp
}
