package jsonapi

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
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
	Status int
	Body   []byte
}

// Get sends an HTTP GET request.
func (c *Client) Get(path string) (*Response, error) {
	return c.do(http.MethodGet, path, nil)
}

// Post sends an HTTP POST request.
func (c *Client) Post(path string, body []byte) (*Response, error) {
	return c.do(http.MethodPost, path, body)
}

// Delete sends an HTTP DELETE request.
func (c *Client) Delete(path string, body []byte) (*Response, error) {
	return c.do(http.MethodDelete, path, body)
}

// ServerURL returns the server URL configured for this client.
func (c *Client) ServerURL() string {
	return c.serverURL
}

func (c *Client) do(method, path string, body []byte) (*Response, error) {
	requestLogger := c.logger.WithFields(logrus.Fields{"method": method, "path": path})

	requestLogger.WithField("body-length", len(body)).Infof("Issuing request.")
	requestLogger.Debugf("Request body: %v.", body)

	req, err := http.NewRequest(method, c.serverURL+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("unable to create http request: %v", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to perform http request: %v", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			requestLogger.Warnf("Cannot close response body: %v.", err)
		}
	}()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("unable to read response body: %v", err)
	}

	requestLogger.WithField("body-length", len(body)).Infof("Received response: %d.", resp.StatusCode)
	requestLogger.Debugf("Response body: %v.", body)

	return &Response{
		Status: resp.StatusCode,
		Body:   body,
	}, nil
}

// NewClient returns a new HTTP client.
func NewClient(host string, port uint16, tlsConfig *tls.Config) *Client {
	serverURL := fmt.Sprintf("https://%s:%d", host, port)
	return &Client{
		client: &http.Client{
			Transport: &http.Transport{TLSClientConfig: tlsConfig},
			Timeout:   3 * time.Second,
		},
		serverURL: serverURL,
		logger: logrus.WithFields(logrus.Fields{
			"component":  "http-client",
			"server-url": serverURL}),
	}
}
