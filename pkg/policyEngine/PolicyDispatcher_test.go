package policyEngine_test

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

	event "github.ibm.com/mbg-agent/pkg/controlplane/eventManager"
	"github.ibm.com/mbg-agent/pkg/policyEngine"
	"github.ibm.com/mbg-agent/pkg/policyEngine/policytypes"
)

var (
	selectAllSelector = metav1.LabelSelector{}
	simpleSelector    = metav1.LabelSelector{MatchLabels: policytypes.WorkloadAttrs{policyEngine.ServiceNameLabel: "svc"}}
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
	policyEngine.StartPolicyDispatcher(router, event.Allow)

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
	resp, err := client.Post(server.URL+policyEngine.AccessRoute+policyEngine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = client.Get(server.URL + policyEngine.AccessRoute)
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
	resp, err := client.Post(server.URL+policyEngine.AccessRoute+policyEngine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	resp, err = client.Post(server.URL+policyEngine.AccessRoute+policyEngine.DelRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// deleting the same policy again should result in a not-found error
	resp, err = client.Post(server.URL+policyEngine.AccessRoute+policyEngine.DelRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestAddBadPolicy(t *testing.T) {
	badPolicy := policytypes.ConnectivityPolicy{Name: "bad-policy"}
	policyBuf, err := json.Marshal(badPolicy)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+policyEngine.AccessRoute+policyEngine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)

	notEvenAPolicy := []byte{'{'} // a malformed json
	resp, err = client.Post(server.URL+policyEngine.AccessRoute+policyEngine.AddRoute, jsonEncoding, bytes.NewReader(notEvenAPolicy))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDeleteMalformedPolicy(t *testing.T) {
	notEvenAPolicy := []byte{'{'}
	resp, err := client.Post(server.URL+policyEngine.AccessRoute+policyEngine.DelRoute, jsonEncoding, bytes.NewReader(notEvenAPolicy))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestDecision(t *testing.T) {
	policy2 := policy
	policy2.To = []policytypes.WorkloadSetOrSelector{{WorkloadSelector: &selectAllSelector}}
	policyBuf, err := json.Marshal(policy2)
	require.Nil(t, err)
	resp, err := client.Post(server.URL+policyEngine.AccessRoute+policyEngine.AddRoute, jsonEncoding, bytes.NewReader(policyBuf))
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	requestAttr := event.ConnectionRequestAttr{SrcService: "svc", Direction: event.Incoming}
	connReqResp := sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Allow, connReqResp.Action)

	requestAttr = event.ConnectionRequestAttr{SrcService: "sv", Direction: event.Incoming}
	connReqResp = sendConnectionRequest(t, &requestAttr)
	require.Equal(t, event.Deny, connReqResp.Action)
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
