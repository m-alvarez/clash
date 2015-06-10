package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

const BlockSize = 3

type WordBlock [BlockSize]string
type Prefix [BlockSize - 1]string
type Suffix [BlockSize - 1]string

func (b WordBlock) Prefix() Prefix {
	p := [BlockSize - 1]string{}
	for i := range p {
		p[i] = b[i]
	}
	return p
}
func (b WordBlock) Suffix() Suffix {
	p := [BlockSize - 1]string{}
	for i := range p {
		p[i] = b[i+1]
	}
	return p
}

type Node struct {
	text     WordBlock
	tag      string
	inLinks  []*Node
	outLinks []*Node
	end      bool
}

func (node *Node) Link(other *Node) {
	if other.tag == node.tag {
		node.inLinks = append(node.inLinks, other)
	} else {
		node.outLinks = append(node.outLinks, other)
	}
}

func (node *Node) HasLinks() bool {
	return len(node.inLinks) > 0 || len(node.outLinks) > 0
}

type Network struct {
	starts   []*Node
	nodes    map[WordBlock][]*Node
	prefixes map[Prefix][]*Node
	suffixes map[Suffix][]*Node
}

func Compose(funcs ...bufio.SplitFunc) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF || len(data) == 0 {
			return 0, nil, nil
		}

		for _, fn := range funcs {
			advance, token, err := fn(data, atEOF)
			if err == nil {
				return advance, token, err
			}
		}

		fmt.Println("Nothing worked!!")
		fmt.Println("Fuckup with string", string(data[0:10]))
		return 0, nil, nil
	}
}

func IsWhitespace(char byte) bool {
	switch char {
	case ' ', '\t', '\n':
		return true
	default:
		return false
	}
}

func SkippingWhitespace(fn bufio.SplitFunc) bufio.SplitFunc {
	return func(data []byte, atEOF bool) (int, []byte, error) {
		pos := 0

		for pos < len(data) && IsWhitespace(data[pos]) {
			pos++
		}

		if pos == len(data) {
			return 0, nil, nil
		}

		advance, token, err := fn(data[pos:], atEOF)
		if err == nil {
			advance += pos
		}
		return advance, token, err
	}
}

func ScanDots(data []byte, atEOF bool) (int, []byte, error) {
	pos := 0
	for pos < len(data) && data[pos] == '.' {
		pos++
	}

	if pos == 0 {
		return 0, nil, errors.New("No dots")
	} else {
		return pos, data[:pos], nil
	}
}

func ScanPunctuation(data []byte, atEOF bool) (int, []byte, error) {
	switch data[0] {
	case ',', ';', ':', '"', '\'', '-', '?', '(', ')', '!', '/', '\\':
		return 1, data[0:1], nil
	default:
		return 0, nil, errors.New("No punctuation")
	}
}

func IsAlpha(char byte) bool {
	switch {
	case 'a' <= char && char <= 'z':
		return true
	case 'A' <= char && char <= 'Z':
		return true
	case '0' <= char && char <= '9':
		return true
	default:
		return false
	}
}

func ScanWord(data []byte, atEOF bool) (int, []byte, error) {
	pos := 0
	for pos < len(data) && IsAlpha(data[pos]) {
		pos++
	}

	if pos == 0 {
		return 0, nil, errors.New("No word")
	} else {
		return pos, data[:pos], nil
	}
}

func Tokenize(text string) ([]string, error) {
	// Parsing text is my least favourite activity of all time

	splitter := Compose(
		SkippingWhitespace(ScanDots),
		SkippingWhitespace(ScanPunctuation),
		SkippingWhitespace(ScanWord),
	)

	scanner := bufio.NewScanner(strings.NewReader(strings.ToLower(text)))
	scanner.Split(splitter)

	var contents []string
	for scanner.Scan() {
		contents = append(contents, scanner.Text())
	}
	return contents, scanner.Err()
}

func (p *WordBlock) Before(q *WordBlock) bool {
	for i, str := range p {
		if i != 0 && str == q[i-1] {
			return false
		}
	}
	return true
}

func (n *Network) AddNode(node *Node) {
	hasPreviousNode := false
	previousIsEnd := false

	if nodesWithSameText, ok := n.nodes[node.text]; ok {
		n.nodes[node.text] = append(nodesWithSameText, node)
	} else {
		n.nodes[node.text] = []*Node{node}
	}

	if nodesWithSamePrefix, ok := n.prefixes[node.text.Prefix()]; ok {
		n.prefixes[node.text.Prefix()] = append(nodesWithSamePrefix, node)
	} else {
		n.prefixes[node.text.Prefix()] = []*Node{node}
	}

	if nodesWithSameSuffix, ok := n.suffixes[node.text.Suffix()]; ok {
		n.suffixes[node.text.Suffix()] = append(nodesWithSameSuffix, node)
	} else {
		n.suffixes[node.text.Suffix()] = []*Node{node}
	}

	if nodesBefore, ok := n.suffixes[Suffix(node.text.Prefix())]; ok {
		for _, previousNode := range nodesBefore {
			previousNode.Link(node)
			if previousNode.tag == node.tag {
				hasPreviousNode = true
				if previousNode.text[0] == "." {
					previousIsEnd = true
				}
			}
		}
	}
	if nodesAfter, ok := n.prefixes[Prefix(node.text.Suffix())]; ok {
		for _, nextNode := range nodesAfter {
			node.Link(nextNode)
		}
	}

	if !hasPreviousNode || previousIsEnd {
		n.starts = append(n.starts, node)
	}
}

func (n *Network) AddText(tag string, text string) {
	tokens, err := Tokenize(text)
	if err != nil {
		// THE STRING IS ON FIRE
		panic(err)
	}

	for i := range tokens {
		if i+BlockSize > len(tokens) {
			break
		}
		block := [BlockSize]string{}
		for j := range block {
			block[j] = tokens[i+j]
		}
		node := Node{
			tag:      tag,
			text:     block,
			inLinks:  []*Node{},
			outLinks: []*Node{},
			end:      block[BlockSize-1] == ".",
		}
		n.AddNode(&node)
	}
}

func NewNetwork() Network {
	return Network{
		starts:   []*Node{},
		nodes:    make(map[WordBlock][]*Node),
		prefixes: make(map[Prefix][]*Node),
		suffixes: make(map[Suffix][]*Node),
	}
}

func (n Network) RandomPath(maxLength int) []string {
	start := rand.Intn(len(n.starts))
	current := n.starts[start]
	length := 0
	path := []string{}
	changed := false

	for length < maxLength && current.HasLinks() {
		path = append(path, current.text[0])
		if current.end {
			length++
		}

		// Choose a link at random -- prioritize nodes without the same tag.
		var preferred, avoided []*Node
		if changed {
			preferred, avoided = current.inLinks, current.outLinks
		} else {
			preferred, avoided = current.outLinks, current.inLinks
			if len(current.outLinks) > 0 {
				//	changed = true
			}
		}
		if len(preferred) > 0 {
			current = preferred[rand.Intn(len(preferred))]
		} else {
			current = avoided[rand.Intn(len(avoided))]
		}
	}

	path = append(path, current.text[0:BlockSize-1]...)

	return path
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	network := NewNetwork()

	num, _ := strconv.Atoi(os.Args[3])

	bytes1, _ := ioutil.ReadFile(os.Args[1])
	bytes2, _ := ioutil.ReadFile(os.Args[2])

	string1 := string(bytes1)
	string2 := string(bytes2)

	network.AddText(os.Args[1], string1)
	network.AddText(os.Args[2], string2)

	for _, str := range network.RandomPath(num) {
		switch str {
		case ".", ",", "-", ":", ";", "?", "!", "\"", "'":
			fmt.Print(str)
		default:
			fmt.Print(" ", str)
		}
	}
	fmt.Println()
}
