// Copyright (c) The ClusterLink Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"

	"github.com/clusterlink-net/clusterlink/pkg/controlplane/api"
)

const (
	bearerSchemaPrefix = "Bearer "
)

type server struct {
	manager *Manager
	logger  *logrus.Entry
}

// Check a dataplane connection.
func (s *server) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	if req == nil ||
		req.Attributes == nil ||
		req.Attributes.Source == nil ||
		req.Attributes.Request == nil ||
		req.Attributes.Request.Http == nil {
		s.logger.Errorf("Invalid check request: %+v", req)
		return nil, fmt.Errorf("invalid check request: %v", req)
	}

	var resp *authv3.CheckResponse
	if req.Attributes.Source.Address != nil &&
		req.Attributes.Source.Address.GetEnvoyInternalAddress() != nil {
		resp = s.checkEgress(ctx, req)
	} else {
		resp = s.checkIngress(ctx, req)
	}

	s.logger.WithFields(logrus.Fields{
		"request":  req,
		"response": resp,
	}).Debugf("Check.")

	return resp, nil
}

// check an egress dataplane connection.
func (s *server) checkEgress(ctx context.Context, req *authv3.CheckRequest) *authv3.CheckResponse {
	httpReq := req.Attributes.Request.Http
	headers := httpReq.Headers

	expectedHeaders := []string{api.ClientIPHeader, api.ImportNameHeader, api.ImportNamespaceHeader}
	for _, header := range expectedHeaders {
		if _, ok := headers[header]; !ok {
			errorString := fmt.Sprintf("Missing '%s' header.", header)
			return buildDeniedResponse(code.Code_INVALID_ARGUMENT, typev3.StatusCode_BadRequest, errorString)
		}
	}

	resp, err := s.manager.authorizeEgress(ctx, &egressAuthorizationRequest{
		ImportName: types.NamespacedName{
			Namespace: headers[api.ImportNamespaceHeader],
			Name:      headers[api.ImportNameHeader],
		},
		IP: headers[api.ClientIPHeader],
	})
	if err != nil {
		return buildDeniedResponse(code.Code_INTERNAL, typev3.StatusCode_InternalServerError, err.Error())
	}

	if !resp.Allowed {
		errorString := fmt.Sprintf(
			"Access denied for '%s/%s'.", headers[api.ImportNamespaceHeader], headers[api.ImportNameHeader])
		return buildDeniedResponse(code.Code_PERMISSION_DENIED, typev3.StatusCode_Forbidden, errorString)
	}

	return buildAllowedResponse(&authv3.OkHttpResponse{
		Headers: []*corev3.HeaderValueOption{
			{
				Header: &corev3.HeaderValue{
					Key:   api.TargetClusterHeader,
					Value: resp.RemotePeerCluster,
				},
			},
			{
				Header: &corev3.HeaderValue{
					Key:   api.AuthorizationHeader,
					Value: bearerSchemaPrefix + resp.AccessToken,
				},
			},
		},
	})
}

// check an ingress dataplane connection.
func (s *server) checkIngress(ctx context.Context, req *authv3.CheckRequest) *authv3.CheckResponse {
	httpReq := req.Attributes.Request.Http
	switch {
	case httpReq.Method == http.MethodGet && httpReq.Path == api.HeartbeatPath:
		// heartbeat request always simply allowed
		labels := s.manager.peerLabels
		respHeader := []*corev3.HeaderValueOption{}
		for key, val := range labels {
			hvo := &corev3.HeaderValueOption{Header: &corev3.HeaderValue{Key: key, Value: val}, AppendAction: corev3.HeaderValueOption_APPEND_IF_EXISTS_OR_ADD}
			respHeader = append(respHeader, hvo)
		}
		return buildAllowedResponse(&authv3.OkHttpResponse{ResponseHeadersToAdd: respHeader})
	case httpReq.Method == http.MethodPost && httpReq.Path == api.RemotePeerAuthorizationPath:
		return s.checkAuthorizationRequest(ctx, httpReq)
	case httpReq.Method == http.MethodConnect:
		return s.checkServiceAccessRequest(httpReq)
	}

	errorString := fmt.Sprintf("No handler defined for %s %s.", httpReq.Method, httpReq.Path)
	return buildDeniedResponse(code.Code_INVALID_ARGUMENT, typev3.StatusCode_BadRequest, errorString)
}

// check an ingress connection for accessing an exported service.
func (s *server) checkServiceAccessRequest(req *authv3.AttributeContext_HttpRequest) *authv3.CheckResponse {
	authorization, ok := req.Headers[api.AuthorizationHeader]
	if !ok {
		errorString := fmt.Sprintf("Missing '%s' header.", api.AuthorizationHeader)
		return buildDeniedResponse(code.Code_INVALID_ARGUMENT, typev3.StatusCode_BadRequest, errorString)
	}

	if !strings.HasPrefix(authorization, bearerSchemaPrefix) {
		errorString := "Authorization header is not using the bearer scheme."
		return buildDeniedResponse(code.Code_INVALID_ARGUMENT, typev3.StatusCode_BadRequest, errorString)
	}
	token := strings.TrimPrefix(authorization, bearerSchemaPrefix)

	targetCluster, err := s.manager.parseAuthorizationHeader(token)
	if err != nil {
		return buildDeniedResponse(code.Code_PERMISSION_DENIED, typev3.StatusCode_Forbidden, err.Error())
	}

	return buildAllowedResponse(&authv3.OkHttpResponse{
		Headers: []*corev3.HeaderValueOption{
			{
				Header: &corev3.HeaderValue{
					Key:   api.TargetClusterHeader,
					Value: targetCluster,
				},
			},
		},
	})
}

// check an ingress connection for authorizing access to an exported service.
func (s *server) checkAuthorizationRequest(
	ctx context.Context,
	req *authv3.AttributeContext_HttpRequest,
) *authv3.CheckResponse {
	var authzReq api.AuthorizationRequest
	if err := json.NewDecoder(strings.NewReader(req.Body)).Decode(&authzReq); err != nil {
		s.logger.Errorf("Cannot decode authorization request: %v.", err)
		return buildDeniedResponse(code.Code_INVALID_ARGUMENT, typev3.StatusCode_BadRequest, err.Error())
	}

	resp, err := s.manager.authorizeIngress(
		ctx,
		&ingressAuthorizationRequest{
			ServiceName: types.NamespacedName{
				Namespace: authzReq.ServiceNamespace,
				Name:      authzReq.ServiceName,
			},
			SrcAttributes: authzReq.SrcAttributes,
		})
	switch {
	case err != nil:
		return buildDeniedResponse(code.Code_INTERNAL, typev3.StatusCode_InternalServerError, err.Error())
	case !resp.ServiceExists:
		errorString := fmt.Sprintf(
			"Exported service '%s/%s' not found.",
			authzReq.ServiceNamespace, authzReq.ServiceName)
		return buildDeniedResponse(code.Code_NOT_FOUND, typev3.StatusCode_NotFound, errorString)
	case !resp.Allowed:
		errorString := fmt.Sprintf(
			"Permission denied for '%s/%s'.",
			authzReq.ServiceNamespace, authzReq.ServiceName)
		return buildDeniedResponse(code.Code_PERMISSION_DENIED, typev3.StatusCode_Forbidden, errorString)
	}

	return buildAllowedResponse(&authv3.OkHttpResponse{
		ResponseHeadersToAdd: []*corev3.HeaderValueOption{
			{
				Header: &corev3.HeaderValue{
					Key:   api.AccessTokenHeader,
					Value: resp.AccessToken,
				},
			},
		},
	})
}

// RegisterService registers an ext_authz service backed by Manager to the given gRPC server.
func RegisterService(manager *Manager, grpcServer *grpc.Server) {
	srv := newServer(manager)
	authv3.RegisterAuthorizationServer(grpcServer, srv)
}

func buildAllowedResponse(resp *authv3.OkHttpResponse) *authv3.CheckResponse {
	return &authv3.CheckResponse{
		Status: &status.Status{
			Code: int32(code.Code_OK),
		},
		HttpResponse: &authv3.CheckResponse_OkResponse{
			OkResponse: resp,
		},
	}
}

func buildDeniedResponse(rpcCode code.Code, httpCode typev3.StatusCode, message string) *authv3.CheckResponse {
	return &authv3.CheckResponse{
		Status: &status.Status{
			Code:    int32(rpcCode),
			Message: message,
		},
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typev3.HttpStatus{
					Code: httpCode,
				},
				Body: message,
			},
		},
	}
}

func newServer(manager *Manager) *server {
	return &server{
		manager: manager,
		logger:  logrus.WithField("component", "controlplane.authz.server"),
	}
}
