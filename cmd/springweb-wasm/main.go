package main

import (
	"math"
	"syscall/js"
	"time"

	"github.com/biotty/springweb"
)

const defaultK = 1e-9
const minK = defaultK * .15
const maxK = defaultK * 5
const defaultMass = 5e7
const minMass = defaultMass * .15
const maxMass = defaultMass * 5
const sizeFactor = 5e-2
const sizeButtonClick = 5
const voidColor = "#ccf"
const barColor = "#baf"
const buttonColor = "#70f"
const dotColor = "#a70"
const lineColor = "#050"
const selectedDotColor = "#c83"
const selectedLineColor = "#05f"

var never time.Time = time.Unix(0, 0)

func (a *anim) newDot(x, y float64) {
	a.dots[a.nDots] = springweb.NewNode(x, y, a.lastMass())
	a.nDots++
}

func (a *anim) findDot(x, y float64) int {
	for i := 0; i < a.nDots; i++ {
		d := a.dots[i]
		r := a.dotRadius(d.InvMass) + a.dotSize*.2
		if math.Pow(x-d.X, 2)+math.Pow(y-d.Y, 2) <= math.Pow(r, 2) {
			return i
		}
	}
	return -1
}

type anim struct {
	width, height, dotSize float64
	dots, initialDots      []springweb.Node
	nDots                  int
	selectedDot            int
	ctx                    js.Value
	callback               js.Func
	lastCall               time.Time
	haltAnim               bool
}

func (a *anim) running() bool {
	return a.lastCall != never
}

func (a *anim) stopRunning() {
	a.haltAnim = true
	a.lastCall = never
}

func (a *anim) buttonHeight() float64 {
	return a.dotSize * 2
}

func (a *anim) buttonRight(i int) float64 {
	return float64(i+1) * a.width / 3
}

func (a *anim) dotRadius(invMass float64) float64 {
	return a.dotSize * math.Pow(defaultMass*invMass, -.5)
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
		if i == 0 && a.running() {
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
	a.ctx.Call("arc", d.X, d.Y, a.dotRadius(d.InvMass), 0, math.Pi*2)
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
		if i == a.selectedDot && !a.running() {
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
	const bounceFactor float64 = -.01
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
		make([]springweb.Node, nNodes), nil, 0, 0,
		ctx, js.Func{}, never, false}
	a.clear()
	a.callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if a.haltAnim {
			a.haltAnim = false
			return nil
		}
		t := time.Now()
		if a.running() {
			springweb.Step(a.dots, float64(t.Sub(a.lastCall)))
			a.borderStep()
			a.clear()
			a.drawWeb()
		}
		a.lastCall = t
		js.Global().Call("requestAnimationFrame", a.callback)
		return nil
	})
	return &a
}

func (a *anim) playPause() {
	if !a.running() {
		a.initialDots = make([]springweb.Node, a.nDots)
		copy(a.initialDots, a.dots)
		js.Global().Call("requestAnimationFrame", a.callback)
	} else {
		copy(a.dots, a.initialDots)
		a.stopRunning()
		a.selectedDot = a.nDots - 1
		a.clear()
		a.drawWeb()
	}
}

func (a *anim) clickOnVoid(x, y float64) {
	if a.nDots == len(a.dots) {
		return
	}
	a.selectedDot = a.nDots
	a.newDot(x, y)
	a.drawWeb()
}

func (a *anim) clickOnDot(i int) {
	j := a.nDots - 1
	if i == j {
		a.nDots--
		a.clear()
		if j != 0 {
			a.selectedDot = a.nDots - 1
			a.drawWeb()
		}
		return
	}
	d := &a.dots[j]
	for k, _ := range d.Springs {
		if d.Springs[k].To == &a.dots[i] {
			d.Springs = append(d.Springs[:k], d.Springs[k+1:]...)
			a.clear()
			a.drawWeb()
			return
		}
	}
	d.NewSpring(&a.dots[i], a.lastK())
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
	return 1 / a.dots[a.nDots-1].InvMass
}

func (a *anim) clickRunning(x, y float64) {
	node := &a.dots[a.selectedDot]
	impulse := node.InvMass * defaultMass * defaultK
	node.VelocityX = (x - node.X) * impulse
	node.VelocityY = (y - node.Y) * impulse
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
		w := d.InvMass * (1 + z*sizeFactor)
		if 1/w >= minMass && 1/w <= maxMass {
			d.InvMass = w
		}
	}
	a.clear()
	a.drawWeb()
}

func (a *anim) upDown(z float64) {
	if a.running() {
		a.dotSelect(z)
	} else {
		a.sizeCurrent(z)
	}
}

func (a *anim) clickButton(x float64) {
	log(x)
	if x < a.buttonRight(0) {
		a.playPause()
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

	if a.running() {
		a.clickRunning(x, y)
		return
	}

	i := a.findDot(x, y)
	if i >= 0 {
		a.clickOnDot(i)
		return
	}

	a.clickOnVoid(x, y)
}

func (a *anim) wheel(event js.Value) {
	event.Call("preventDefault")
	z := event.Get("deltaY").Float()
	a.upDown(z)
}

func main() {
	height := js.Global().Get("innerHeight").Float() - 24
	width := js.Global().Get("innerWidth").Float() - 24
	a := newAnim(width, height, height*0.05, 128)
	js.Global().Call("addEventListener", "pointerdown",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			a.click(args[0])
			return nil
		}))

	js.Global().Call("addEventListener", "wheel",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			a.wheel(args[0])
			return nil
		}))

	<-make(chan bool)
}

func log(args ...interface{}) {
	js.Global().Get("console").Call("log", args...)
}
