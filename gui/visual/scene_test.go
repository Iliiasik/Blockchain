package visual

import (
	"Blockchain/resources"
	"fmt"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func near(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func TestVectorMath(t *testing.T) {
	a, b := vec3{1, 2, 3}, vec3{4, 5, 6}
	if a.add(b) != (vec3{5, 7, 9}) || b.sub(a) != (vec3{3, 3, 3}) || a.mul(2) != (vec3{2, 4, 6}) {
		t.Fatal("basic vector operations are wrong")
	}
	if a.dot(b) != 32 {
		t.Fatal("dot product is wrong")
	}
	if (vec3{1, 0, 0}).cross(vec3{0, 1, 0}) != (vec3{0, 0, 1}) {
		t.Fatal("cross product is wrong")
	}
	if !near((vec3{3, 4, 0}).unit().length(), 1) || (vec3{}).unit() != (vec3{}) {
		t.Fatal("unit vector is wrong")
	}

	start, end := vec3{0, 0, 0}, vec3{4, 0, 0}
	path := arc(start, end, 2)
	if path[0] != start || path[len(path)-1] != end || len(path) != arcSteps+1 {
		t.Fatal("arc must start and end at its anchors")
	}
	if !near(path[arcSteps/2].y, 1) {
		t.Fatalf("arc peak = %v, want half of the lift", path[arcSteps/2].y)
	}
	if line := straight(start, end, 4); line[2] != (vec3{2, 0, 0}) {
		t.Fatal("straight line is not evenly spaced")
	}
}

func TestCameraAndProjection(t *testing.T) {
	c := camera{yaw: 0.3, pitch: 0.6, dist: 10, target: vec3{1, 2, 3}}
	if !near(c.eye().sub(c.target).length(), 10) {
		t.Fatal("eye is not at the requested distance")
	}

	s := &Scene{cam: c, size: fyne.NewSize(800, 600)}
	v := s.makeView()
	center := v.toCam(c.target)
	if !near(center.x, 0) || !near(center.y, 0) || !near(center.z, 10) {
		t.Fatalf("target in camera space = %+v", center)
	}
	if p := v.screen(center); !near(float64(p.X), 400) || !near(float64(p.Y), 300) {
		t.Fatalf("target projects to %v, want the view center", p)
	}

	behind := c.eye().add(c.eye().sub(c.target))
	if _, ok := v.polygon([]vec3{behind, c.target, c.target}); ok {
		t.Fatal("a face behind the camera must be skipped")
	}
	if _, _, _, ok := v.segment(behind, behind.add(vec3{1, 0, 0})); ok {
		t.Fatal("a segment fully behind the camera must be skipped")
	}
	a, _, depth, ok := v.segment(behind, c.target)
	if !ok || depth <= 0 || math.IsInf(float64(a.X), 0) {
		t.Fatal("a segment crossing the near plane must be clipped, not dropped")
	}

	half := blendCamera(camera{yaw: 3, dist: 10}, camera{yaw: -3, dist: 20}, 0.5)
	if math.Abs(half.yaw) < 3 || !near(half.dist, 15) {
		t.Fatalf("blend must take the short way around: %+v", half)
	}
	if blendCamera(c, camera{}, 0) != c {
		t.Fatal("blend at zero must keep the start")
	}
}

func TestScreenHelpers(t *testing.T) {
	square := []fyne.Position{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	if !inside(fyne.NewPos(5, 5), square) || inside(fyne.NewPos(15, 5), square) {
		t.Fatal("point in polygon test is wrong")
	}
	lo, hi := bounds(square)
	if lo != fyne.NewPos(0, 0) || hi != fyne.NewPos(10, 10) {
		t.Fatal("bounds are wrong")
	}
	if arrowHead(fyne.NewPos(1, 1), fyne.NewPos(1, 1), 8) != nil {
		t.Fatal("a zero length segment has no direction")
	}
	if head := arrowHead(fyne.NewPos(0, 0), fyne.NewPos(10, 0), 8); len(head) != 4 || head[0] != fyne.NewPos(10, 0) {
		t.Fatal("arrow head must point at the end of the segment")
	}

	base := color.NRGBA{R: 100, G: 200, B: 50, A: 200}
	if got := withAlpha(base, 0.5).(color.NRGBA); got.A != 100 || got.R != 100 {
		t.Fatalf("withAlpha = %+v", got)
	}
	if got := shade(base, 2).(color.NRGBA); got.R != 200 || got.G != 255 || got.A != 200 {
		t.Fatalf("shade = %+v", got)
	}
	if plural(1, "block") != "1 block" || plural(3, "block") != "3 blocks" {
		t.Fatal("plural is wrong")
	}
	if easeInOut(0) != 0 || easeInOut(1) != 1 || !near(easeInOut(0.5), 0.5) {
		t.Fatal("easing must be symmetric and anchored")
	}
}

func TestArrangePlacesEveryObject(t *testing.T) {
	m := demoModel(false)
	p := arrange(m, false, nil)

	for id := range m.txs {
		if _, ok := p.anchors[txRef(id)]; !ok {
			t.Errorf("transaction %s has no position", id[:6])
		}
	}
	for _, w := range m.wallets {
		if _, ok := p.anchors[walletRef(w.address)]; !ok {
			t.Errorf("wallet %s has no position", w.name)
		}
	}

	kinds := map[linkKind]int{}
	for _, l := range p.links {
		kinds[l.kind]++
		if len(l.pts) < 2 {
			t.Fatal("a link needs at least two points")
		}
	}
	if kinds[linkHash] != len(m.blocks)-1 {
		t.Errorf("hash links = %d, want %d", kinds[linkHash], len(m.blocks)-1)
	}
	if kinds[linkSpend] != 2 || kinds[linkPending] != 2 {
		t.Errorf("spend links = %d, pending links = %d", kinds[linkSpend], kinds[linkPending])
	}
	if kinds[linkOwn] != 3 {
		t.Errorf("ownership links = %d, want one per unspent output", kinds[linkOwn])
	}
	if kinds[linkMine] != 0 || p.nonce >= 0 {
		t.Error("no mining links without mining")
	}

	for i, members := range p.members {
		for _, idx := range members {
			if p.solids[idx].group != i {
				t.Fatal("solid listed in the wrong group")
			}
		}
	}

	busy := arrange(m, true, nil)
	ghosts := 0
	for _, s := range busy.solids {
		if s.ghost {
			ghosts++
		}
	}
	if ghosts != 1 || busy.nonce < 0 || len(busy.links) <= len(p.links) {
		t.Fatal("mining must add a ghost block, a nonce label and mining links")
	}
}

func TestArrangeLimitsLongChains(t *testing.T) {
	m := newModel()
	for h := 0; h < visibleBlocks+5; h++ {
		cb := &txn{id: fmt.Sprintf("cb%d", h), block: h, coinbase: true, outputs: []output{{value: 10, owner: "w"}}}
		m.blocks = append(m.blocks, &blockInfo{height: h, hash: "0000", txs: []*txn{cb}})
		m.add(cb)
	}
	m.wallets = []*walletInfo{{address: "w", name: "w", balance: 350}}
	m.link()

	p := arrange(m, false, nil)
	if _, ok := p.anchors[blockRef(4)]; ok {
		t.Fatal("blocks before the visible window must not be placed")
	}
	if _, ok := p.anchors[blockRef(5)]; !ok {
		t.Fatal("first visible block is missing")
	}
	found := false
	for _, tg := range p.tags {
		if tg.text == "5 earlier" {
			found = true
		}
	}
	if !found {
		t.Fatal("hidden blocks must be summarised")
	}
}

func themedApp(t *testing.T) {
	t.Helper()
	test.NewTempApp(t).Settings().SetTheme(&resources.Theme{Dark: true})
}

func newTestScene(t *testing.T, m *model) (*Scene, fyne.Window) {
	t.Helper()
	themedApp(t)
	s := NewScene()
	w := test.NewWindow(s)
	t.Cleanup(w.Close)
	w.Resize(fyne.NewSize(900, 560))
	s.SetModel(m, false)
	s.Refresh()
	return s, w
}

func screenOf(s *Scene, r ref) fyne.Position {
	v := s.makeView()
	at, _ := s.anchor(r)
	return v.screen(v.toCam(at.sub(vec3{0, 0.05, 0})))
}

func countObjects[T any](s *Scene) int {
	n := 0
	for _, o := range s.world.Objects {
		if _, ok := o.(T); ok {
			n++
		}
	}
	return n
}

func TestSceneRendersAndPicks(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))
	if len(s.world.Objects) == 0 || len(s.hits) == 0 {
		t.Fatal("scene rendered nothing")
	}
	if !s.framed {
		t.Fatal("first model must frame the camera")
	}

	var picked []ref
	s.OnSelect = func(r ref) { picked = append(picked, r) }

	target := txRef(demoAlice)
	s.Tapped(&fyne.PointEvent{Position: screenOf(s, target)})
	if s.selection() != target || len(picked) != 1 {
		t.Fatalf("tap selected %+v", s.selection())
	}
	if !s.focus[txRef(demoCoinbase[0])] || s.focus[txRef(demoCoinbase[2])] {
		t.Fatal("focus must follow the selected transaction's relations")
	}

	s.Tapped(&fyne.PointEvent{Position: screenOf(s, target)})
	if s.selection().kind != refNone || s.focus != nil {
		t.Fatal("tapping the selection again must clear it")
	}

	s.Select(walletRef("bob"))
	s.Tapped(&fyne.PointEvent{Position: fyne.NewPos(2, 2)})
	if s.selection().kind != refNone {
		t.Fatal("tapping empty space must clear the selection")
	}

	s.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: screenOf(s, walletRef("miner"))}})
	if s.hover != walletRef("miner") || s.Cursor() != desktop.PointerCursor {
		t.Fatal("hovering a wallet must highlight it")
	}
	if s.describe(s.hover) != "Wallet Miner: 21 coins" {
		t.Fatalf("tooltip = %q", s.describe(s.hover))
	}
	s.MouseOut()
	if s.hover.kind != refNone || s.Cursor() != desktop.DefaultCursor {
		t.Fatal("leaving the scene must clear the hover")
	}
}

func TestSceneCameraControls(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))
	start := s.cam

	s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(50, 1000)})
	if s.cam.yaw == start.yaw || s.cam.pitch > 1.5 || !s.manual {
		t.Fatal("dragging must rotate and keep the pitch in range")
	}
	s.DragEnd()

	s.MouseDown(&desktop.MouseEvent{Modifier: fyne.KeyModifierShift, Button: desktop.MouseButtonPrimary})
	before := s.cam.target
	s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(40, 0)})
	if s.cam.target == before {
		t.Fatal("shift dragging must pan")
	}
	s.DragEnd()

	s.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonSecondary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(10, 10)}})
	before = s.cam.target
	s.MouseMoved(&desktop.MouseEvent{Button: desktop.MouseButtonSecondary, PointEvent: fyne.PointEvent{Position: fyne.NewPos(30, 10)}})
	if s.cam.target == before {
		t.Fatal("right dragging must pan")
	}
	s.MouseUp(&desktop.MouseEvent{Button: desktop.MouseButtonSecondary})

	dist := s.cam.dist
	s.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 200)})
	if s.cam.dist >= dist {
		t.Fatal("scrolling up must zoom in")
	}
	for range 50 {
		s.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -2000)})
	}
	if s.cam.dist > 160 {
		t.Fatal("zoom must be limited")
	}

	s.ResetView()
	if s.manual || s.tween == nil {
		t.Fatal("reset must animate back to the framed view")
	}
	s.LookAt(start, false)
	if s.cam != start || s.tween != nil {
		t.Fatal("LookAt without animation must jump")
	}
}

func TestSceneLinkTogglesAndFocus(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))
	lines := countObjects[*canvas.Line](s)

	for _, kind := range []linkKind{linkHash, linkSpend, linkOwn} {
		s.SetLinkVisible(kind, false)
	}
	if fewer := countObjects[*canvas.Line](s); fewer >= lines {
		t.Fatalf("hiding links kept %d of %d lines", fewer, lines)
	}

	s.SetFocus(txRef(demoBob), txRef(demoAlice))
	if s.selection().kind != refNone || !s.focus[txRef(demoBob)] {
		t.Fatal("SetFocus must highlight without selecting")
	}
	if s.linkAlpha(link{kind: linkSpend, a: txRef(demoAlice), b: txRef(demoBob)}) != 1 {
		t.Fatal("a focused hidden link must still be drawn")
	}
	if s.linkAlpha(link{kind: linkSpend, a: txRef(demoCoinbase[0]), b: txRef(demoAlice)}) != 0 {
		t.Fatal("an unfocused hidden link must stay hidden")
	}
	if s.emphasis(walletRef("miner")) >= 1 || s.emphasis(txRef(demoBob)) != 1 {
		t.Fatal("unfocused objects must fade")
	}
	s.SetFocus()
	if s.focus != nil {
		t.Fatal("empty SetFocus must clear the focus")
	}
}

func TestSceneMiningAndFlights(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))

	s.SetMining(true)
	s.SetNonce("nonce 42")
	found := false
	for _, o := range s.world.Objects {
		if text, ok := o.(*canvas.Text); ok && text.Text == "nonce 42" {
			found = true
		}
	}
	if !found {
		t.Fatal("the nonce label is not drawn while mining")
	}
	s.SetMining(false)
	if s.nonce != "" {
		t.Fatal("stopping mining must clear the nonce")
	}

	s.Select(txRef(demoPending))
	s.SetModel(demoModel(true), true)
	if len(s.flights) == 0 {
		t.Fatal("a new block must launch coin flights")
	}
	if s.selection() != txRef(demoPending) {
		t.Fatal("a mined transaction must stay selected")
	}

	s.SetModel(newModel(), false)
	if s.selection().kind != refNone {
		t.Fatal("a vanished selection must be cleared")
	}

	s.SetSpin(true)
	s.SetPaused(true)
	if s.anim != nil {
		t.Fatal("pausing must stop the animation loop")
	}
	s.Stop()
	if s.spin || len(s.flights) != 0 || s.tween != nil {
		t.Fatal("Stop must clear every animation")
	}
}

func pendingDemo(extra int) *model {
	m := demoModel(false)
	for i := 0; i < extra; i++ {
		t := &txn{id: fmt.Sprintf("p%02d", i), block: -1, inputs: []input{{owner: "bob"}}, outputs: []output{{value: 1, owner: "alice"}}}
		m.mempool = append([]*txn{t}, m.mempool...)
		m.add(t)
	}
	m.link()
	return m
}

func TestMempoolSlotsStayPut(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))
	first := s.slots[demoPending]
	before, _ := s.anchor(txRef(demoPending))

	s.SetModel(pendingDemo(3), true)
	after, _ := s.anchor(txRef(demoPending))
	if s.slots[demoPending] != first || after != before {
		t.Fatal("a waiting transaction moved when new ones arrived")
	}
	seen := map[int]bool{}
	for id, slot := range s.slots {
		if seen[slot] {
			t.Fatalf("slot %d is used twice (%s)", slot, id)
		}
		seen[slot] = true
	}

	landing := 0
	for _, f := range s.flights {
		if f.land.kind != refNone {
			landing++
			target, _ := s.anchor(f.land)
			if f.to != target.sub(vec3{0, slotSize / 2, 0}) || f.size != slotSize {
				t.Fatal("a new transaction must fly straight into its own slot")
			}
		}
	}
	if landing != 3 {
		t.Fatalf("%d landing flights, want 3", landing)
	}

	s.render()
	for _, h := range s.hits {
		if h.owner.kind == refTx && h.owner.id == "p00" {
			t.Fatal("a cube still in flight must not be drawn in its slot yet")
		}
	}

	s.SetModel(demoModel(true), true)
	if len(s.slots) != 0 {
		t.Fatal("mined transactions must free their slots")
	}
	for _, f := range s.flights {
		if f.land.kind == refNone && f.to.y < 0.7 {
			t.Fatal("rewards and mined transactions must end above their target")
		}
	}
}

func TestMempoolIsThreeByThree(t *testing.T) {
	markers := func(m *model) int {
		n := 0
		for _, sd := range arrange(m, false, nil).solids {
			if sd.owner.kind == refNone && sd.child && sd.max.y-sd.min.y < 0.05 {
				n++
			}
		}
		return n
	}

	empty := newModel()
	for capacity, want := range map[int]int{1: 1, 2: 2, 5: 5, 9: 9, 50: 9} {
		empty.maxPool = capacity
		if got := markers(empty); got != want {
			t.Errorf("capacity %d shows %d slots, want %d", capacity, got, want)
		}
	}

	small, big := demoModel(false), demoModel(false)
	small.maxPool, big.maxPool = 2, 50
	if arrange(small, false, nil).pool != arrange(big, false, nil).pool {
		t.Fatal("the platform must stay 3 by 3")
	}

	m := pendingDemo(11)
	m.maxPool = 50
	p := arrange(m, false, nil)
	layers := map[float64]int{}
	for _, tx := range m.mempool {
		at, ok := p.anchors[txRef(tx.id)]
		if !ok {
			t.Fatalf("pending %s has no place", tx.id)
		}
		layers[math.Round((at.y-0.12)/slotSize)]++
	}
	if len(layers) != 2 || layers[1] != 9 || layers[2] != 3 {
		t.Fatalf("12 transactions must fill 9 slots and stack 3 on top, got %v", layers)
	}
}

func TestLayoutSitsOnTheGrid(t *testing.T) {
	whole := func(v float64) bool {
		return math.Abs(v-math.Round(v)) < 1e-9
	}
	for _, m := range []*model{demoModel(false), pendingDemo(4)} {
		p := arrange(m, true, nil)
		if len(p.ground) != 3 {
			t.Fatalf("expected chain, pool and wallet zones, got %d", len(p.ground))
		}
		for _, z := range p.ground {
			if !whole(z.min.x) || !whole(z.min.z) || !whole(z.max.x) || !whole(z.max.z) {
				t.Errorf("zone %v..%v is off the grid", z.min, z.max)
			}
		}
		for _, s := range p.solids {
			if s.child {
				continue
			}
			if !whole(s.min.x) || !whole(s.min.z) || !whole(s.max.x) || !whole(s.max.z) {
				t.Errorf("footprint %v..%v is off the grid", s.min, s.max)
			}
		}
		chain, pool, wallets := p.ground[0], p.ground[1], p.ground[2]
		if a, b := pool.min.z-chain.max.z, wallets.min.z-pool.max.z; !near(a, b) || a <= 0 {
			t.Errorf("gaps between zones must match, got %v and %v", a, b)
		}
		for _, g := range p.grid {
			if !whole(g[0].x) || !whole(g[0].z) {
				t.Fatalf("grid line %v is off the unit grid", g)
			}
		}
	}
}

func TestWalletsAndPoolStartWithTheChain(t *testing.T) {
	for _, m := range []*model{demoModel(false), pendingDemo(4)} {
		p := arrange(m, false, nil)
		start := p.anchors[blockRef(0)].x
		if first := p.anchors[walletRef(m.wallets[0].address)].x; first != start {
			t.Errorf("first wallet at %v, first block at %v", first, start)
		}
		pool := p.ground[1]
		if mid := (pool.min.x + pool.max.x) / 2; mid != start {
			t.Errorf("mempool at %v, first block at %v", mid, start)
		}
	}
}

func TestNewBlocksKeepPoolAndWalletsInPlace(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))
	base := func(r ref) fyne.Position {
		at, _ := s.anchor(r)
		v := s.makeView()
		return v.screen(v.toCam(vec3{at.x, 0, at.z}))
	}
	before := base(walletRef("alice"))
	pending, _ := s.anchor(txRef(demoPending))
	pool := s.plan.ground[1]

	s.SetModel(demoModel(true), true)
	if after := base(walletRef("alice")); distance(before, after) > 0.5 {
		t.Fatalf("wallets moved on screen from %v to %v", before, after)
	}
	if s.plan.ground[1] != pool {
		t.Fatal("the mempool must stay where it is")
	}
	want := pending.sub(vec3{0, slotSize / 2, 0})
	landed, _ := s.anchor(txRef(demoPending))
	for _, f := range s.flights {
		if f.to.sub(landed.add(vec3{0, 0.75, 0})).length() < 1e-9 {
			if f.from.sub(want).length() > 1e-9 {
				t.Fatalf("flight starts at %v, want the mempool slot at %v", f.from, want)
			}
			return
		}
	}
	t.Fatal("no flight lands on the mined transaction")
}

func TestChainZoneEndsAtTheTip(t *testing.T) {
	m := demoModel(false)
	tip := arrange(m, false, nil).anchors[blockRef(len(m.blocks)-1)].x
	idle := arrange(m, false, nil).ground[0].max.x
	busy := arrange(m, true, nil).ground[0].max.x
	if !near(idle, tip+blockW/2+zoneMargin) {
		t.Fatalf("idle chain zone ends at %v, want it to close right after the tip at %v", idle, tip)
	}
	if !near(busy, idle+blockPitch) {
		t.Fatalf("mining must reserve exactly one block, got %v", busy-idle)
	}
}

func TestLinksKeepClearOfTheirEnds(t *testing.T) {
	straightLine := trail{pieces: []piece{{a: fyne.NewPos(0, 0), b: fyne.NewPos(100, 0), length: 100, seen: true}}, length: 100}
	segs := straightLine.stroke(false, true, 2)
	if len(segs) != 1 {
		t.Fatalf("a straight visible link must be one line, got %d", len(segs))
	}
	if segs[0][0].X != linkGapStart {
		t.Fatalf("line starts at %v, want a gap of %d", segs[0][0].X, linkGapStart)
	}
	head := straightLine.head(2)
	if head[0].X != 100-linkGapEnd {
		t.Fatalf("arrow tip at %v, want it to stop %d short of the target", head[0].X, linkGapEnd)
	}
	if segs[0][1].X >= head[0].X-headSize(2)*0.5 || segs[0][1].X <= head[0].X-headSize(2) {
		t.Fatalf("the line must end inside the arrow head, ends at %v with the tip at %v", segs[0][1].X, head[0].X)
	}

	dashes := straightLine.stroke(true, false, 2)
	for _, d := range dashes {
		if l := distance(d[0], d[1]); l > dashOn+0.01 {
			t.Fatalf("dash of %v px is longer than %d", l, dashOn)
		}
	}
	if len(dashes) < 7 {
		t.Fatalf("expected an even dash pattern, got %d dashes", len(dashes))
	}

	hidden := straightLine
	hidden.pieces = []piece{{a: fyne.NewPos(0, 0), b: fyne.NewPos(100, 0), length: 100}}
	if hidden.stroke(false, true, 2) != nil || hidden.head(2) != nil {
		t.Fatal("an occluded link must not be drawn")
	}
}

func TestArcsClearObjects(t *testing.T) {
	wall := []solid{{min: vec3{-0.5, 0, -0.5}, max: vec3{0.5, 3, 0.5}}}
	a, b := vec3{-3, 0.5, 0}, vec3{3, 0.5, 0}
	lift := clearLift(wall, a, b, 0.5, 0.1)
	if lift <= 0.5 {
		t.Fatal("an arc through a wall must be lifted")
	}
	c := lerp3(a, b, 0.5)
	c.y += lift
	for i := 0; i <= 24; i++ {
		pt := bezier(a, c, b, float64(i)/24)
		if math.Abs(pt.x) < 0.6 && math.Abs(pt.z) < 0.6 && pt.y < 3.1 {
			t.Fatalf("the lifted arc still crosses the wall at %+v", pt)
		}
	}
	if clearLift(nil, a, b, 0.5, 0.1) != 0.5 {
		t.Fatal("a free path must keep its lift")
	}
}

func TestRayHitsBox(t *testing.T) {
	lo, hi := vec3{-1, -1, -1}, vec3{1, 1, 1}
	if !rayHitsBox(vec3{0, 0, -5}, vec3{0, 0, 5}, lo, hi) {
		t.Fatal("a ray through the box must hit it")
	}
	if rayHitsBox(vec3{0, 5, -5}, vec3{0, 5, 5}, lo, hi) {
		t.Fatal("a ray above the box must miss")
	}
	if rayHitsBox(vec3{0, 0, -5}, vec3{0, 0, -1.5}, lo, hi) {
		t.Fatal("a ray that stops before the box must miss")
	}
	if rayHitsBox(vec3{0, 5, 0}, vec3{0, 1, 0}, lo, hi) {
		t.Fatal("a point lying on the surface is visible from outside")
	}
}

func TestLinksBehindModelsAreHidden(t *testing.T) {
	s, _ := newTestScene(t, demoModel(false))
	v := s.makeView()
	blockers := s.occluders(v, nil)
	slab, _ := s.anchor(blockRef(1))
	inside := slab.sub(vec3{0, blockH / 2, 0})
	if !blockers.hide(v, inside) {
		t.Fatal("a point inside a block must be hidden")
	}
	above := slab.add(vec3{0, 3, 0})
	if blockers.hide(v, above) {
		t.Fatal("a point high above the chain must stay visible")
	}
}

func TestGrownPolygonsOverlap(t *testing.T) {
	square := []fyne.Position{{X: 0, Y: 0}, {X: 10, Y: 0}, {X: 10, Y: 10}, {X: 0, Y: 10}}
	out := grown(square, 1)
	if out[0].X >= 0 || out[2].X <= 10 || out[2].Y <= 10 {
		t.Fatalf("grown polygon = %v", out)
	}
}
