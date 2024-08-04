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

package jsonapi

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

// Client for issuing HTTP requests.
type Client struct {
	client    *http.Client
	serverURL string

	logger *logrus.Entry
}

// Response for a request.
type Response struct {
	Status  int
	Headers *http.Header
	Body    []byte
}

// Get sends an HTTP GET request.
func (c *Client) Get(ctx context.Context, path string) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post sends an HTTP POST request.
func (c *Client) Post(ctx context.Context, path string, body []byte) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// Put sends an HTTP PUT request.
func (c *Client) Put(ctx context.Context, path string, body []byte) (*Response, error) {
	return c.do(ctx, http.MethodPut, path, body)
}

// Delete sends an HTTP DELETE request.
func (c *Client) Delete(ctx context.Context, path string, body []byte) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, body)
}

// ServerURL returns the server URL configured for this client.
func (c *Client) ServerURL() string {
	return c.serverURL
}

func (c *Client) do(ctx context.Context, method, path string, body []byte) (*Response, error) {
	requestLogger := c.logger.WithFields(logrus.Fields{"method": method, "path": path})

	requestLogger.WithField("body-length", len(body)).Debugf("Issuing request.")
	requestLogger.Debugf("Request body: %v.", body)

	req, err := http.NewRequestWithContext(ctx, method, c.serverURL+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("unable to create http request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// check for timeout error which could be due to a failed re-used connection
		var uerr *url.Error
		if errors.As(err, &uerr) && uerr.Timeout() {
			// close old connections
			c.client.Transport.(*http.Transport).CloseIdleConnections()

			// retry request with a fresh connection
			req.Body = io.NopCloser(bytes.NewBuffer(body))
			resp, err = c.client.Do(req)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("unable to perform http request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			requestLogger.Warnf("Cannot close response body: %v.", err)
		}
	}()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %w", err)
	}

	requestLogger.WithField("body-length", len(body)).Debugf("Received response: %d.", resp.StatusCode)
	requestLogger.Debugf("Response body: %v.", body)

	return &Response{
		Status:  resp.StatusCode,
		Headers: &resp.Header,
		Body:    body,
	}, nil
}

// NewClient returns a new HTTP client.
func NewClient(host string, port uint16, tlsConfig *tls.Config) *Client {
	serverURL := "https://" + net.JoinHostPort(host, strconv.Itoa(int(port)))
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   3 * time.Second,
		},
		serverURL: serverURL,
		logger: logrus.WithFields(logrus.Fields{
			"component":  "http-client",
			"server-url": serverURL,
		}),
	}
}
