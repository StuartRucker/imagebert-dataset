package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/buckhx/gobert/tokenize"
	"github.com/buckhx/gobert/tokenize/vocab"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// each maskify struct contains a word, token, and coordinates
type Maskify struct {
	word        string
	token       string
	start_index int
	end_index   int
	node        *cdp.Node
}

func maskifyText(text string) []Maskify {
	fmt.Printf("%s\n", text)
	// todo: move initialization out of the loop

	vocabPath := "vocab.txt"
	voc, err := vocab.FromFile(vocabPath)
	if err != nil {
		panic(err)
	}
	tkz := tokenize.NewTokenizer(voc)
	fmt.Print("Initialized Tokenizer\n")

	tokens := tkz.Tokenize(text)

	fmt.Print("tokenized...\n")

	//make a list of tuples
	maskifiedWords := make([]Maskify, len(tokens))
	last_index := 0

	unknownsEncountered := 0
	true_position := 0
	for i, token := range tokens {
		fmt.Printf("%s\n", token)
		// find the next instance of the token in the text

		if token == "[UNK]" {
			unknownsEncountered++
			continue
		}

		use_token := token
		if strings.HasPrefix(token, "##") {
			use_token = token[2:]
		}

		start_index := strings.Index(text, use_token)
		end_index := start_index + len(use_token)

		// iterate for each unknown encountered
		// divide the text between last_index and start_index into [UNK] maskify tokens
		// for j := 1; j <= unknownsEncountered; j++ {

		// 	if j == 1 {

		// 		maskifiedWords[i-j] = Maskify{
		// 			word:        text[last_index:start_index],
		// 			token:       "[UNK]",
		// 			start_index: last_index,
		// 			end_index:   start_index,
		// 		}
		// 		last_index = start_index
		// 	}
		// 	if j != 1 {
		// 		maskifiedWords[i-j] = Maskify{
		// 			word:        "",
		// 			token:       "[UNK]",
		// 			start_index: start_index,
		// 			end_index:   start_index,
		// 		}

		// 	}
		// }
		unknownsEncountered = 0

		m := Maskify{
			word:        text[:end_index],
			token:       use_token,
			start_index: last_index,
			end_index:   end_index,
		}
		maskifiedWords[i] = m

		text = text[end_index:]
		true_position += end_index

	}

	return maskifiedWords
}

/* wraps the text of every node in the subtree of the given node in a <maskify> token
 Example:
<li id="listItem">
    This is some text
    <span id="firstSpan">First span text</span>
    <span id="secondSpan">Second span text</span>
</li>

 becomes:
<li id="listItem">
	<maskify>This</maskify> <maskify>is</maskify> <maskify>some</maskify> <maskify>text</maskify>
	<span id="firstSpan"><maskify>First</maskify> <maskify>span</maskify> <maskify>text</maskify></span>
	<span id="secondSpan"><maskify>Second</maskify> <maskify>span</maskify> <maskify>text</maskify></span>
</li>
*/
func maskify(nodes []*cdp.Node, ctx *context.Context) {
	//iterte through each node
	for _, node := range nodes {
		//print node.NodeName
		//if there is more than one child, then we need to maskify the children
		if len(node.Children) > 0 {
			//recursively maskify the children
			maskify(node.Children, ctx)
		}
		//extract the text from the node
		text := node.NodeValue
		//if the node is a text node, then we need to maskify it
		//check if is a text node
		if len(text) > 4 && node.NodeType == 3 {
			//maskify the text
			// (*replaceMap)[text] = maskifyText(text)
			maskifiedText := maskifyText(text)
			// fmt.Printf("%s\n", maskifiedText)
			jsfunc := `function maskifyTextJS(inputHTMlString) {
					function createElementFromHTML(htmlString) {
						var div = document.createElement('metamaskify');
						div.innerHTML = "<metamaskify>" + htmlString.trim() + "</metamaskify>";

						// Change this to div.childNodes to support multiple top-level nodes.
						return div.firstChild;
					  }
					var elements = createElementFromHTML(inputHTMlString);
					this.replaceWith(elements);
				}`

			// callFunctionOnNode(ctx, nodes, jsfunc, nil)
			//print the node id
			// fmt.Printf("%v\n", node.NodeID)
			chromedp.CallFunctionOnNode(*ctx, node, jsfunc, nil, maskifiedText)

			// err := dom.SetNodeValue(node.NodeID, maskifyText(text)).Do(*ctx)
			// if err != nil {
			// 	log.Fatal(err)
			// }
		}
	}
}
