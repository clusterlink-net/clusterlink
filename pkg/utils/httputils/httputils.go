package httputils

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "httpHandler")

const (
	RESPOK   string = "Success"
	RESPFAIL string = "Fail"
)

// HttpGet is a convenience function to issue a GET request
func HttpGet(url string, cl http.Client) ([]byte, error) {
	resp, err := cl.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return []byte(RESPFAIL), err
	}
	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte(RESPFAIL), err
	}
	//Convert the body to type string
	return body, nil
}

// HttpPOST is a convenience function to issue a POST request
func HttpPost(url string, jsonData []byte, cl http.Client) ([]byte, error) {
	resp, err := cl.Post(url, "application/json",
		bytes.NewBuffer(jsonData))

	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return []byte(RESPFAIL), err
	}

	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte(RESPFAIL), err
	}

	return body, nil
}

// HttpDelete is a convenience function to issue a DELETE request
func HttpDelete(url string, jsonData []byte, cl http.Client) ([]byte, error) {
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return []byte(RESPFAIL), err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := cl.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return []byte(RESPFAIL), err
	}

	//We Read the response body on the line below.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte(RESPFAIL), err
	}

	return body, nil
}

// HttpConnect is a convenience function to issue a CONNECT request
func HttpConnect(address, url string, jsonData string) (net.Conn, error) {
	c, err := dial(address)
	if err != nil {
		return nil, err
	}

	log.Infof("Send Connect request to url: %v", url)
	client := http.Client{Transport: &http.Transport{Dial: connDialer{c}.Dial}}
	req, err := http.NewRequest(http.MethodConnect, url, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Errorln(err)
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("connect response code: %v", resp.StatusCode)
	} else {
		return c, nil
	}
}

func dial(addr string) (net.Conn, error) {
	log.Infof("Start dial to address: %v\n", addr)
	c, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}
	log.Infof("Finish dial to address: %v\n", addr)

	return c, err
}

type connDialer struct {
	c net.Conn
}

// Dial fakes a connect to an existing connection
func (cd connDialer) Dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}
