package main

import (
	"math"
	"math/rand"
	"syscall/js"
	"time"

	"github.com/biotty/springweb"
)

const (
	defaultK            = 1.
	armKFactor          = 1e3
	minK                = defaultK * .25
	maxK                = defaultK * 5
	defaultMass         = 5e-3
	minMass             = defaultMass * .25
	maxMass             = defaultMass * 5
	platformBounce      = .5
	platformSpeed       = 9
	platformStick       = .3
	gravity             = 7e2
	maxWheelForce       = 1.1
	maxWheelVelocity    = 1e1
	wheelGyrationFactor = 1
	wheelDriveArmFactor = 1
	voidColor           = "#ffd"
	barColor            = "#bd3"
	barTextColor        = "#451"
	barTextHighColor    = "#fff"
	lineColor           = "rgba(96, 32, 0, 0.2)"
	platformColor       = "rgba(0, 128, 128, 0.5)"
	rightForceColor     = "rgba(0, 0, 255, 0.2)"
	leftForceColor      = "rgba(255, 0, 0, 0.2)"
	letterColor         = "rgba(64,32,200,.5)"
	letterCupColor      = "rgba(64,32,200,.75)"
)

func (a *anim) vary() float64 {
	return .8 + .4*a.rands.Float64()
}

func (a *anim) newDotM(x, y, m float64) {
	m *= a.vary()
	r := a.dotRadius(m)
	a.dots[a.nDots] = springweb.NewNode(x, y, r, m)
	a.nDots++
}

func (a *anim) newDot(x, y float64) {
	a.newDotM(x, y, defaultMass)
}

func (a *anim) newLineK(i, j int, k float64) {
	k *= a.vary()
	a.dots[i].NewSpring(&a.dots[j], k, armKFactor*k)
}

func (a *anim) newLine(i, j int) {
	if i <= j { // contract: springweb
		log("newLine", i, j)
		return
	}
	a.newLineK(i, j, defaultK)
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

type wheel struct {
	angleVelocity float64
	angle         float64
	onPlatform    int
}

type anim struct {
	width, height, dotSize float64
	dots                   []springweb.Node
	nDots                  int
	iLetterDots            int
	nCarDots               int
	ctx                    js.Value
	images                 []js.Value
	callback               js.Func
	lastCall               time.Time
	deltaT                 float64
	viewX                  float64
	wheelForce             float64
	wheels                 []wheel
	nWheels                int
	platforms              []platform
	nPlatforms             int
	alienLetters           []int
	nLetterAliens          int
	haveLetters            []bool
	rands                  *rand.Rand
}

func (a *anim) setCallback() {
	a.callback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		t := time.Now()

		deltaT := 1e-9 * float64(t.Sub(a.lastCall))
		if deltaT < .3 || a.deltaT == 0 {
			a.deltaT = deltaT
		}
		a.lastCall = t

		springweb.Step(a.dots[:a.nDots], a.deltaT)
		a.wheelsStep()
		a.lettersStep()
		a.platformsStep(deltaT)
		a.gravityStep()
		a.viewBorderStep()
		a.viewScrollStep()
		a.worldCycle()
		a.drawView()
		a.drawControl()

		js.Global().Call("requestAnimationFrame", a.callback)
		return nil
	})
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
	ctx.Set("font", "15px Arial")
	a := anim{width, height, dotSize,
		make([]springweb.Node, nNodes), 0, 0, 0,
		ctx, images, js.Func{}, time.Time{}, 0, 0, 0,
		nil, 2, nil, 15, nil, 7, nil,
		rand.New(rand.NewSource(time.Now().UnixNano()))}

	a.wheels = make([]wheel, a.nWheels)
	a.platforms = make([]platform, a.nPlatforms)
	a.alienLetters = make([]int, a.nLetterAliens)
	a.haveLetters = make([]bool, 26)
	a.setCallback()
	return &a
}

func (a *anim) barHeight() float64 {
	return a.dotSize * 2
}

func (a *anim) dotRadius(mass float64) float64 {
	return a.dotSize * math.Sqrt(mass/defaultMass)
}

func (a *anim) lineWidth(k float64) float64 {
	return a.dotSize * k / (2 * defaultK)
}

func alphabetLetter(i int) int {
	return i + 65
}

func (a *anim) drawLettersHave(have bool, color string) {
	a.ctx.Call("save")
	a.ctx.Set("fillStyle", color)
	y := a.barHeight()
	s := a.dotSize / 19
	n := len(a.haveLetters)
	for i := 0; i < n; i++ {
		if have == a.haveLetters[i] {
			x := ((float64(i)-.5*float64(n))*.03 + .5) * a.width
			a.ctx.Call("setTransform", s, 0, 0, s, x, y/2)
			text := string(rune(alphabetLetter(i)))
			a.ctx.Call("fillText", text, 0, 0)
		}
	}
	a.ctx.Call("restore")
}

func (a *anim) drawBar() {
	a.ctx.Set("fillStyle", barColor)
	a.ctx.Call("fillRect", 0, 0, a.width, a.barHeight())
	a.drawLettersHave(false, barTextColor)
	a.drawLettersHave(true, barTextHighColor)
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
	y := a.barHeight()
	a.ctx.Call("fillRect", x, y, w, y*.5)
}

func (a *anim) clear() {
	a.ctx.Set("fillStyle", voidColor)
	a.ctx.Call("fillRect", 0, a.barHeight(), a.width, a.height)
	a.drawBar()
}

func (a *anim) drawDot(i int) {
	d := a.dots[i]
	b := d.Angle
	if i >= a.iLetterDots {
		r := d.R
		a.ctx.Set("fillStyle", letterCupColor)
		a.ctx.Call("beginPath")
		a.ctx.Call("arc", d.X-a.viewX, d.Y, r, b, math.Pi+b)
		a.ctx.Call("fill")
		a.ctx.Call("closePath")
	}
	img := a.images[0]
	if i >= a.iLetterDots {
		text := string(rune(alphabetLetter(a.alienLetters[i-a.iLetterDots])))
		s := d.R / 9
		a.ctx.Set("fillStyle", letterColor)
		a.ctx.Call("save")
		a.ctx.Call("setTransform", s, 0, 0, s, d.X-a.viewX, d.Y)
		a.ctx.Call("rotate", b)
		a.ctx.Call("fillText", text, -5, 4)
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
	a.ctx.Set("strokeStyle", lineColor)
	for i := 0; i < a.nDots; i++ {
		from := a.dots[i]
		for _, s := range from.Springs {
			a.drawLineTo(i, s.To.X, s.To.Y, s.K)
		}
	}
	for i := 0; i < a.nDots; i++ {
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
		if i >= a.nWheels && i < a.iLetterDots {
			continue
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

func (a *anim) wheelsStep() {
	for i := 0; i < a.nWheels; i++ {
		a.wheelRotation(i)
	}
}

func (a *anim) lettersStep() {
	for i := a.iLetterDots; i < a.nDots; i++ {
		u := a.alienLetters[i-a.iLetterDots]
		if !a.haveLetters[u] {
			d := &a.dots[i]
			c := &a.dots[a.nCarDots-1]
			if distanceXY(c.X-d.X, c.Y-d.Y) < a.dotSize*3 {
				a.haveLetters[u] = true
			}
		}
	}
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
	leftY := (.65 + a.rands.Float64()) * .75 * a.height
	rightY := leftY + (a.rands.Float64()-.5)*.4*a.height
	height := (1 + a.rands.Float64()) * .4 * a.dotSize
	a.platforms[i] = newPlatform(leftX, rightX, leftY, rightY, height)
}

func (a *anim) alienCycle(i int) {
	d := &a.dots[i]
	a.alienLetters[i-a.iLetterDots] = a.rands.Intn(len(a.haveLetters))
	x := a.viewX + (1+a.rands.Float64())*a.width + a.dotSize
	y := 0.
	h := a.dotSize * 2
	for d != nil {
		d.X = x + a.rands.Float64()*a.dotSize*.125
		d.Y = y + a.rands.Float64()*a.dotSize*.125
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
	for i := a.iLetterDots; i < a.nDots; i++ {
		if a.dots[i].X < a.viewX-a.width*.5 || a.dots[i].Y > a.height*1.5 {
			a.alienCycle(i)
		}
	}
}

func (a *anim) appendAliens() {
	nAliens := 5
	nBody := 2 // minimum: 1
	n := a.nDots + nAliens*(1+nBody)
	if n >= len(a.dots) {
		return // overflow: skip aliens
	}
	a.iLetterDots = n - nAliens
	for q := a.nDots; q < n; q++ {
		a.newDot(0, 0)
	}
	j := a.iLetterDots
	i := a.nDots
	for i > a.iLetterDots {
		i--
		j--
		a.newLine(i, j)
		for q := 1; q < nBody; q++ {
			a.newLine(j, j-1)
			j--
		}
		a.alienCycle(i)
	}
}

func (a *anim) start() {
	springweb.StepsPrepare(a.dots[:a.nDots])
	a.lastCall = time.Now()
	a.nCarDots = a.nDots
	a.appendAliens()
	js.Global().Call("requestAnimationFrame", a.callback)
}

func (a *anim) pointerMove(event js.Value) {
	x := event.Get("clientX").Float()
	a.wheelForce = (2*x/a.width - 1) * maxWheelForce
}

func main() {
	height := js.Global().Get("innerHeight").Float() - 24
	width := js.Global().Get("innerWidth").Float() - 24
	a := newAnim(width, height, height*0.05, 24)
	js.Global().Call("addEventListener", "pointermove",
		js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			a.pointerMove(args[0])
			return nil
		}))

	h := 2 * a.dotSize
	a.newDot(h, a.height-h)
	a.newDot(h*3, a.height-h)
	a.newDotM(h*(1+a.vary()), a.height-h*1.5, minMass)
	a.newLine(1, 0)
	a.newLineK(2, 1, minK)
	a.newLineK(2, 0, minK)
	a.start()
	log("h", height)
	log("w", width)

	<-make(chan bool)
}

func log(args ...interface{}) {
	js.Global().Get("console").Call("log", args...)
}
