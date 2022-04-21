package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/rs/xid"
)

func processWebsite(url string) []string {
	fmt.Printf("Processing %s\n", url)
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// create random sequences of characters to be the run id
	run := Run{id: xid.New().String()}

	//make a new folder data/{run.id} if it doesn't exist

	run.tokenCsv = fmt.Sprintf("data/csv/%s-tokens.csv", run.id)
	run.url = url
	//channel of strings
	run.urls = make(chan []string, 1)

	err := chromedp.Run(ctx, processWebsiteHelper(&run, url, "html"))
	if err != nil {
		fmt.Printf("Failed on %s, but continuing\n", url)
	}

	//wait for run.urls to be populate
	select {
	case found_urls := <-run.urls:
		// fmt.Printf("%d Select found urls\n", len(found_urls))
		return found_urls
	case <-time.After(time.Second * 600):
		fmt.Printf("%d Select timeout\n", len(run.urls))
		return []string{}
	}

}

// returns the list of commands to process a website
func processWebsiteHelper(run *Run, pageUrl, of string, opts ...chromedp.QueryOption) chromedp.Tasks {
	var nodes []*cdp.Node
	var buf []byte
	// bool map to track visible nodes
	var visible map[*cdp.Node]bool

	return chromedp.Tasks{
		chromedp.EmulateViewport(512, 512),
		chromedp.Navigate(pageUrl),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		//populate tree of dom nodes
		chromedp.Nodes(of, &nodes),
		chromedp.ActionFunc(func(c context.Context) error {
			// fmt.Printf("Processing Nodes\n")
			return dom.RequestChildNodes(nodes[0].NodeID).WithDepth(-1).Do(c)
		}),

		// wait a little while for dom.EventSetChildNodes to be fired and handled
		chromedp.Sleep(time.Second),
		chromedp.ActionFunc(func(c context.Context) error {
			// get valid nodes
			visible = make(map[*cdp.Node]bool)
			for _, node := range nodes {
				checkVisible(&c, &visible, run, node)
			}
			return nil
		}),
		chromedp.ActionFunc(func(c context.Context) error {
			nodeToMaskifies := make(map[cdp.NodeID][]Maskify)
			for _, node := range nodes {
				// fmt.Printf("Finding valid screenshots %d/%d\n", i, len(nodes))
				_, nodesToScreenshot := maskify(&c, &visible, node, &nodeToMaskifies)
				for _, nodeToScreenshot := range nodesToScreenshot {
					clip := getCoordinates(&c, nodeToScreenshot)
					screenShotAndSaveNode(run, &c, nodeToScreenshot, clip)
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

func checkVisibleSimple(ctx *context.Context, visible *map[*cdp.Node]bool, run *Run, node *cdp.Node) {
	(*visible)[node] = true
	for _, child := range node.Children {
		checkVisibleSimple(ctx, visible, run, child)
	}
}

func checkVisible(ctx *context.Context, visible *map[*cdp.Node]bool, run *Run, node *cdp.Node) {
	clip := getCoordinates(ctx, node)

	// take screenshot of the box
	buf1, _ := page.CaptureScreenshot().
		WithFormat(page.CaptureScreenshotFormatPng).
		WithCaptureBeyondViewport(true).
		WithClip(&clip).
		Do(*ctx)

	jsFunc := `function makeSelected(){
			let range = new Range()
			range.selectNode(this)
			let selection = window.getSelection()
			selection.removeAllRanges()
			selection.addRange(range)
		}
	`

	// run the jsFunc on the node
	chromedp.CallFunctionOnNode(*ctx, node, jsFunc, nil)

	buf2, _ := page.CaptureScreenshot().
		WithFormat(page.CaptureScreenshotFormatPng).
		WithCaptureBeyondViewport(true).
		WithClip(&clip).
		Do(*ctx)

	//purely for testing ideally
	// fileName := fmt.Sprintf("tmp/%s-%d-normal.png", run.id, node.NodeID)
	// if err := ioutil.WriteFile(fileName, buf1, 0o644); err != nil {
	// 	log.Fatal(err)
	// }

	// fileName = fmt.Sprintf("tmp/%s-%d-modified.png", run.id, node.NodeID)
	// if err := ioutil.WriteFile(fileName, buf2, 0o644); err != nil {
	// 	log.Fatal(err)
	// }

	same := buf1 != nil && buf2 != nil && len(buf1) > 0 && !bytes.Equal(buf1, buf2)
	(*visible)[node] = same

	for _, child := range node.Children {
		checkVisible(ctx, visible, run, child)
	}
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
