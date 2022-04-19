package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/rs/xid"
)

// URL struct with a list of URLs
type URLs struct {
	mu   sync.Mutex
	list []string
	used map[string]bool
}

func BeWorker() {

	// create directories
	os.MkdirAll("data/img", 0755)
	os.MkdirAll("data/csv", 0755)

	urls := URLs{
		used: make(map[string]bool),
		list: []string{"https://www.google.com/search?q=lovely&rlz=1C5CHFA_enUS999US1000&oq=lovely&aqs=chrome..69i57j46i433i512j46i175i199i512j46i131i433i512j46i433i512j0i433i457i512j69i61l2.4509j0j7&sourceid=chrome&ie=UTF-8"}}

	// while urls is not empty
	for {

		// get a random url from the list
		urls.mu.Lock()
		if len(urls.list) == 0 {
			urls.mu.Unlock()
			continue
		}
		if len(urls.list) > 5000 {
			urls.list = urls.list[:3000]
		}

		delete_index := rand.Intn(len(urls.list))
		url := urls.list[delete_index]
		urls.used[url] = true
		if len(urls.list) > 1 {
			urls.list = append(urls.list[:delete_index], urls.list[delete_index+1:]...)
		} else {
			urls.list = []string{}
		}

		urls.mu.Unlock()

		go func() {
			// process the url

			newUrls := processWebsite(url)
			// print the length of the new urls
			// fmt.Printf("%d new urls\n", len(newUrls))
			urls.mu.Lock()
			//append the newURLs that are not used
			for _, newUrl := range newUrls {
				if _, ok := urls.used[newUrl]; !ok {
					urls.list = append(urls.list, newUrl)
				}
			}
			urls.mu.Unlock()

		}()

		time.Sleep(time.Second)

	}
}

func processWebsite(url string) []string {
	fmt.Printf("Processing %s\n", url)
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// create random sequences of characters to be the run id
	run := Run{id: xid.New().String()}

	//make a new folder data/{run.id} if it doesn't exist

	run.tokenCsv = fmt.Sprintf("data/csv/{%s}-tokens.csv", run.id)
	run.url = url
	//channel of strings
	run.urls = make(chan []string, 1)

	err := chromedp.Run(ctx, processWebsiteHelper(&run, url, "html"))
	if err != nil {
		fmt.Printf("Failed on %s, but continuing", url)
	}

	//wait for run.urls to be populate
	select {
	case found_urls := <-run.urls:
		// fmt.Printf("%d Select found urls\n", len(found_urls))
		return found_urls
	case <-time.After(time.Second * 10):
		fmt.Printf("%d Select timeout\n", len(run.urls))
		return []string{}
	}

}

// returns the list of commands to process a website
func processWebsiteHelper(run *Run, pageUrl, of string, opts ...chromedp.QueryOption) chromedp.Tasks {
	var nodes []*cdp.Node
	var buf []byte
	return chromedp.Tasks{
		chromedp.EmulateViewport(512, 512),
		chromedp.Navigate(pageUrl),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// populate tree of dom nodes
		chromedp.Nodes(of, &nodes),
		chromedp.ActionFunc(func(c context.Context) error {
			// fmt.Printf("Processing Nodes\n")
			return dom.RequestChildNodes(nodes[0].NodeID).WithDepth(-1).Do(c)
		}),

		// wait a little while for dom.EventSetChildNodes to be fired and handled
		chromedp.Sleep(time.Second),
		chromedp.ActionFunc(func(c context.Context) error {
			nodeToMaskifies := make(map[cdp.NodeID][]Maskify)
			for _, node := range nodes {
				// fmt.Printf("Finding valid screenshots %d/%d\n", i, len(nodes))
				_, nodesToScreenshot := maskify(&c, node, &nodeToMaskifies)
				for _, nodeToScreenshot := range nodesToScreenshot {
					clip := getCoordinates(&c, nodeToScreenshot)
					screenShotNode(run, &c, nodeToScreenshot, clip)
					logNode(run, &c, nodeToScreenshot.NodeID, nodeToMaskifies[nodeToScreenshot.NodeID], clip)
				}
			}

			// write extract_links(&c, nodes[0]) to the run.urls channel
			run.urls <- extract_links(&c, nodes[0])

			return nil
		}),

		chromedp.FullScreenshot(&buf, 90),
		chromedp.ActionFunc(func(c context.Context) error {
			if err := ioutil.WriteFile("fullScreenshot.png", buf, 0o644); err != nil {
				log.Fatal(err)
			}
			return nil
		}),
	}

}

type ListofLinks struct {
	Links []string `json:"links"`
}

func extract_links(ctx *context.Context, node *cdp.Node) []string {
	js_code := `function getLinks() {
		const links = document.querySelectorAll("a");
		const links_array = [];
		for (let i = 0; i < links.length; i++) {
			links_array.push(links[i].href);
		}
		return {'links': links_array};
	}`
	var links ListofLinks
	chromedp.CallFunctionOnNode(*ctx, node, js_code, &links)
	return links.Links
}

func getCoordinates(ctx *context.Context, node *cdp.Node) page.Viewport {
	getClientRectJS := `function getClientRect() {
		const e = this.getBoundingClientRect(),
		t = this.ownerDocument.documentElement.getBoundingClientRect();
		return {
			x: e.left - t.left,
			y: e.top - t.top,
			width: e.width,
			height: e.height,
		};
	}`

	var clip page.Viewport
	chromedp.CallFunctionOnNode(*ctx, node, getClientRectJS, &clip)

	x, y := math.Round(clip.X), math.Round(clip.Y)
	clip.Width, clip.Height = math.Round(clip.Width+clip.X-x), math.Round(clip.Height+clip.Y-y)
	clip.Scale = 1
	return clip
}

func wouldScreenShotNode(ctx *context.Context, node *cdp.Node) bool {
	clip := getCoordinates(ctx, node)
	return clip.Height <= 256 && clip.Height >= 66
}

func fullScreenshot(urlstr string, quality int, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(urlstr),
		chromedp.FullScreenshot(res, quality),
	}
}
