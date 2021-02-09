package logos

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
)

//----------------------------------------
// Buffer

// A Buffer is a buffer area in which to draw.
type Buffer struct {
	Size
	Cells []Cell
}

func NewBuffer(sz Size) *Buffer {
	return &Buffer{
		Size:  sz,
		Cells: make([]Cell, sz.Width*sz.Height),
	}
}

func (bb *Buffer) GetCell(x, y int) *Cell {
	if bb.Width <= x {
		panic(fmt.Sprintf(
			"index x=%d out of bounds, width=%d",
			x, bb.Width))
	}
	if bb.Height <= y {
		panic(fmt.Sprintf(
			"index y=%d out of bounds, height=%d",
			y, bb.Height))
	}
	return &bb.Cells[y*bb.Width+x]
}

func (bb *Buffer) Sprint() string {
	lines := []string{}
	for y := 0; y < bb.Height; y++ {
		parts := []string{}
		for x := 0; x < bb.Width; x++ {
			cell := bb.GetCell(x, y)
			parts = append(parts, cell.Character)
		}
		line := strings.Join(parts, "")
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (bb *Buffer) DrawToScreen(s tcell.Screen) {
	sw, sh := s.Size()
	if bb.Size.Width != sw || bb.Size.Height != sh {
		panic("buffer doesn't match screen size")
	}
	var st tcell.Style = tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.ColorWhite)
	bgst := st.Dim(true).
		Background(tcell.ColorGrey)
	for y := 0; y < sh; y++ {
		for x := 0; x < sw; x++ {
			cell := bb.GetCell(x, y)
			if cell.Width == 0 {
				// For debugging.
				s.SetContent(x, y, '.', nil, bgst)
			} else {
				rz := toRunes(cell.Character)
				st2 := st
				if cell.Foreground.Valid() {
					st2 = st2.Foreground(cell.Foreground)
				}
				if cell.Background.Valid() {
					st2 = st2.Background(cell.Background)
				}
				if cell.GetIsShaded() {
					// XXX some bug when this is true,
					// some cells won't have gray background.
					// why is that?
					// st2 = st2.Dim(true)
					st2 = st2.Background(tcell.ColorGray)
				}
				s.SetContent(x, y, rz[0], rz[1:], st2)
			}
		}
	}
}

//----------------------------------------
// Cell

// A terminal character cell.
type Cell struct {
	Character  string // 1 unicode character, or " ".
	Width      int
	Foreground Color
	Background Color
	StyleFlags
	Elem // reference to element
}

func (cc *Cell) SetCell(oc *Cell) {
	*cc = *oc
}

func (cc *Cell) SetValue(chs string, w int, st Style, el Elem) {
	cc.Character = chs
	cc.Width = w
	cc.Foreground = st.Foreground
	cc.Background = st.Background
	cc.StyleFlags = st.StyleFlags
	cc.Elem = el
}

//----------------------------------------
// View
// analogy: "Buffer:View :: array:slice".

// Offset and Size must be within bounds of *Buffer.
type View struct {
	Base   *Buffer
	Offset Coord // offset within Buffer
	Bounds Size  // total size of slice
}

// offset elements must be 0 or positive.
func (bb *Buffer) NewView(offset Coord) View {
	if !offset.IsNonNegative() {
		panic("should not happen")
	}
	return View{
		Base:   bb,
		Offset: offset,
		Bounds: bb.Size,
	}
}

func (bs View) NewView(offset Coord) View {
	return View{
		Base:   bs.Base,
		Offset: bs.Offset.Add(offset),
		Bounds: bs.Bounds.SubCoord(offset),
	}
}

func (bs View) GetCell(x, y int) *Cell {
	if bs.Bounds.Width <= x {
		panic("should not happen")
	}
	if bs.Bounds.Height <= y {
		panic("should not happen")
	}
	return bs.Base.GetCell(
		bs.Offset.X+x,
		bs.Offset.Y+y,
	)
}

//----------------------------------------
// BufferedView

// A view onto an element.
// Somewhat like a slice onto an array
// (as a view is onto an elem),
// except cells are allocated here.
type BufferedElemView struct {
	Coord
	Size
	Style
	Attrs         // e.g. to focus on a scrollbar
	Base    Elem  // the underlying elem
	Offset  Coord // within elem for pagination
	*Buffer       // view's internal draw screen
}

// Returns a new *BufferedElemView that spans the whole elem.
// If size is zero, the elem is measured first to get the full buffer
// size. The result must still be rendered before drawing.
// The *BufferedElemView inherits the coordinate of the elem,
// and the elem's coord is otherwise ignored.
func NewBufferedElemView(elem Elem, size Size) *BufferedElemView {
	if size.IsZero() {
		size = elem.Measure()
	}
	bpv := &BufferedElemView{
		Size:   size,
		Style:  *elem.GetStyle(), // TODO
		Base:   elem,
		Offset: Coord{0, 0},
		// NOTE: be lazy, size may change.
		// Buffer: NewBuffer(size),
	}
	bpv.SetCoord(elem.GetCoord())
	elem.SetParent(bpv)
	bpv.SetIsDirty(true)
	return bpv
}

func (bpv *BufferedElemView) String() string {
	return fmt.Sprintf("Buffered%v{%v}@%p",
		bpv.Size,
		bpv.Base,
		bpv)
}

func (bpv *BufferedElemView) SetSize(size Size) {
	bpv.Size = size
	bpv.Buffer = nil
	bpv.SetIsDirty(true)
}

// BufferedElemView's size is simply defined by .Size.
func (bpv *BufferedElemView) Measure() Size {
	return bpv.Size
}

// Renders the elem onto the internal buffer.
// Assumes buffered elem view's elem was already rendered.
// TODO: this function could be optimized to reduce
// redundant background cell modifications.
func (bpv *BufferedElemView) Render() (updated bool) {
	if !bpv.GetIsDirty() {
		return
	} else {
		defer bpv.SetIsDirty(false)
	}
	buffer := bpv.Buffer
	if buffer == nil {
		buffer = NewBuffer(bpv.Size)
		bpv.Buffer = buffer
	}
	// First, draw buffer background style.
	if true {
		style := bpv.Style
		for x := 0; x < buffer.Size.Width; x++ {
			for y := 0; y < buffer.Size.Height; y++ {
				cell := buffer.GetCell(x, y)
				cell.Foreground = style.Foreground
				cell.Background = style.Foreground
				cell.StyleFlags = style.StyleFlags
			}
		}
	}
	// Then, render and draw elem.
	bpv.Base.Render()
	bpv.Base.Draw(bpv.Offset, buffer.NewView(Coord{}))
	return true
}

func (bpv *BufferedElemView) Draw(offset Coord, view View) {
	minX, maxX, minY, maxY := computeIntersection(bpv.Size, offset, view.Bounds)
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			bcell := bpv.Buffer.GetCell(x, y)
			vcell := view.GetCell(x-minX, y-minY)
			vcell.SetCell(bcell)
		}
	}
}

func (bpv *BufferedElemView) ProcessEventKey(ev *EventKey) bool {
	// TODO pagination
	return bpv.Base.ProcessEventKey(ev)
}