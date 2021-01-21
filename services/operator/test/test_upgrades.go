package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
)

func postFile(filename string, targetURL string) (string, error) {
	// Needed to skip the tls certificate
	//http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("upgradeFiles", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return "", err
	}

	// open file handle
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return "", err
	}
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return "", err
	}

	bodyWriter.CreateFormField("upgradeType")
	bodyWriter.WriteField("upgradeType", "major")
	bodyWriter.CreateFormField("changelog")
	bodyWriter.WriteField("changelog", "Write changelog data")

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	resp, err := http.Post(targetURL, contentType, bodyBuf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode == 200 {
		fmt.Printf("Uploaded release success, %s\n", respBody)
	} else {
		return "", fmt.Errorf("Response status is %s", resp.Status)
	}
	return string(respBody), nil
}

func request(ty string, URL string, login string, params io.Reader) (string, error) {
	req, err := http.NewRequest(ty, URL, params)
	req.Header.Set("Content-Type", "application/json")
	if login != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", login))
	}

	res, _ := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("Request Error : %s\n", err)
		return "", err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("Request Error : %s\n", err)
		return "", err
	}
	fmt.Println(res)
	fmt.Println(string(body))
	return string(body), nil
}

func main() {

	cloudURL := flag.String("cloud_url", "https://anuraag.ntnxsherlock.com", "<cloud url>")
	edgeid1 := flag.String("edge_id1", "afe97f6c-4547-4b82-bd19-50e8ed09e956", "<edge id1>")
	//edgeid2 := flag.String("edge_id2", "", "<edge id2>")
	filename := "sherlock_edge_deployer.tgz"
	flag.Parse()

	correctPrefix := strings.HasPrefix(*cloudURL, "https://")
	if correctPrefix == false {
		fmt.Printf("Check cloud url should be https://\n")
		return
	}
	fmt.Printf("Using operator %s to upload release\n", *cloudURL)
	path := "/v1/releases"

	operatorURL := strings.Replace(*cloudURL, ".ntnxsherlock.com", ".operator.ntnxsherlock.com", 1)

	operatorEndPoint := operatorURL + path

	// Upload file to operator
	release, err := postFile(filename, operatorEndPoint)
	if err != nil {
		log.Printf("Error posting upload to operator %s\n", err)
		return
	}
	//strip quotes and new line
	release = strings.Replace(release, "\"", "", -1)
	release = strings.Trim(release, "\n")
	fmt.Printf("---------------------------------------------------------\n\n")
	time.Sleep(1 * time.Second)
	fmt.Println(release)

	fmt.Printf("List possible releases from cloud url\n")
	loginURL := *cloudURL + "/v1/login"

	// I will log in using test tenant and list edges...
	payload := strings.NewReader("{\n  \"email\": \"test@ntnxsherlock.com\",\n  \"password\": \"test\"\n}")
	res, err := request("POST", loginURL, "", payload)
	if err != nil {
		log.Fatal(err)
		return
	}
	m := make(map[string]string)
	err = json.Unmarshal([]byte(res), &m)
	if err != nil {
		log.Fatal(err)
		return
	}
	loginToken := m["token"]
	payload = strings.NewReader("")
	infoURL := *cloudURL + "/v1/edges/" + *edgeid1 + "/info"
	res, err = request("GET", infoURL, loginToken, payload)
	if err != nil {
		log.Fatal(err)
		return
	}
	upgradeListURL := *cloudURL + "/v1/edges/" + *edgeid1 + "/upgradecompatible"
	res, err = request("GET", upgradeListURL, loginToken, payload)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("\nReleases Available: %s\n", res)
	fmt.Printf("{\"release\": \"%s\",\n  \"edgeIds\": [\"%s\"] }\n", release, *edgeid1)
	payload = strings.NewReader(fmt.Sprintf("{\"release\": \"%s\",\n  \"edgeIds\": [\"%s\"] }", release, *edgeid1))
	upgradeURL := *cloudURL + "/v1/edges/upgrade"
	res, err = request("POST", upgradeURL, loginToken, payload)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Printf("Waiting for upgrade to complete\n")
	fmt.Printf("---------------------------------------------------------\n\n")
	//return
	//Check if upgrade is succesful
	intervalTicker := time.NewTicker(time.Minute)
	stopTicker := time.NewTicker(15 * time.Minute)

	// This is needed to start sending as soon as possible
	for {
		select {
		case <-intervalTicker.C:
			payload = strings.NewReader("")
			infoURL = *cloudURL + "/v1/edges/" + *edgeid1 + "/info"
			res, err = request("GET", infoURL, loginToken, payload)
			if err != nil {
				log.Fatal(err)
				return
			}
			resMap := map[string]interface{}{}
			err := json.Unmarshal([]byte(res), &resMap)
			if err != nil {
				glog.Errorf("Failed to unmarshal res %s: %s\n", res, err)
				return
			}
			version := resMap["EdgeVersion"]
			if version == release {
				glog.Infof("Success\n")
				return
			}
			glog.Warningf("Not Upgraded: Retrying after 1 minute\n")
		case <-stopTicker.C:
			glog.Errorf("Error: Edge not upgraded, timeout\n")
			return
		}

	}

}
