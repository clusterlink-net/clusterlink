package httpAux

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
)

//Helper gunction for get response
func HttpGet(url string) []byte {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalln(err)
	}
	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	//Convert the body to type string
	return body
}

func HttpPost(url string, jsonData []byte) []byte {

	resp, err := http.Post(url, "application/json",
		bytes.NewBuffer(jsonData))

	if err != nil {
		log.Fatal(err)
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return body
}

func HttpDelete(url string, jsonData []byte) []byte {

	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal(err)
	}

	//We Read the response body on the line below.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	return body
}
