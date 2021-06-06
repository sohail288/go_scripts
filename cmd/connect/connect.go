package main

import "fmt"
import "net/http"
import "os"
import "strings"
import "log"
import "io"

func makeRequest(url string, channel chan string) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal("unable to connect to: " + url)		
	}
	
	var body []byte
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("unable to read body for: " + url)		
	}

	// fmt.Println(string(body))
	channel <- string(body)
}

func main() {
	prog, urls := os.Args[:1], os.Args[1:]
	fmt.Printf("%s - %s\n", prog[0], strings.Join(urls, ", "))
	queue := make(chan string, len(urls))
	for _, url := range(urls) {
		go makeRequest(url, queue)
	}
	
	var receivedCount = 0
	var response string
	for receivedCount < len(urls) {
		response = <- queue	
		fmt.Println(response[:10])
		receivedCount += 1
	}
	close(queue)
}
