package main

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/buckhx/gobert/tokenize"
	"github.com/buckhx/gobert/tokenize/vocab"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// each maskify struct contains a word, token, and coordinates
type Maskify struct {
	Word   string  `json:"word"`
	Token  string  `json:"token"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
	node   *cdp.Node
}

type maskifyJsOutput struct {
	Data []Maskify `json:"data"`
}

func tokenizeText(text string) []string {

	// todo: move initialization out of the loop

	vocabPath := "vocab.txt"
	voc, err := vocab.FromFile(vocabPath)
	if err != nil {
		panic(err)
	}
	tkz := tokenize.NewTokenizer(voc)

	return tkz.Tokenize(text)
}

func maskify(ctx *context.Context, node *cdp.Node, nodeToMaskify *map[cdp.NodeID][]Maskify) ([]Maskify, []*cdp.Node) {

	maskifies := make([]Maskify, 0)
	allChildScreenshotNodes := make([]*cdp.Node, 0)

	text := node.NodeValue

	if len(node.Children) > 0 {
		for _, child := range node.Children {

			// append screenshotify(ctx, child) to maskifies

			childMaskifies, childScreenshotNodes := maskify(ctx, child, nodeToMaskify)
			maskifies = append(maskifies, childMaskifies...)
			allChildScreenshotNodes = append(allChildScreenshotNodes, childScreenshotNodes...)
		}
	} else if len(text) > 4 && node.NodeType == 3 {

		//read from the file "maskify.js" into a string
		maskify_js := getFile("maskify.js")

		tokens := tokenizeText(text)
		var output maskifyJsOutput
		chromedp.CallFunctionOnNode(*ctx, node, maskify_js, &output, tokens)

		maskifies = append(maskifies, output.Data...)
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

// returns the contents of the file as a string
func getFile(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	var result string
	buf := make([]byte, 1024)
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		result += string(buf[:n])
		if err == io.EOF {
			break
		}
	}
	return result
}
