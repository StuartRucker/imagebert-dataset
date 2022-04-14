package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/rs/xid"
)

func BeWorker() {
	test_webdrive()
}

func test_webdrive() {
	// processWebsite("https://www.google.com/search?q=%22&sxsrf=APq-WBsRVfgtHc3KapN5k47kjlq8CfK5eg%3A1649894032505&source=hp&ei=kGJXYr33GqCdptQPpeKzmAI&iflsig=AHkkrS4AAAAAYldwoGmFA9aiwhycbhZXhZxUlB83TQmz&ved=0ahUKEwi9_-GdnpL3AhWgjokEHSXxDCMQ4dUDCAk&uact=5&oq=%22&gs_lcp=Cgdnd3Mtd2l6EANQAFgAYO8BaABwAHgAgAFjiAFjkgEBMZgBAKABAQ&sclient=gws-wiz")
	// // processWebsite("https://www.miniclip.com/games/en/")
	s := "💰 Crypto📈 Economics🗣️ Language👨‍👩‍👦 Lifestyle🐃 Linux😎 Personal🎓 Philosophy👑 Politics⛪ Religion🥼 Science🖥️ Software⚙️ Technology📜 Tradition📖 Tutorial🆕 Updates"
	// s := "Hello World"
	tokens := maskifyText(s)
	for _, token := range tokens {
		fmt.Printf("%s\t(%s)\n", token.word, token.token)
	}

}

func processWebsite(url string) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// create random sequences of characters to be the run id
	run := Run{id: xid.New().String()}

	//make a new folder data/{run.id} if it doesn't exist
	run.folder = fmt.Sprintf("data/%s", run.id)
	run.tokenCsv = "data/tokens.csv"
	run.url = url
	os.MkdirAll(run.folder, 0755)

	err := chromedp.Run(ctx, processWebsiteHelper(run, url, "html"))
	if err != nil {
		log.Fatal(err)
	}
}

// returns the list of commands to process a website
func processWebsiteHelper(run Run, pageUrl, of string, opts ...chromedp.QueryOption) chromedp.Tasks {
	var nodes []*cdp.Node
	var buf []byte
	return chromedp.Tasks{
		chromedp.EmulateViewport(512, 512),
		chromedp.Navigate(pageUrl),
		chromedp.WaitVisible(`body`, chromedp.ByQuery),

		// populate tree of dom nodes
		chromedp.Nodes(of, &nodes),
		chromedp.ActionFunc(func(c context.Context) error {
			fmt.Printf("Processing Nodes\n")
			return dom.RequestChildNodes(nodes[0].NodeID).WithDepth(-1).Do(c)
		}),

		// wait a little while for dom.EventSetChildNodes to be fired and handled
		chromedp.Sleep(time.Second),
		chromedp.ActionFunc(func(c context.Context) error {
			fmt.Printf("Applying Mask Tag\n")
			maskify(nodes, &c)
			return nil
		}),

		// //reset the dom tree nodes to include maskify tags
		chromedp.ActionFunc(func(c context.Context) error {
			fmt.Printf("Updating nodes after adding mask tags\n")
			return dom.RequestChildNodes(nodes[0].NodeID).WithDepth(-1).Do(c)
		}),

		// // wait a little while for dom.EventSetChildNodes to be fired and handled
		chromedp.Sleep(time.Second),

		chromedp.ActionFunc(func(c context.Context) error {
			//print the len of nodes
			fmt.Printf("%v\n", len(nodes))
			nodeToMaskifies := make(map[cdp.NodeID][]Maskify)

			for i, node := range nodes {
				fmt.Printf("Finding valid screenshots %d/%d\n", i, len(nodes))
				_, nodesToScreenshot := screenshotify(&c, node, &nodeToMaskifies)
				for _, nodeToScreenshot := range nodesToScreenshot {
					screenShotNode(run, &c, nodeToScreenshot)
					logNode(run, &c, nodeToScreenshot.NodeID, nodeToMaskifies[nodeToScreenshot.NodeID])
				}
			}
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

func screenshotify(ctx *context.Context, node *cdp.Node, nodeToMaskify *map[cdp.NodeID][]Maskify) ([]Maskify, []*cdp.Node) {
	maskifies := make([]Maskify, 0)
	// fmt.Printf("Node Name: %s\n", node.NodeName)
	allChildScreenshotNodes := make([]*cdp.Node, 0)
	if len(node.Children) > 0 {
		for _, child := range node.Children {

			// append screenshotify(ctx, child) to maskifies

			childMaskifies, childScreenshotNodes := screenshotify(ctx, child, nodeToMaskify)
			maskifies = append(maskifies, childMaskifies...)
			allChildScreenshotNodes = append(allChildScreenshotNodes, childScreenshotNodes...)
		}
	}

	if node.NodeName == "MASKIFY" {
		m := Maskify{
			word:  node.NodeValue,     //todo not working
			token: node.Attributes[1], // flat key, value, key, value structure
			node:  node,
		}
		// fmt.Printf("APPTCHA%v\n", m)

		//TODO add logic to make sure coordinates check out
		maskifies = append(maskifies, m)
	}

	if len(maskifies) > 50 && len(maskifies) < 512 {
		//screenshot worthy!
		//screenshot the element
		if wouldScreenShotNode(ctx, node) {
			allChildScreenshotNodes = make([]*cdp.Node, 0)
			allChildScreenshotNodes = append(allChildScreenshotNodes, node)
			(*nodeToMaskify)[node.NodeID] = maskifies
		}
	}

	return maskifies, allChildScreenshotNodes
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
