package deck

import (
	"strings"
	"testing"
)

func sceneOf(t *testing.T, body string) *Scene {
	t.Helper()
	sc, probs := ParseScene(map[string]string{}, body)
	if len(probs) > 0 {
		t.Fatalf("problems: %v", probs)
	}
	return sc
}

// obj finds the first object of a kind (and name, if given).
func obj(sc *Scene, kind, name string) int {
	for i, o := range sc.Objs {
		if o.Kind == kind && (name == "" || strings.EqualFold(o.Name, name)) {
			return i
		}
	}
	return -1
}

func last(sc *Scene) []SceneState { return sc.States[len(sc.States)-1] }

func TestSceneVerbs(t *testing.T) {
	cases := []struct {
		name, body string
		check      func(t *testing.T, sc *Scene)
	}{
		{"flies", "box Shop\nbox Customer\nparcel flies Shop -> Customer · arc", func(t *testing.T, sc *Scene) {
			p, c := obj(sc, "parcel", ""), obj(sc, "box", "Customer")
			end := last(sc)
			if sc.States[0][p].V || !end[p].V || end[p].X != end[c].X || end[p].Y >= end[c].Y {
				t.Errorf("the parcel ends on Customer: %+v", end[p])
			}
			if tr := obj(sc, "trail", ""); tr < 0 || !strings.Contains(sc.Objs[tr].D, " C") {
				t.Error("an arc trail")
			}
		}},
		{"flies straight from where it is", "box A\nbox B\nbox C\ntoken flies A → B\ntoken flies → C · straight", func(t *testing.T, sc *Scene) {
			if !sc.Steps[1].Straight || sc.Steps[1].From[0] != sc.States[1][obj(sc, "token", "")].X {
				t.Errorf("the second flight starts where the first ended: %+v", sc.Steps[1])
			}
			if tr := sc.Steps[1].Objs[1]; !strings.Contains(sc.Objs[tr].D, " L") || last(sc)[sc.Steps[0].Objs[1]].V {
				t.Error("a straight trail; the first trail is gone")
			}
		}},
		{"hops", "row Shop, Payments, Warehouse, Courier\ntoken hops Shop → Payments → Warehouse → Courier", func(t *testing.T, sc *Scene) {
			if len(sc.Steps[0].Pts) != 4 || len(sc.Steps[0].Lights) != 4 {
				t.Fatalf("four landings: %+v", sc.Steps[0])
			}
			for _, b := range sc.Steps[0].Lights {
				if !last(sc)[b].On || sc.States[0][b].On {
					t.Error("every block lights")
				}
			}
			if a, b := last(sc)[0].X, last(sc)[3].X; a >= b {
				t.Error("one row, left to right")
			}
		}},
		{"morphs", "box Order\nOrder morphs → Event circle → Message (diamond)\nMessage morphs → Note", func(t *testing.T, sc *Scene) {
			st := sc.Steps[0].Stages
			if len(st) != 2 || st[0].L != "Event" || st[0].R != st[0].W/2 || st[1].L != "Message" || st[1].A != 45 {
				t.Errorf("two stages: %+v", st)
			}
			if l := last(sc)[0]; l.L != "Note" || l.A != 45 {
				t.Errorf("the new name finds it; the shape stays: %+v", l)
			}
		}},
		{"splits", "box Order\nbox Berlin\nbox Lyon\nbox Milan\nOrder splits → Berlin, Lyon, Milan", func(t *testing.T, sc *Scene) {
			if len(sc.Steps[0].Objs) != 3 {
				t.Fatal("three parcels")
			}
			b, l := last(sc)[obj(sc, "box", "Berlin")], last(sc)[obj(sc, "box", "Lyon")]
			if b.X != l.X || b.Y >= l.Y {
				t.Errorf("the targets stack in a column: %+v %+v", b, l)
			}
			p := last(sc)[sc.Steps[0].Objs[0]]
			if !p.V || p.X >= b.X {
				t.Errorf("a parcel waits at the left of its block: %+v", p)
			}
		}},
		{"merge", "box Delivery\nparcels merge → Delivery", func(t *testing.T, sc *Scene) {
			o := sc.Steps[0].Objs
			if len(o) != 3 || !sc.States[0][o[0]].V || last(sc)[o[0]].V || !last(sc)[0].On {
				t.Errorf("three parcels wait, then fuse into a lit Delivery: %+v", o)
			}
		}},
		{"swap", `box "Stock drift"` + "\nbox Courier\nswap Stock drift <-> Courier", func(t *testing.T, sc *Scene) {
			if a, b := sc.States[0][0].X, last(sc)[1].X; a != b {
				t.Error("they trade places")
			}
		}},
		{"assemble and scatter", "tiles assemble 3×3\ntiles scatter", func(t *testing.T, sc *Scene) {
			if len(sc.Steps[0].Objs) != 9 || !sc.States[1][sc.Steps[0].Objs[0]].V || last(sc)[sc.Steps[0].Objs[0]].V {
				t.Errorf("nine tiles come and go: %+v", sc.Steps)
			}
		}},
		{"a block scatters", "box Plan\nPlan scatters", func(t *testing.T, sc *Scene) {
			if last(sc)[0].V || len(sc.Steps[0].Objs) != 8 {
				t.Errorf("the block is gone, in 8 pieces (%d)", len(sc.Steps[0].Objs))
			}
		}},
		{"queue", "queue orders · consumer takes 3\nConsumer takes 1", func(t *testing.T, sc *Scene) {
			q, c := obj(sc, "queue", "orders"), obj(sc, "box", "consumer")
			if q < 0 || c < 0 || last(sc)[c].L != "Consumer" || last(sc)[c].X <= last(sc)[q].X {
				t.Fatalf("a lane, its consumer after it: %v %v", q, c)
			}
			s := sc.Steps[0]
			if len(s.Fill) != 5 || s.Take != 3 || len(s.Lane) != 5 {
				t.Errorf("five join, three are taken: %+v", s)
			}
			n := 0
			for _, m := range s.Fill {
				if last(sc)[m].V {
					n++
				}
			}
			if n != 1 {
				t.Errorf("one left in the queue, %d", n)
			}
		}},
		{"fans out", "box Exchange\ncolumn Email, Stock, Analytics\nExchange fans out → Email, Stock, Analytics", func(t *testing.T, sc *Scene) {
			if len(sc.Steps[0].Objs) != 3 || sc.Objs[sc.Steps[0].Objs[0]].Kind != "message" || !last(sc)[0].On {
				t.Errorf("three copies: %+v", sc.Steps[0])
			}
		}},
		{"opens", "box Shop\nbox Warehouse\nWarehouse opens → Pick, Pack, Ship", func(t *testing.T, sc *Scene) {
			w, pick := obj(sc, "box", "Warehouse"), obj(sc, "box", "Pick")
			if sc.States[0][pick].V || !last(sc)[pick].V || last(sc)[obj(sc, "box", "Shop")].V || last(sc)[w].W < 200 {
				t.Errorf("the block grows over the stage, its parts show: %+v", last(sc)[w])
			}
		}},
		{"route", "row Depot, Stop 1, Stop 2, Stop 3\nroute Depot → Stop 1 → Stop 2 → Stop 3", func(t *testing.T, sc *Scene) {
			s := sc.Steps[0]
			if len(s.Stops) != 3 || s.Stops[2] != 1 || len(s.Lights) != 3 || !last(sc)[s.Lights[0]].V {
				t.Errorf("three pins along the way: %+v", s)
			}
		}},
		{"drops into", "box Locker 12\nparcel drops into Locker 12", func(t *testing.T, sc *Scene) {
			if p := last(sc)[1]; p.X != last(sc)[0].X || p.Y != last(sc)[0].Y {
				t.Errorf("it lands in the locker: %+v", p)
			}
		}},
		{"at places", "box A at 10,50\nbox B at 90,50", func(t *testing.T, sc *Scene) {
			if a := last(sc)[0]; a.X != 32 || a.Y != 90 {
				t.Errorf("percent of the stage: %+v", a)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.check(t, sceneOf(t, c.body)) })
	}
}

func TestSceneProblems(t *testing.T) {
	for _, c := range []struct{ body, want string }{
		{"box Shop\nparcel leaps Shop → X", "scene line 2: no verb"},
		{"box Shop\nparcel flies Shop → Custmer", `scene line 2: unknown block "Custmer" (blocks: Shop)`},
		{"box A\nbox B\nparcel flies A → B · wobbly", `scene line 3: unknown option "wobbly"`},
		{"box A\nA flies A → A", "scene line 2: \"A\" is a block"},
		{"tiles assemble 3 by 3", "scene line 1: write tiles assemble 3×3"},
		{"box A\nswap A B", "scene line 2: write swap A ↔ B"},
		{"Billing takes 2", "scene line 1: Billing takes from a queue"},
		{"route Depot", "scene line 1: a route goes through two"},
	} {
		_, probs := ParseScene(map[string]string{}, c.body)
		if !strings.Contains(strings.Join(probs, "\n"), c.want) {
			t.Errorf("%q: want %q, got %v", c.body, c.want, probs)
		}
	}
}
