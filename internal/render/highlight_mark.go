package render

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// ==text== marks a highlight: <span class="hl">text</span>, the phrase a
// theme underlines or sweeps in (AceMQ Pro animates it as the slide arrives).
// Exactly two = on each side, like ~~strikethrough~~, so `a = b` and `==`
// in code are left alone.

var kindHighlight = ast.NewNodeKind("Highlight")

type highlightNode struct{ ast.BaseInline }

func (n *highlightNode) Kind() ast.NodeKind { return kindHighlight }
func (n *highlightNode) Dump(src []byte, level int) {
	ast.DumpHelper(n, src, level, nil, nil)
}

type highlightDelims struct{}

func (highlightDelims) IsDelimiter(b byte) bool                   { return b == '=' }
func (highlightDelims) CanOpenCloser(o, c *parser.Delimiter) bool { return o.Char == c.Char }
func (highlightDelims) OnMatch(consumes int) ast.Node             { return &highlightNode{} }

type highlightParser struct{}

func (highlightParser) Trigger() []byte { return []byte{'='} }

func (highlightParser) Parse(parent ast.Node, block text.Reader, pc parser.Context) ast.Node {
	before := block.PrecendingCharacter()
	line, segment := block.PeekLine()
	node := parser.ScanDelimiter(line, before, 2, highlightDelims{})
	if node == nil || node.OriginalLength != 2 || before == '=' {
		return nil
	}
	node.Segment = segment.WithStop(segment.Start + node.OriginalLength)
	block.Advance(node.OriginalLength)
	pc.PushDelimiter(node)
	return node
}

func (highlightParser) CloseBlock(parent ast.Node, pc parser.Context) {}

type highlightRenderer struct{}

func (highlightRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindHighlight, func(w util.BufWriter, src []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			w.WriteString(`<span class="hl">`)
		} else {
			w.WriteString("</span>")
		}
		return ast.WalkContinue, nil
	})
}

type highlightExt struct{}

func (highlightExt) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(util.Prioritized(highlightParser{}, 500)))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(util.Prioritized(highlightRenderer{}, 500)))
}
