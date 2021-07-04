package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

func processPage(url string) []string {
	return []string{"https://google.com?q=abc"}
}

type Crawler struct {
	CrawledUrls            []string
	urlsSeen               map[string]bool
	shouldFollowSubdomains bool
	maxLinksToCrawl        int
	pageProcessor          PageProcessor
}

type PageProcessor interface {
	ProcessHtml(html string)
	GetLinks() []string
}

type DefaultPageProcessor struct {
	extractedLinks []string
}

func (processor *DefaultPageProcessor) ProcessHtml(htmlResponse string) {
	// terrible way of doing this...
	doc, err := html.Parse(strings.NewReader(htmlResponse))
	processor.extractedLinks = []string{}
	if err != nil {
		return
	}
	var f func(*html.Node)

	// from: https://pkg.go.dev/golang.org/x/net/html
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					processor.extractedLinks = append(processor.extractedLinks, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
}

func (processor *DefaultPageProcessor) GetLinks() []string {
	return processor.extractedLinks
}

func (crawler *Crawler) makeRequest(url string) (string, error) {
	resp, err := http.Get(url)

	if err != nil {
		log.Println("Unable to crawl", url)
		return "", err
	}
	var body []byte
	body, err = io.ReadAll(resp.Body)

	if err != nil {
		log.Println("Unable to read body at", url)
		return "", err
	}

	return string(body), nil
}

func (crawler *Crawler) processor(receiveChannel chan string, sendChannel chan string) {

	receivingChannelClosed := false
	inflight := 0
	_requester := func(urlToProcess string, resultChannel chan string) {
		response, err := crawler.makeRequest(urlToProcess)
		if err != nil {
			resultChannel <- ""
			return
		}
		resultChannel <- response
		inflight -= 1

		log.Println("Inlight to process: ", inflight)
		if inflight == 0 && receivingChannelClosed {
			log.Println("Closing processor sending channel")
			close(sendChannel)
		}
	}

	for urlToProcess := range receiveChannel {
		log.Println("Processing ", urlToProcess)
		go _requester(urlToProcess, sendChannel)
		inflight += 1
	}

	// to notify the goroutine that it is done?
	receivingChannelClosed = true
	log.Println("Receiving channel closed?")
}

func (crawler *Crawler) crawl(entrypoint string) error {
	log.Println("Crawling", entrypoint)
	crawler.pageProcessor.ProcessHtml("")
	log.Println(crawler.pageProcessor.GetLinks())

	var currentUrl string
	var sendingChannelOpen bool = true
	sendingChannel := make(chan string, 4)
	receivingChannel := make(chan string)
	var frontier []string = []string{entrypoint}
	crawler.urlsSeen[entrypoint] = true

	inflight := 0

	go crawler.processor(sendingChannel, receivingChannel)

	for len(frontier) > 0 || inflight > 0 {
		shouldSendData := len(frontier) > 0

		if shouldSendData {
			currentUrl, frontier = frontier[0], frontier[1:]
			log.Println("processing", currentUrl)

			select {
			case sendingChannel <- currentUrl:
				log.Println("Sent : ", currentUrl)
				inflight += 1
			default:
			}
		}

		select {
		case response, ok := <-receivingChannel:
			log.Println("Received Response")
			// we want to process at most n pages
			// when we pop off the frontier, we will check if we can send a page to process
			// if we can't send, then we will see if we can receive a page result
			// each time we receive a page result, we update state of crawler and add new
			// this function will close the sending channel if no more links are left
			// the processor will close the send result channel when all inflights are processed
			inflight -= 1
			crawler.pageProcessor.ProcessHtml(response)
			nextLinks := crawler.pageProcessor.GetLinks()
			for _, link := range nextLinks {
				if _, ok = crawler.urlsSeen[link]; !ok {
					frontier = append(frontier, link)
				}
				crawler.urlsSeen[link] = true
			}
		default:
			if inflight == 0 {
				close(receivingChannel)
			}
		}

		if inflight == 0 && len(frontier) == 0 && sendingChannelOpen {
			close(sendingChannel)
			sendingChannelOpen = false
			// we're basically done at this point?
			log.Println("Closed sending channel")
		}

	}
	return nil
}

func main() {
	// get vars from cli
	var (
		entryUrl         string
		followSubdomains bool
		linksToCrawl     int
	)
	flag.StringVar(&entryUrl, "entry-url", "", "the url to start crawls from")
	flag.BoolVar(&followSubdomains, "follow-subdomains", false, "should follow subdomains")
	flag.IntVar(&linksToCrawl, "links-to-crawl", 10, "how many links should crawl")
	flag.Parse()

	fmt.Printf("Starting crawl on: %s\n", entryUrl)
	parsedUrl, err := url.Parse(entryUrl)

	if err != nil {
		log.Fatal(fmt.Sprintf("Unable to process entry: %s"), entryUrl)
	}
	log.Println(parsedUrl.Host)

	processor := DefaultPageProcessor{}
	crawler := Crawler{maxLinksToCrawl: linksToCrawl, shouldFollowSubdomains: followSubdomains, pageProcessor: &processor, urlsSeen: make(map[string]bool)}

	err = crawler.crawl(entryUrl)
	/*

	  allow page processor handle func, it's data structure will contain data for each page
	  set limit or depth of crawl
	  when crawling
	    extract links of current page
	    run page processor on page
	    // allow concurrent crawls?
	    for each extracted link
	      if it isn't already seen
	      add it to a hashmap (link) -> bool
	      if not already seen, add to queue

	      worker processors process a page from queue
	        they receive data via channels
	        send results through channels (or directly post results?)

	      when is it done? queue is empty
	*/
}
