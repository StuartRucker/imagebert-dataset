package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
)

/*
Handles logic involving saving to files
*/

// struct to contain run meta data
type Run struct {
	id       string
	folder   string
	url      string
	tokenCsv string
	//create a channel that holds a list of urls
	urls chan []string
}

func screenShotNode(run *Run, ctx *context.Context, node *cdp.Node, clip page.Viewport) {

	// take screenshot of the box
	buf, _ := page.CaptureScreenshot().
		WithFormat(page.CaptureScreenshotFormatPng).
		WithCaptureBeyondViewport(true).
		WithClip(&clip).
		Do(*ctx)

	// write to {run.folder}/{node.NodeID}.png
	fileName := fmt.Sprintf("data/img/%s-%d.png", run.id, node.NodeID)
	if err := ioutil.WriteFile(fileName, buf, 0o644); err != nil {
		log.Fatal(err)
	}
}

/*  Write a CSV of form
run.url, run.id, maskifies.word, maskifies.token, coords.x, coords.y, coords.width, coords.height
*/
func logNode(run *Run, ctx *context.Context, nodeId cdp.NodeID, maskifies []Maskify, clip page.Viewport) {
	f, err := os.OpenFile(run.tokenCsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	f.WriteString("URL,RUNID,NODEID,WORD,TOKEN,X,Y,WIDTH,HEIGHT\n")

	for _, maskify := range maskifies {

		// make sure maskify is within the clip
		if maskify.X < clip.X || maskify.Y < clip.Y || maskify.X+maskify.Width > clip.X+clip.Width || maskify.Y+maskify.Height > clip.Y+clip.Height {
			continue
		}

		line := fmt.Sprintf("\"%s\",\"%s\",%d,\"%s\",\"%s\",%d,%d,%d,%d\n", run.url, run.id, nodeId, maskify.Word, maskify.Token, int(maskify.X-clip.X), int(maskify.Y-clip.Y), int(maskify.Width), int(maskify.Height))
		//append line to run.tokencsv

		if _, err := f.WriteString(line); err != nil {
			log.Println(err)
		}
	}

}

// Todo: meta logging with the whole text
