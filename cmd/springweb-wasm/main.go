package main

import (
	"math"
	"math/rand"
	"syscall/js"
	"time"

	"github.com/biotty/springweb"
)

const (
	defaultK            = 1.  // note: K (and dependent armK) seems depending on dotSize
	armKFactor          = 1e3 //       as the model operates on the distances provided directly
	minK                = defaultK * .15
	maxK                = defaultK * 5
	defaultMass         = 1e-2
	minMass             = defaultMass * .15
	maxMass             = defaultMass * 5
	bounceFactor        = -.65
	gravity             = 3e2  // note: dep. on dotSize.  make independent by multiplying at calc.
	maxWheelForce       = 1.1  //       idem
	maxWheelVelocity    = 1e1  // note: measured in dotSize as unit, so constant not dep. on dotSize
	wheelGyrationFactor = 7e-1 // note: deps as explained
	wheelDriveArmFactor = 3e-1 //       idem .. but maybe not this one, as it is purely angular
	sizeFactor          = 5e-2 //       (but seems the related maxWheelForce is dep on scale ..)
	sizeButtonClick     = 5.
	voidColor           = "#ffd"
	barColor            = "#bd3"
	buttonColor         = "#451"
	dotColor            = "#42d"
	selectedDotColor    = "#87e"
	lineColor           = "rgba(96, 32, 0, 0.2)"
	selectedLineColor   = "rgba(255, 96, 16, 0.5)"
	platformColor       = "rgba(0, 128, 128, 0.5)"
	rightForceColor     = "rgba(0, 0, 255, 0.2)"
	leftForceColor      = "rgba(255, 0, 0, 0.2)"
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
	outsideAllow := 1.7
	if !a.running {
		outsideAllow = 1.3
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

type wheel struct {
	angleVelocity     float64
	angle             float64
	onPlatform        int
}

type platform struct {
	left, right, y float64
}

func (p *platform) border(d *springweb.Node) bool {
	yUpper := d.Y + d.R*.4 // quantity: on the cap on v in springweb move
	yLower := d.Y + d.R
	if d.X > p.left && d.X < p.right && p.y > yUpper && p.y < yLower {
		d.VelocityY *= bounceFactor  // assume: VelocityY > 0 already checked
		d.Y = p.y - d.R  // improve: a little sudden (alt: triangular at right and left)
		return true
	}
	return false
}

type anim struct {
	width, height, dotSize float64
	dots, resetNodes       []springweb.Node
	nDots                  int
	nGassDots int
	selectedDot            int
	dragging               bool
	ctx                    js.Value
	images                 []js.Value
	callback               js.Func
	lastCall               time.Time
	deltaT                 float64
	running                bool
	keyisdown              bool
	viewX float64
	wheelForce             float64
	wheels []wheel
	nWheels                int
	platforms              []platform
	nPlatforms             int
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

func (a *anim) drawControl() {
	color := rightForceColor
	x := a.width * .5
	w := x * a.wheelForce / maxWheelForce
	if w < 0 {
		color = leftForceColor
		w = -w
		x -= w
	}
	a.ctx.Set("fillStyle", color)
	y := a.buttonHeight()
	a.ctx.Call("fillRect", x, y, w, y*.5)
}

func (a *anim) clear() {
	a.ctx.Set("fillStyle", voidColor)
	a.ctx.Call("fillRect", 0, a.buttonHeight(), a.width, a.height)
	a.drawBar()
}

func (a *anim) drawDot(i int) {
	d := a.dots[i]
	if !a.running || i == a.selectedDot {
		r := d.R
		if a.running {
			r *= 1.1
		}
		a.ctx.Call("beginPath")
		a.ctx.Call("arc", d.X - a.viewX, d.Y, r, 0, math.Pi*2)
		a.ctx.Call("fill")
		a.ctx.Call("closePath")
	}
	if a.running {
		img := a.images[0]
		b := d.Angle
		if i < a.nWheels {
			img = a.images[1]
			b += a.wheels[i].angle // alt: =
		}
		a.ctx.Call("save")
		a.ctx.Call("translate", d.X - a.viewX, d.Y)
		a.ctx.Call("rotate", b)
		a.ctx.Call("drawImage", img, -d.R, -d.R, d.R*2, d.R*2)
		a.ctx.Call("restore")
	}
}

func (a *anim) drawLineTo(i int, x, y, k float64) {
	d := a.dots[i]
	a.ctx.Set("lineWidth", a.lineWidth(k))
	a.ctx.Call("beginPath")
	a.ctx.Call("moveTo", d.X - a.viewX, d.Y)
	a.ctx.Call("lineTo", x - a.viewX, y)
	a.ctx.Call("stroke")
}

func (a *anim) drawWeb() {
	a.clear()
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

func (a *anim) drawPlatforms() {
	h := a.dotSize *.5
	a.ctx.Set("strokeStyle", platformColor)
	for i := 0; i < a.nPlatforms; i++ {
		p := a.platforms[i]
		a.ctx.Set("lineWidth", h)
		y := p.y + h * .5
		a.ctx.Call("beginPath")
		a.ctx.Call("moveTo", p.left - a.viewX, y)
		a.ctx.Call("lineTo", p.right - a.viewX, y)
		a.ctx.Call("stroke")
	}
}

func (a *anim) platformsStep() {
	// idea: dots are the car, but also the other "objects", by >= nCar
	for i := 0; i < a.nDots; i++ {
		d := &a.dots[i]
		if i >= a.nWheels && i < a.nGassDots {
			continue  // note: avoid car hooked stuck
		}
		if d.VelocityY <= 0 {
			continue  // note: instead of in per-platform check
		}
		// optimize: if d.VelocityY <= 0 then CONTINUE instead of cond in platform.border
		// optimize: if wheel.OnPlatform then check that platform first
		for j := 0; j < a.nPlatforms; j++ {
			if a.platforms[j].border(&a.dots[i]) {
				if i < a.nWheels {
					a.wheels[i].onPlatform = j
				}
				break
			}
		}
	}
}

func (a *anim) gravityStep() {
	g := gravity
	for i := 0; i < a.nDots; i++ {
		d := &a.dots[i]
		if d.Y < a.height-d.R {
			d.VelocityY += g * a.deltaT
		}
	}
}

func (a *anim) wheelVelocityBelowMax(velocityX float64) bool {
	if a.wheelForce > 0 && velocityX < maxWheelVelocity*a.dotSize {
		return true
	}
	if a.wheelForce < 0 && velocityX > -maxWheelVelocity*a.dotSize {
		return true
	}
	return false
}

func (a *anim) wheelRotation(i int) {
	d := &a.dots[i]
	w := &a.wheels[i]
	j := w.onPlatform
	if j >= 0 {
		if d.Y+d.R*1.1 < a.platforms[j].y {
			w.onPlatform = -1
		}
	}

	if w.onPlatform >= 0 {
		w.angleVelocity = d.VelocityX / d.R
		if a.wheelVelocityBelowMax(d.VelocityX) {
			d.VelocityX += d.R * a.wheelForce * a.deltaT / d.M
			d.Angle -= a.wheelForce * wheelDriveArmFactor
		}
	} else {
		if a.wheelVelocityBelowMax(w.angleVelocity*d.R) {
			w.angleVelocity += a.wheelForce * a.deltaT / (d.M * wheelGyrationFactor)
		}
	}

	w.angle += w.angleVelocity * a.deltaT
}

func (a *anim) viewBorderStep() {
	for i := 0; i < a.nDots; i++ {
		d := &a.dots[i]
		if d.VelocityX < 0 && d.X < a.viewX + d.R {
			d.VelocityX *= bounceFactor
			d.X = a.viewX + d.R
		}
	}
}


func (a *anim) viewScrollStep() {
	if a.nDots == 0 {
		return
	}
	d := &a.dots[0]
	q := a.width * .5
	if d.X > a.viewX + q {
		a.viewX = d.X - q
	}
}

func (a *anim) worldCycle() {
	for i := 1; i < a.nPlatforms; i++ {  // index: 0 is ground
		p := &a.platforms[i]
		if p.right < a.viewX {
			p.left = a.viewX+(1+rand.Float64())*a.width
			p.right = p.left + (1+rand.Float64())*.2*a.width
			p.y = (1+rand.Float64())*.5*a.height  //todo:  up-n-down going
		}
	}
}

func newAnim(width, height, dotSize float64, nNodes int) *anim {
	doc := js.Global().Get("document")
	elem := doc.Call("createElement", "canvas")
	elem.Set("width", width)
	elem.Set("height", height)
	doc.Get("body").Call("appendChild", elem)
	images := []js.Value{
		doc.Call("getElementById", "image0"),
		doc.Call("getElementById", "image1"),
	}
	ctx := elem.Call("getContext", "2d")
	a := anim{width, height, dotSize,
		make([]springweb.Node, nNodes), nil, 0, 0, 0, false,
		ctx, images, js.Func{}, time.Time{}, 0, false, false, 0, 0,
		make([]wheel, 2), 2,
		make([]platform, 3), 3}

	a.platforms[0] = platform{0, 1e9, a.height}  // note: the ground

	a.clear()
	a.callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !a.running {
			return nil
		}
		t := time.Now()
		if a.running {
			x, y := a.draggedDotPosition()
			deltaT := 1e-9 * float64(t.Sub(a.lastCall))
			if a.deltaT == 0 || deltaT < a.deltaT*9 {
				a.deltaT = deltaT
			}
			springweb.Step(a.dots[:a.nDots], a.deltaT)
			for i := 0; i < a.nWheels; i++ {
				if i < a.nDots {
					a.wheelRotation(i)
				}
			}
			a.platformsStep()
			a.gravityStep()
			a.viewBorderStep()
			a.viewScrollStep()
			a.worldCycle()
			a.positionDraggedDot(x, y)
			a.drawWeb()
			a.drawPlatforms()
			a.drawControl()
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
		a.nGassDots = a.nDots
		js.Global().Call("requestAnimationFrame", a.callback)
	} else {
		copy(a.dots, a.resetNodes)
		a.selectedDot = a.nDots - 1
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
		if j != 0 {
			a.selectedDot = j - 1
		}
		return
	}
	for k, _ := range d.Springs {
		if d.Springs[k].To == &a.dots[i] {
			d.Springs = append(d.Springs[:k], d.Springs[k+1:]...)
			return
		}
	}
	a.newLine(j, i)
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
			node.VelocityX = .1 * (9*node.VelocityX + (x-node.X)/a.deltaT)
			node.VelocityY = .1 * (9*node.VelocityY + (y-node.Y)/a.deltaT)
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
	if a.nDots <= 0 {
		return
	}
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
			d.R = a.dotRadius(d.M)
		}
	}
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
		a.drawWeb()
		return
	}

	a.editClickVoid(x, y)
}

func (a *anim) pointerWheel(event js.Value) {
	z := event.Get("deltaY").Float()
	a.upDown(z)
}

func (a *anim) pointerRelease(event js.Value) {
	a.dragging = false
}

func (a *anim) pointerMove(event js.Value) {
	if !a.running {
		return
	}
	x := event.Get("clientX").Float()
	if a.dragging {
		y := event.Get("clientY").Float()
		a.positionDraggedDot(x, y)
	}
	a.wheelForce = (2*x/a.width - 1) * maxWheelForce
}

func (a *anim) keyup(event js.Value) {
	a.keyisdown = false
}

func (a *anim) keydown(event js.Value) {
	if a.keyisdown {
		return
	}
	a.keyisdown = true
	switch event.Get("code").String() {
	case "ArrowDown":
		event.Call("preventDefault")
		a.upDown(+sizeButtonClick)
	case "ArrowUp":
		event.Call("preventDefault")
		a.upDown(-sizeButtonClick)
	case "ArrowRight":
		event.Call("preventDefault")
		a.toggleRunEdit()
	}
}

func main() {
	height := js.Global().Get("innerHeight").Float() - 24
	width := js.Global().Get("innerWidth").Float() - 24
	a := newAnim(width, height, height*0.05, 128)
	eventHandlers := map[string]func(js.Value){
		"pointerdown": a.click,
		"wheel":       a.pointerWheel,
		"pointerup":   a.pointerRelease,
		"pointermove": a.pointerMove,
		"keyup":       a.keyup,
		"keydown":     a.keydown,
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
