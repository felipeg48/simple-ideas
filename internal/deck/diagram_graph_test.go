package deck

import (
	"strings"
	"testing"
)

func TestState(t *testing.T) {
	d, probs := ParseDiagram("state", nil, "Placed → Paid on authorised\nPaid → Shipped on picked\nPlaced → Cancelled on timeout\nevent authorised\nevent picked\nevent nope")
	if len(probs) != 1 || !strings.Contains(probs[0], `no transition from Shipped on "nope"`) {
		t.Errorf("problems %q", probs)
	}
	nodes := elsOf(d, "si-dg-node")
	if len(nodes) != 4 || !strings.Contains(d.Els[nodes[0]].C, "si-dg-mid") || d.States[0][nodes[0]].C != "is-on" {
		t.Fatalf("nodes %v", nodes)
	}
	if d.States[2][nodes[2]].C != "is-on" || d.States[2][nodes[0]].C != "" || d.Steps[1] != "picked → Shipped" {
		t.Errorf("after picked: %+v, steps %q", d.States[2][nodes[2]], d.Steps)
	}
}

func TestFlowchart(t *testing.T) {
	d, probs := ParseDiagram("flowchart", nil, "Start → In stock?\nIn stock? → Pay: yes\nIn stock? → Wait: no\nWait → Pay\nPay → End\ntoken Start → In stock? → Pay → End → Nowhere")
	if len(probs) != 1 || !strings.Contains(probs[0], `"Nowhere" is not a box`) {
		t.Errorf("problems %q", probs)
	}
	if len(elsOf(d, "si-dg-dec")) != 1 || len(elsOf(d, "is-end")) != 0 && !strings.Contains(d.Els[elsOf(d, "si-dg-node")[0]].C, "is-end") {
		t.Error("a decision diamond, start and end pills")
	}
	tok := elsOf(d, "si-dg-token")
	if len(tok) != 1 || len(d.States) != 5 || d.States[0][tok[0]].O != 0 || d.States[3][tok[0]].O != 1 {
		t.Fatalf("the token moves one box per step: %d states", len(d.States))
	}
	if d.States[3][tok[0]].X == d.States[2][tok[0]].X {
		t.Error("the token moved")
	}
}

func TestClass(t *testing.T) {
	d, probs := ParseDiagram("class", nil, "Order: id, total()\nLineItem: sku\nOrder has many LineItem\nCard is a Payment\nOrder uses Payment\nwhat is this:: nonsense here")
	if len(probs) != 1 {
		t.Errorf("problems %q", probs)
	}
	cards := elsOf(d, "si-dg-cls")
	if len(cards) != 4 || len(d.Els[cards[0]].Lines) != 2 {
		t.Fatalf("cards %v", cards)
	}
	if len(elsOf(d, "is-inherit")) != 1 || len(elsOf(d, "is-uses")) != 1 {
		t.Error("an inheritance and a uses relation")
	}
	if len(d.States) != 4 || d.States[0][cards[1]].O != 0 || d.States[1][cards[1]].O != 1 {
		t.Error("one class per step")
	}
}

func TestWBS(t *testing.T) {
	d, probs := ParseDiagram("wbs", nil, "- Lockers\n  - Design\n    - Research · 2d\n    - Screens · 3d\n  - Build · 8d\n- Second root")
	if len(probs) != 1 {
		t.Errorf("problems %q", probs)
	}
	boxes := elsOf(d, "si-dg-wb")
	if len(boxes) != 5 || d.States[len(d.States)-1][boxes[0]].U != "13 days" || d.States[len(d.States)-1][boxes[1]].U != "1 · 5 days" {
		t.Fatalf("boxes %v, root %+v", boxes, d.States[len(d.States)-1][boxes[0]])
	}
	if len(d.States) != 3 || d.States[0][boxes[1]].O != 0 || d.States[1][boxes[1]].O != 1 || d.States[1][boxes[2]].O != 0 {
		t.Error("one level per step")
	}
}

func TestStorymap(t *testing.T) {
	d, probs := ParseDiagram("storymap", nil, "## Find | Pay\nSearch · Filters | Card · Wallet\nrelease MVP: Search, Card, Ghost")
	if len(probs) != 1 || !strings.Contains(probs[0], `"ghost" is not a story`) {
		t.Errorf("problems %q", probs)
	}
	stories := elsOf(d, "si-dg-story")
	if len(stories) != 4 || len(d.Steps) != 2 || d.Steps[0] != "release MVP" || d.Steps[1] != "the rest" {
		t.Fatalf("stories %v steps %q", stories, d.Steps)
	}
	if d.States[1][stories[0]].O != 1 || d.States[1][stories[1]].O != 0 || d.States[2][stories[1]].O != 1 {
		t.Error("the MVP slice first, the rest after")
	}
}
