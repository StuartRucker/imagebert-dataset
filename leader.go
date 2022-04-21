//class called Leader with start method

package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Leader struct {
	mu                  sync.Mutex
	urls                []string
	used                map[string]bool
	usedFileLock        sync.Mutex
	urlsFileLock        sync.Mutex
	currentlyActive     int
	currentlyActiveLock sync.Mutex
}

type GetURLArgs struct{}
type GetURLReply struct {
	URL string
}

func (l *Leader) GetURL(args GetURLArgs, reply *GetURLReply) error {
	// get a url from the list
	url := l.generateURL()
	reply.URL = url
	return nil
}

// appends the url to data/used.txt, and creates it if it doesnt exist
func (l *Leader) writeUsed(url string) {
	l.usedFileLock.Lock()
	fileUsed, err := os.OpenFile("data/used.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	if _, err = fileUsed.WriteString(url + "\n"); err != nil {
		panic(err)
	}
	fileUsed.Close()
	l.usedFileLock.Unlock()

}

func (l *Leader) generateURL() string {
	// get a random url from the list
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.urls) == 0 {
		return ""
	}
	if len(l.urls) > 5000 {
		l.urls = l.urls[:3000]
	}

	delete_index := rand.Intn(len(l.urls))
	url := l.urls[delete_index]

	l.used[url] = true
	go l.writeUsed(url)
	if len(l.urls) > 1 {
		l.urls = append(l.urls[:delete_index], l.urls[delete_index+1:]...)
	} else {
		l.urls = []string{}
	}

	return url
}

// if data/used.txt exists, add each line to l.used
func (l *Leader) initializeUsed() {
	l.used = make(map[string]bool)
	file, err := os.Open("data/used.txt")
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l.used[scanner.Text()] = true
	}

	// read data/urls.txt into l.urls if it exists
	file, err = os.Open("data/urls.txt")
	if err != nil {
		return
	}
	defer file.Close()
	scanner = bufio.NewScanner(file)
	for scanner.Scan() {
		l.urls = append(l.urls, scanner.Text())
	}

}
func (l *Leader) initializeUrls() {
	go func() {
		for {
			l.urlsFileLock.Lock()

			//write the urls to data/urls.txt
			file, err := os.OpenFile("data/urls.txt", os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				panic(err)
			}
			for _, url := range l.urls {
				if len(url) > 2 {
					fmt.Fprintln(file, url)
				}
			}
			file.Close()
			l.urlsFileLock.Unlock()

			time.Sleep(time.Second * 5)
		}
	}()
}

func (l *Leader) run() {
	// create directories
	os.MkdirAll("data/img", 0755)
	os.MkdirAll("data/csv", 0755)

	l.initializeUsed()
	l.initializeUrls()

	// while urls is not empty
	for {
		url := l.generateURL()
		if url == "" || l.currentlyActive >= 10 {
			time.Sleep(time.Second)
			continue
		}

		go func() {
			// process the url

			l.currentlyActiveLock.Lock()
			l.currentlyActive++
			l.currentlyActiveLock.Unlock()

			newUrls := processWebsite(url)
			// print the length of the new urls
			// fmt.Printf("%d new urls\n", len(newUrls))
			l.mu.Lock()
			//append the newURLs that are not used
			for _, newUrl := range newUrls {
				if _, ok := l.used[newUrl]; !ok {
					l.urls = append(l.urls, newUrl)
				}
			}
			l.mu.Unlock()

			l.currentlyActiveLock.Lock()
			l.currentlyActive--
			l.currentlyActiveLock.Unlock()

		}()

		// sleep for 10 millseconds
		time.Sleep(10 * time.Millisecond)

	}
}
