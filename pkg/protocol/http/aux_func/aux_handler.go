package httpAux

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "httpHandler")

const (
	RESPOK   string = "Success"
	RESPFAIL string = "Fail"
)

//Helper gunction for get response
func HttpGet(url string, cl http.Client) []byte {
	resp, err := cl.Get(url)
	if err != nil {
		log.Errorln("Get Response", err)
		return []byte(RESPFAIL)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln("Get buffer read Response", err)
		return []byte(RESPFAIL)
	}
	//Convert the body to type string
	return body
}

func HttpPost(url string, jsonData []byte, cl http.Client) []byte {

	resp, err := cl.Post(url, "application/json",
		bytes.NewBuffer(jsonData))

	if err != nil {
		log.Errorln("Post Response", err)
		return []byte(RESPFAIL)
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln("Post buffer read Response", err)
		return []byte(RESPFAIL)
	}

	return body
}

func HttpDelete(url string, jsonData []byte, cl http.Client) []byte {

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := cl.Do(req)

	if err != nil {
		log.Errorln("HttpDelete req:", err)
		return []byte(RESPFAIL)
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln("HttpDelete Read:", err)
		return []byte(RESPFAIL)
	}

	return body
}

func HttpConnect(address, url string, jsonData string) (net.Conn, error) {
	c, err := dial(address)
	//defer c.Close()

	log.Infof("Send Connect request to url: %v", url)
	client := http.Client{Transport: &http.Transport{Dial: connDialer{c}.Dial}}
	req, err := http.NewRequest(http.MethodConnect, url, bytes.NewBuffer([]byte(jsonData)))
	resp, err := client.Do(req)
	if err != nil {
		log.Errorln(err)
		return nil, nil
	}
	log.Println("Connect resp: ", resp.StatusCode)

	return c, nil
}

func dial(addr string) (net.Conn, error) {
	log.Info("Start dial to address: %v", addr)
	c, err := net.Dial("tcp", addr)

	if err != nil {
		return nil, err
	}
	log.Info("Finish dial to address: %v", addr)

	return c, err
}

type connDialer struct {
	c net.Conn
}

func (cd connDialer) Dial(network, addr string) (net.Conn, error) {
	return cd.c, nil
}
