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
}

func screenShotNode(run Run, ctx *context.Context, node *cdp.Node) {

	clip := getCoordinates(ctx, node)

	// take screenshot of the box
	buf, _ := page.CaptureScreenshot().
		WithFormat(page.CaptureScreenshotFormatPng).
		WithCaptureBeyondViewport(true).
		WithClip(&clip).
		Do(*ctx)

	// write to {run.folder}/{node.NodeID}.png
	fileName := fmt.Sprintf("%s/%d.png", run.folder, node.NodeID)
	if err := ioutil.WriteFile(fileName, buf, 0o644); err != nil {
		log.Fatal(err)
	}
}

/*  Write a CSV of form
run.url, run.id, maskifies.word, maskifies.token, coords.x, coords.y, coords.width, coords.height
*/
func logNode(run Run, ctx *context.Context, nodeId cdp.NodeID, maskifies []Maskify) {
	f, err := os.OpenFile(run.tokenCsv, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	for _, maskify := range maskifies {
		coords := getCoordinates(ctx, maskify.node)
		line := fmt.Sprintf("\"%s\",\"%s\",%d,\"%s\",\"%s\",%d,%d,%d,%d\n", run.url, run.id, nodeId, maskify.word, maskify.token, int(coords.X), int(coords.Y), int(coords.Width), int(coords.Height))
		//append line to run.tokencsv

		if _, err := f.WriteString(line); err != nil {
			log.Println(err)
		}
	}

}
