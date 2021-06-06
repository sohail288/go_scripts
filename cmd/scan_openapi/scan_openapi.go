/*
  Parses and Scans OpenAPI definitions
*/
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	// "io"
	"io/ioutil"
	"net/http"
	"os"
	// "path/filepath"
)

type OpenApiInfo struct {
	Title   string `json:"title,omitempty"`
	Version string `json:"version,omitempty"`
}

type OpenApiServer struct {
	Url         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type OpenApiResponse struct {
	Description string `json:"description,omitempty"`
}

type OpenApiParameter struct {
	Name        string `json:"description,omitempty"`
	In          string `json:"in,omitempty"`
	Description string `json:"description,omitempty"`
}

type OpenApiOperationObject struct {
	Security    []OpenApiSecurityDeclaration `json:"security,omitempty"`
	OperationId string                       `json:"operationId"`
	Parameters  []OpenApiParameter           `json:"parameters,omitempty"`
	Responses   map[int]OpenApiResponse      `json:"responses,omitempty"`
}

type OpenApiSecurityDeclaration map[string][]string
type OpenApiPathObject map[string]OpenApiOperationObject

type OpenApiDefinition struct {
	OpenApi  string                       `json:"openapi"`
	Info     OpenApiInfo                  `json:"info"`
	Security []OpenApiSecurityDeclaration `json:"security"`
	Servers  []OpenApiServer              `json:"servers"`
	Paths    map[string]OpenApiPathObject `json:"paths"`
}

var httpMethods = []string{"get", "post", "patch", "delete", "put"}

func findStringInArray(s string, arr []string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

type ScanResult struct {
	scanRequest *ScanRequest
	err         error
	response    *http.Response
}

type ScanRequest struct {
	serverUrl    string
	path         string
	method       string
	operationObj *OpenApiOperationObject
}

func makeRequest(client *http.Client, queue chan *ScanResult, scanRequest *ScanRequest) {
	var (
		err  error
		req  *http.Request
		resp *http.Response
	)
	if !findStringInArray(strings.ToLower(scanRequest.method), httpMethods) {
		queue <- &ScanResult{scanRequest, nil, nil}
		return
	}

	endpointUrl := fmt.Sprintf("%s%s", scanRequest.serverUrl, scanRequest.path)

	req, err = http.NewRequest(strings.ToUpper(scanRequest.method), endpointUrl, nil)
	if err != nil {
		queue <- &ScanResult{scanRequest, err, nil}
		return
	}

	resp, err = client.Do(req)
	if err != nil {
		queue <- &ScanResult{scanRequest, err, nil}
		return
	}

	queue <- &ScanResult{scanRequest, nil, resp}
}

func exitError(err error, msg string) {
	if err != nil {
		log.Fatal(msg)
	}
}

func main() {
	fmt.Println("scan_openapi")
	var (
		err        error
		contents   []byte
		openapiDef OpenApiDefinition
	)

	if len(os.Args) < 2 {
		log.Fatal("Usage: " + os.Args[0] + " filepath")
	}
	defPath := os.Args[1]

	contents, err = ioutil.ReadFile(defPath)
	exitError(err, "unable to open file")

	err = json.Unmarshal(contents, &openapiDef)
	exitError(err, "unable to unmarshal def")

	fmt.Println("scanning: " + openapiDef.Info.Title)
	fmt.Println(openapiDef.Paths["/get"])

	client := &http.Client{}
	queue := make(chan *ScanResult, 10)

	var scanRequests []*ScanRequest

	// create the scan requests
	serverUrl := openapiDef.Servers[0].Url
	for path, pathObj := range openapiDef.Paths {
		for method, operationObj := range pathObj {
			scanRequests = append(scanRequests, &ScanRequest{serverUrl, path, method, &operationObj})
		}
	}

	totalRequests := len(scanRequests)
	receivedCount := 0
	var result *ScanResult

	// finally make the request with a channel
	for _, request := range scanRequests {
		go makeRequest(client, queue, request)
	}

	// wait here to get all the scan results
	for receivedCount < totalRequests {
		result = <-queue
		fmt.Printf("%s %s%s - %s\n", result.scanRequest.method, result.scanRequest.serverUrl, result.scanRequest.path, result.response.Status)
		receivedCount += 1
	}
	close(queue)
}
