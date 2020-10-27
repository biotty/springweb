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
	defaultMass         = 5e-3
	minMass             = defaultMass * .15
	maxMass             = defaultMass * 5
	platformBounce      = .5
	platformSpeed       = 9
	platformStick       = .3
	gravity             = 1e3  // note: dep. on dotSize.  make independent by multiplying at calc.
	maxWheelForce       = 2.0  //       idem
	maxWheelVelocity    = 2e1  // note: measured in dotSize as unit, so constant not dep. on dotSize
	wheelGyrationFactor = 1 // note: deps as explained
	wheelDriveArmFactor = 1 //       idem .. but maybe not this one, as it is purely angular
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
	angleVelocity float64
	angle         float64
	onPlatform    int
}

type platform struct {
	leftX, rightX, leftY, rightY, height, surfaceX, surfaceY, surfaceP float64
}

func distanceXY(xDiff, yDiff float64) float64 {
	return math.Sqrt(math.Pow(xDiff, 2) + math.Pow(yDiff, 2))
}

func newPlatform(leftX, rightX, leftY, rightY, height float64) platform {
	leftwardsX := rightX - leftX
	leftwardsY := rightY - leftY
	length := distanceXY(leftwardsX, leftwardsY)
	surfaceX := leftwardsY / length
	surfaceY := -leftwardsX / length
	surfaceP := leftX*surfaceX + leftY*surfaceY
	return platform{leftX, rightX, leftY, rightY, height, surfaceX, surfaceY, surfaceP}
}

func (p *platform) onHeight(d *springweb.Node) float64 {
	if d.X <= p.leftX || d.X >= p.rightX { // simplification: horizontal
		return 1e9
	}

	h := d.X*p.surfaceX + d.Y*p.surfaceY - p.surfaceP - d.R
	if h < 0 {
		return 1e9
	}
	leftSlope := d.X - p.leftX
	if leftSlope < p.height {
		h += p.height - leftSlope
	} else {
		rightSlope := p.rightX - d.X
		if rightSlope < p.height {
			h += p.height - rightSlope
		}
	}
	return h
}

func (p *platform) bounce(d *springweb.Node, depth float64) {
	vProd := d.VelocityX*p.surfaceX + d.VelocityY*p.surfaceY
	if vProd < 0 {
		d.VelocityX = platformBounce * (vProd*-2*p.surfaceX + d.VelocityX)
		d.VelocityY = platformBounce * (vProd*-2*p.surfaceY + d.VelocityY)
	}
	d.X += depth * p.surfaceX
	d.Y += depth * p.surfaceY
}

type anim struct {
	width, height, dotSize float64
	dots, resetNodes       []springweb.Node
	nDots                  int
	nGassDots              int
	nCarDots               int
	selectedDot            int
	dragging               bool
	ctx                    js.Value
	images                 []js.Value
	callback               js.Func
	lastCall               time.Time
	deltaT                 float64
	running                bool
	keyisdown              bool
	viewX, resetViewX      float64
	wheelForce             float64
	wheels                 []wheel
	nWheels                int
	platforms              []platform
	nPlatforms             int
	rands                  *rand.Rand
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
		make([]springweb.Node, nNodes), nil, 0, 0, 0, 0, false,
		ctx, images, js.Func{}, time.Time{}, 0, false, false, 0, 0, 0,
		nil, 2, nil, 15, rand.New(rand.NewSource(time.Now().UnixNano()))}

	a.wheels = make([]wheel, a.nWheels)
	a.platforms = make([]platform, a.nPlatforms)

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
			a.platformsStep(deltaT)
			a.gravityStep()
			a.viewBorderStep()
			a.viewScrollStep()
			a.worldCycle()
			a.positionDraggedDot(x, y)
			a.drawView()
			a.drawControl()
		}
		a.lastCall = t
		js.Global().Call("requestAnimationFrame", a.callback)
		return nil
	})
	return &a
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
	leftX := 0.0
	for i := 0; i < 3; i++ {
		rightX := a.buttonRight(i)
		center := .5 * (leftX + rightX)
		if i == 0 && a.running {
			y := a.buttonHeight()
			a.ctx.Call("fillRect", center-y/2, y/4, y/2, y/2)
			a.ctx.Call("fillRect", center+y/4, y/4, y/2, y/2)
		} else {
			drawTriangle(a.ctx, i, center, a.buttonHeight())
		}
		leftX = rightX
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
	if !a.running || i == a.selectedDot || i >= a.nGassDots {
		r := d.R
		if a.running {
			r *= 1.1
		}
		a.ctx.Call("beginPath")
		a.ctx.Call("arc", d.X-a.viewX, d.Y, r, 0, math.Pi*2)
		a.ctx.Call("fill")
		a.ctx.Call("closePath")
	}
	if a.running {
		img := a.images[0]
		b := d.Angle
		if i >= a.nGassDots {
			letter := string(rune(65 + (i - a.nGassDots)))
			s := d.R / 9
			a.ctx.Call("save")
			a.ctx.Set("fillStyle", "white")
			a.ctx.Call("setTransform", s, 0, 0, s, d.X-a.viewX, d.Y)
			a.ctx.Call("rotate", b)
			a.ctx.Call("fillText", letter, -5, 4)
			a.ctx.Call("restore")
			return
		}
		if i < a.nWheels {
			img = a.images[1]
			b += a.wheels[i].angle // alt: =
		}
		a.ctx.Call("save")
		a.ctx.Call("translate", d.X-a.viewX, d.Y)
		a.ctx.Call("rotate", b)
		a.ctx.Call("drawImage", img, -d.R, -d.R, d.R*2, d.R*2)
		a.ctx.Call("restore")
	}
}

func (a *anim) drawLineTo(i int, x, y, k float64) {
	d := a.dots[i]
	a.ctx.Set("lineWidth", a.lineWidth(k))
	a.ctx.Call("beginPath")
	a.ctx.Call("moveTo", d.X-a.viewX, d.Y)
	a.ctx.Call("lineTo", x-a.viewX, y)
	a.ctx.Call("stroke")
}

func (a *anim) drawView() {
	a.clear()
	a.ctx.Set("font", "15px Arial")
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
	a.drawPlatforms()
}

func (a *anim) drawPlatforms() {
	a.ctx.Set("strokeStyle", platformColor)
	for i := 0; i < a.nPlatforms; i++ {
		p := a.platforms[i]
		h := p.height
		a.ctx.Set("lineWidth", h)
		a.ctx.Call("beginPath")
		a.ctx.Call("moveTo", p.leftX-a.viewX, p.leftY-h*.5)
		a.ctx.Call("lineTo", p.rightX-a.viewX, p.rightY-h*.5)
		a.ctx.Call("stroke")
	}
}

func (a *anim) platformsStep(deltaT float64) {
	for i := 0; i < a.nDots; i++ {
		d := &a.dots[i]
		if i >= a.nWheels && i < a.nGassDots {
			continue // note: avoid car hooked stuck
		}
		hOn := 0.
		jOn := 0
		for j := 0; j < a.nPlatforms; j++ {
			p := &a.platforms[j]
			h := p.onHeight(d)
			if h < p.height {
				if h > hOn {
					hOn = h
					jOn = j
				}
			}
		}
		if hOn > 0 {
			a.platforms[jOn].bounce(d, hOn*deltaT*platformSpeed)
			if i < a.nWheels {
				a.wheels[i].onPlatform = jOn
			}
		}
	}
}

func (a *anim) gravityStep() {
	for i := 0; i < a.nDots; i++ {
		d := &a.dots[i]
		if d.Y < a.height-d.R {
			d.VelocityY += gravity * a.deltaT
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
	if j == a.nPlatforms { // off-by-1: on the ground
		if d.Y+d.R < a.height-a.dotSize*platformStick {
			w.onPlatform = -1
		}
	} else if j >= 0 {
		p := &a.platforms[j]
		if p.onHeight(d) > p.height+a.dotSize*platformStick {
			w.onPlatform = -1
		}
	}

	if w.onPlatform >= 0 {
		w.angleVelocity = d.VelocityX / d.R // simplification: horizontal
		if a.wheelVelocityBelowMax(d.VelocityX) {
			d.VelocityX += d.R * a.wheelForce * a.deltaT / d.M
			d.Angle -= a.wheelForce * wheelDriveArmFactor
		}
	} else {
		if a.wheelVelocityBelowMax(w.angleVelocity * d.R) {
			w.angleVelocity += a.wheelForce * a.deltaT / (d.M * wheelGyrationFactor)
		}
	}

	w.angle += w.angleVelocity * a.deltaT
}

func (a *anim) viewBorderStep() {
	for i := 0; i < a.nCarDots; i++ {
		d := &a.dots[i]
		if d.VelocityX < 0 && d.X < a.viewX+d.R {
			d.VelocityX *= -platformBounce
			d.X = a.viewX + d.R
		}
		if d.VelocityY > 0 && d.Y > a.height-d.R {
			d.VelocityY *= -platformBounce
			d.Y = a.height - d.R
			if i < a.nWheels {
				a.wheels[i].onPlatform = a.nPlatforms // q: on the ground
			}
		}
	}
}

func (a *anim) viewScrollStep() {
	if a.nDots == 0 {
		return
	}
	x := .5 * (a.dots[0].X + a.dots[1].X)
	q := a.width * .5
	if x > a.viewX+q {
		a.viewX = x - q
	}
}

func (a *anim) platformCycle(i int) {
	leftX := a.viewX + (1+a.rands.Float64())*a.width
	rightX := leftX + (1+a.rands.Float64())*.2*a.width
	leftY := (.65+a.rands.Float64())*.75*a.height
	rightY := leftY + (a.rands.Float64()-.5)*.4*a.height
	height := (1+a.rands.Float64())*.4*a.dotSize
	a.platforms[i] = newPlatform(leftX, rightX, leftY, rightY, height)
}

func (a *anim) alienCycle(i int) {
	d := &a.dots[i]
	x := a.viewX + (a.rands.Float64())*a.width
	y := 0.
	h := a.dotSize * 2
	for d != nil {
		d.X = x + a.rands.Float64() * a.dotSize * .125
		d.Y = y + a.rands.Float64() * a.dotSize * .125
		y += h
		if len(d.Springs) != 0 {
			d.Springs[0].Distance = h
			d = d.Springs[0].To // fornow: assume only one spring
		} else {
			d = nil
		}
	}
}

func (a *anim) worldCycle() {
	for i := 0; i < a.nPlatforms; i++ {
		if a.platforms[i].rightX < a.viewX {
			a.platformCycle(i)
		}
	}
	for i := a.nGassDots; i < a.nDots; i++ {
		if a.dots[i].X < a.viewX-a.width*.5 || a.dots[i].Y > a.height*1.5 {
			a.alienCycle(i)
		}
	}
}

func (a *anim) appendAliens() {
	nAliens := 7
	nBody := 2 // minimum: 1
	n := a.nCarDots + nAliens * (1 + nBody)
	if n >= len(a.dots) {
		return  // overflow: skip aliens
	}
	a.nDots = n
	a.nGassDots = n - nAliens
	for i := a.nCarDots; i < a.nDots; i++ {
		m := defaultMass
		r := a.dotRadius(m)
		a.dots[i] = springweb.NewNode(0, 0, r, m)
	}
	j := a.nGassDots
	i := a.nDots
	for i > a.nGassDots {
		i--
		k := defaultK
		j--
		a.dots[i].NewSpring(&a.dots[j], k, armKFactor*k)
		for q := 1; q < nBody; q++ {
			a.dots[j].NewSpring(&a.dots[j-1], k, armKFactor*k)
			j--
		}
		a.alienCycle(i)
	}
}

func (a *anim) toggleRunEdit() {
	a.running = !a.running
	if a.running {
		a.resetNodes = make([]springweb.Node, a.nDots)
		a.resetViewX = a.viewX
		copy(a.resetNodes, a.dots)
		springweb.StepsPrepare(a.dots[:a.nDots])
		a.lastCall = time.Now()
		a.nCarDots = a.nDots
		a.appendAliens()
		js.Global().Call("requestAnimationFrame", a.callback)
	} else {
		copy(a.dots, a.resetNodes)
		a.nDots = a.nCarDots
		for i := 0; i < a.nDots; i++ {
			a.dots[i].X += a.viewX - a.resetViewX
		}
		a.selectedDot = a.nDots - 1
		a.drawView()
	}
}

func (a *anim) editClickVoid(x, y float64) {
	if a.nDots == len(a.dots) {
		return
	}
	a.selectedDot = a.nDots
	a.newDot(x, y)
	a.drawView()
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
	a.drawView()
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
	x += a.viewX

	if a.running {
		a.clickRunning(x, y)
		return
	}

	i := a.findDot(x, y)
	if i >= 0 {
		a.editClickDot(i)
		a.drawView()
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
		a.positionDraggedDot(x+a.viewX, y)
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
