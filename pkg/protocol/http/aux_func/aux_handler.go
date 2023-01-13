package httpAux

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/sirupsen/logrus"
)

var log = logrus.WithField("component", "httpHandler")

//Helper gunction for get response
func HttpGet(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
		return nil
	}
	//Convert the body to type string
	return body
}

func HttpPost(url string, jsonData []byte) []byte {

	resp, err := http.Post(url, "application/json",
		bytes.NewBuffer(jsonData))

	if err != nil {
		log.Errorln(err)
		return nil
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
		return nil
	}

	return body
}

func HttpDelete(url string, jsonData []byte) []byte {

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Errorln(err)
		return nil
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorln(err)
		return nil
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
