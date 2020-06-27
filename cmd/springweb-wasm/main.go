package main

import (
	"math"
	"syscall/js"
	"time"

	"github.com/biotty/springweb"
)

const (
	defaultK          = 1e-9
	armKFactor        = 1e+3
	minK              = defaultK * .15
	maxK              = defaultK * 5
	defaultMass       = 5e7
	minMass           = defaultMass * .15
	maxMass           = defaultMass * 5
	sizeFactor        = 5e-2
	sizeButtonClick   = 5
	voidColor         = "#ffd"
	barColor          = "#bd3"
	buttonColor       = "#451"
	dotColor          = "#42d"
	lineColor         = "#620"
	selectedDotColor  = "#87e"
	selectedLineColor = "#f61"
)

func (a *anim) newDot(x, y float64) {
	m := a.lastMass()
	r := a.dotRadius(m)
	a.dots[a.nDots] = springweb.NewNode(x, y, r, m)
	a.nDots++
}

func (a *anim) newLine(i, j int) {
	k := a.lastK()
	a.dots[i].NewSpring(&a.dots[j], k, armKFactor*k)
}

func (a *anim) findDot(x, y float64) int {
	outsideAllow := 1.5
	if !a.running {
		outsideAllow = 1.1
	}
	for i := 0; i < a.nDots; i++ {
		d := a.dots[i]
		r := a.dotRadius(d.M) * outsideAllow
		if math.Pow(x-d.X, 2)+math.Pow(y-d.Y, 2) <= math.Pow(r, 2) {
			return i
		}
	}
	return -1
}

type anim struct {
	width, height, dotSize float64
	dots, resetNodes       []springweb.Node
	nDots                  int
	selectedDot            int
	dragging               bool
	ctx                    js.Value
	callback               js.Func
	lastCall               time.Time
	deltaT                 float64
	running                bool
}

func (a *anim) buttonHeight() float64 {
	return a.dotSize * 2
}

func (a *anim) buttonRight(i int) float64 {
	return float64(i+1) * a.width / 3
}

func (a *anim) dotRadius(mass float64) float64 {
	return a.dotSize * math.Sqrt(mass/defaultMass)
}

func (a *anim) lineWidth(k float64) float64 {
	return a.dotSize * k / (2 * defaultK)
}

func drawTriangle(ctx js.Value, i int, x, y float64) {
	var px, py []float64
	if i == 0 {
		px = []float64{x - y/2, x + y/2, x - y/2}
		py = []float64{0, y / 2, y}
	} else {
		px = []float64{x - y/2, x + y/2, x}
		if i == 1 {
			py = []float64{y / 2, y / 2, y}
		} else {
			py = []float64{y / 2, y / 2, 0}
		}
	}
	ctx.Call("beginPath")
	ctx.Call("moveTo", px[0], py[0])
	ctx.Call("lineTo", px[1], py[1])
	ctx.Call("lineTo", px[2], py[2])
	ctx.Call("fill")
}

func (a *anim) drawBar() {
	a.ctx.Set("fillStyle", barColor)
	a.ctx.Call("fillRect", 0, 0, a.width, a.buttonHeight())
	a.ctx.Set("fillStyle", buttonColor)
	left := 0.0
	for i := 0; i < 3; i++ {
		right := a.buttonRight(i)
		center := .5 * (left + right)
		if i == 0 && a.running {
			y := a.buttonHeight()
			a.ctx.Call("fillRect", center-y/2, y/4, y/2, y/2)
			a.ctx.Call("fillRect", center+y/4, y/4, y/2, y/2)
		} else {
			drawTriangle(a.ctx, i, center, a.buttonHeight())
		}
		left = right
	}
}

func (a *anim) clear() {
	a.ctx.Set("fillStyle", voidColor)
	a.ctx.Call("fillRect", 0, a.buttonHeight(), a.width, a.height)
	a.drawBar()
}

func (a *anim) drawDot(i int) {
	d := a.dots[i]
	a.ctx.Call("beginPath")
	a.ctx.Call("arc", d.X, d.Y, a.dotRadius(d.M), 0, math.Pi*2)
	a.ctx.Call("fill")
	a.ctx.Call("closePath")
}

func (a *anim) drawLineTo(i int, x, y, k float64) {
	d := a.dots[i]
	a.ctx.Set("lineWidth", a.lineWidth(k))
	a.ctx.Call("beginPath")
	a.ctx.Call("moveTo", d.X, d.Y)
	a.ctx.Call("lineTo", x, y)
	a.ctx.Call("stroke")
}

func (a *anim) drawWeb() {
	for i := 0; i < a.nDots; i++ {
		from := a.dots[i]
		if i == a.selectedDot && !a.running {
			a.ctx.Set("strokeStyle", selectedLineColor)
		} else {
			a.ctx.Set("strokeStyle", lineColor)
		}
		for _, s := range from.Springs {
			a.drawLineTo(i, s.To.X, s.To.Y, s.K)
		}
	}
	for i := 0; i < a.nDots; i++ {
		if i == a.selectedDot {
			a.ctx.Set("fillStyle", selectedDotColor)
		} else {
			a.ctx.Set("fillStyle", dotColor)
		}
		a.drawDot(i)
	}
}

func (a *anim) borderStep() {
	const bounceFactor float64 = -.1
	for i := 0; i < a.nDots; i++ {
		d := &a.dots[i]
		if d.VelocityX < 0 && d.X < 0 {
			d.VelocityX *= bounceFactor
			d.X = 0
		}
		if d.VelocityY < 0 && d.Y < a.buttonHeight() {
			d.VelocityY *= bounceFactor
			d.Y = a.buttonHeight()
		}
		if d.VelocityX > 0 && d.X > a.width {
			d.VelocityX *= bounceFactor
			d.X = a.width
		}
		if d.VelocityY > 0 && d.Y > a.height {
			d.VelocityY *= bounceFactor
			d.Y = a.height
		}
	}
}

func newAnim(width, height, dotSize float64, nNodes int) *anim {
	doc := js.Global().Get("document")
	elem := doc.Call("createElement", "canvas")
	elem.Set("width", width)
	elem.Set("height", height)
	doc.Get("body").Call("appendChild", elem)
	ctx := elem.Call("getContext", "2d")
	a := anim{width, height, dotSize,
		make([]springweb.Node, nNodes), nil, 0, 0, false,
		ctx, js.Func{}, time.Time{}, 0, false}
	a.clear()
	a.callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !a.running {
			return nil
		}
		t := time.Now()
		if a.running {
			x, y := a.draggedDotPosition()
			a.deltaT = float64(t.Sub(a.lastCall))
			springweb.Step(a.dots[:a.nDots], a.deltaT)
			a.borderStep()
			a.positionDraggedDot(x, y)
			a.clear()
			a.drawWeb()
		}
		a.lastCall = t
		js.Global().Call("requestAnimationFrame", a.callback)
		return nil
	})
	return &a
}

func (a *anim) toggleRunEdit() {
	a.running = !a.running
	if a.running {
		a.resetNodes = make([]springweb.Node, a.nDots)
		copy(a.resetNodes, a.dots)
		springweb.StepsPrepare(a.dots[:a.nDots])
		a.lastCall = time.Now()
		js.Global().Call("requestAnimationFrame", a.callback)
	} else {
		copy(a.dots, a.resetNodes)
		a.selectedDot = a.nDots - 1
		a.clear()
		a.drawWeb()
	}
}

func (a *anim) editClickVoid(x, y float64) {
	if a.nDots == len(a.dots) {
		return
	}
	a.selectedDot = a.nDots
	a.newDot(x, y)
	a.drawWeb()
}

func (a *anim) editClickDot(i int) {
	j := a.nDots - 1
	d := &a.dots[j]
	if i == j {
		a.nDots--
		a.clear()
		if j != 0 {
			a.selectedDot = j - 1
			a.drawWeb()
		}
		return
	}
	for k, _ := range d.Springs {
		if d.Springs[k].To == &a.dots[i] {
			a.clear()
			a.drawWeb()
			return
		}
	}
	a.newLine(j, i)
	a.drawWeb()
}

func (a *anim) lastK() float64 {
	for i := a.nDots - 1; i >= 0; i-- {
		s := a.dots[i].Springs
		n := len(s)
		if n != 0 {
			return s[n-1].K
		}
	}
	return defaultK
}

func (a *anim) lastMass() float64 {
	if a.nDots == 0 {
		return defaultMass
	}
	return a.dots[a.nDots-1].M
}

func (a *anim) clickRunning(x, y float64) {
	if i := a.findDot(x, y); i >= 0 {
		a.selectedDot = i
	}
	a.dragging = true
	a.positionDraggedDot(x, y)
}

func (a *anim) positionDraggedDot(x, y float64) {
	if a.dragging {
		node := &a.dots[a.selectedDot]
		if a.deltaT > 0 {
			node.VelocityX = (x - node.X) / a.deltaT
			node.VelocityY = (y - node.Y) / a.deltaT
		}
		node.X = x
		node.Y = y
	}
}

func (a *anim) draggedDotPosition() (x, y float64) {
	if a.dragging {
		node := a.dots[a.selectedDot]
		x = node.X
		y = node.Y
	}
	return
}

func (a *anim) dotSelect(z float64) {
	if z > 0 {
		a.selectedDot++
		if a.selectedDot == a.nDots {
			a.selectedDot = 0
		}
	} else {
		if a.selectedDot == 0 {
			a.selectedDot = a.nDots
		}
		a.selectedDot--
	}
}

func (a *anim) sizeCurrent(z float64) {
	j := a.nDots - 1
	d := a.dots[j]
	n := len(d.Springs)
	if n != 0 {
		s := &d.Springs[n-1]
		k := s.K * (1 - z*sizeFactor)
		if k >= minK && k <= maxK {
			s.K = k
		}
	} else {
		d := &a.dots[j]
		w := d.M / (1 + z*sizeFactor)
		if w >= minMass && w <= maxMass {
			d.M = w
		}
	}
	a.clear()
	a.drawWeb()
}

func (a *anim) upDown(z float64) {
	if a.running {
		a.dotSelect(z)
	} else {
		a.sizeCurrent(z)
	}
}

func (a *anim) clickButton(x float64) {
	if x < a.buttonRight(0) {
		a.toggleRunEdit()
	} else if x < a.buttonRight(1) {
		a.upDown(+sizeButtonClick)
	} else {
		a.upDown(-sizeButtonClick)
	}
}

func (a *anim) click(event js.Value) {
	x := event.Get("clientX").Float()
	y := event.Get("clientY").Float()
	if y < a.buttonHeight() {
		a.clickButton(x)
		return
	}

	if a.running {
		a.clickRunning(x, y)
		return
	}

	i := a.findDot(x, y)
	if i >= 0 {
		a.editClickDot(i)
		return
	}

	a.editClickVoid(x, y)
}

func (a *anim) wheel(event js.Value) {
	z := event.Get("deltaY").Float()
	a.upDown(z)
}

func (a *anim) pointerRelease(event js.Value) {
	a.dragging = false
}

func (a *anim) pointerMove(event js.Value) {
	if !a.dragging {
		return
	}
	x := event.Get("clientX").Float()
	y := event.Get("clientY").Float()
	a.positionDraggedDot(x, y)
}

func main() {
	height := js.Global().Get("innerHeight").Float() - 24
	width := js.Global().Get("innerWidth").Float() - 24
	a := newAnim(width, height, height*0.05, 128)
	eventHandlers := map[string]func(js.Value){
		"pointerdown": a.click,
		"wheel":       a.wheel,
		"pointerup":   a.pointerRelease,
		"pointermove": a.pointerMove,
	}
	for eventName, f := range eventHandlers {
		handler := f
		js.Global().Call("addEventListener", eventName,
			js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				handler(args[0])
				return nil
			}))
	}

	<-make(chan bool)
}

func log(args ...interface{}) {
	js.Global().Get("console").Call("log", args...)
}
