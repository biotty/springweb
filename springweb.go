package springweb

import "math"

var ArmResist float64 = 1e-8
var SpringResist float64 = 1e-9

type Arm struct {
	K, w, InitAngle, PrevAngle, prevAngleUnrest float64
	Rotations                  int
}

type Spring struct {
	To             *Node
	K,Distance,prevDistance    float64
	FromArm, ToArm Arm
}

type Node struct {
	X, Y, R, M             float64
	VelocityX, VelocityY   float64
	Angle, wAvgSum float64
	Springs                []Spring
}

func (arm *Arm) Prepare() {
	arm.PrevAngle = arm.InitAngle
	arm.Rotations = 0
}

func (s *Spring) Prepare() {
	s.FromArm.Prepare()
	s.ToArm.Prepare()
}

func (node *Node) Prepare() {
	node.VelocityX = 0
	node.VelocityY = 0
	node.avgRotationsPrepare()
	for j, _ := range node.Springs {
		node.Springs[j].Prepare()
	}
}

func distanceXY(xDiff, yDiff float64) float64 {
	return math.Sqrt(math.Pow(xDiff, 2) + math.Pow(yDiff, 2))
}

func distance(a, b *Node) float64 {
	return distanceXY(a.X-b.X, a.Y-b.Y)
}

func (node *Node) angle(to *Node) float64 {
	return math.Atan2(to.Y-node.Y, to.X-node.X)
}

func NewNode(x, y, r, m float64) Node {
	return Node{x, y, r, m, 0, 0, 0, 0, nil}
}

func (node *Node) NewSpring(to *Node, k, a float64) {
	d := distance(node, to)
	node.Springs = append(node.Springs,
		Spring{to, k, d, d,
			Arm{a, 0, node.angle(to), 0, 0, 0},
			Arm{a, 0, to.angle(node), 0, 0, 0}})
}

func (node *Node) accelerate(forceX, forceY, duration float64) {
	w := duration / node.M
	node.VelocityX += forceX * w
	node.VelocityY += forceY * w
}

func (s *Spring) bounce(node *Node, duration float64) {
	xDiff := s.To.X - node.X
	yDiff := s.To.Y - node.Y
	actualDistance := distanceXY(xDiff, yDiff)
	xDiffN := xDiff / actualDistance
	yDiffN := yDiff / actualDistance
	contractF := s.K * (actualDistance - s.Distance)
	distIncr := actualDistance - s.prevDistance
	s.prevDistance = actualDistance
	if distIncr > 0 {
		contractF += SpringResist
	} else {
		contractF -= SpringResist
	}
	forceX := xDiffN * contractF
	forceY := yDiffN * contractF
	impactDepth := (node.R + s.To.R) - actualDistance
	if impactDepth > 0 {
		refDepth := math.Min(node.R, s.To.R)
		elasticF := s.K * s.Distance * impactDepth / refDepth
		forceX -= xDiffN * elasticF
		forceY -= yDiffN * elasticF
	}
	node.accelerate(forceX, forceY, duration)
	s.To.accelerate(-forceX, -forceY, duration)
}

func (arm *Arm) updateAngle(angle float64) {
	diff := angle - arm.PrevAngle
	arm.PrevAngle = angle
	if diff > math.Pi {
		arm.Rotations--
	}
	if diff < -math.Pi {
		arm.Rotations++
	}
}

func (arm *Arm) Angle() float64 {
	return arm.PrevAngle + float64(arm.Rotations)*math.Pi*2
}

func (node *Node) torque(arm *Arm, to *Node, duration float64) {
	d := distance(node, to)
	arm.w = arm.K / d
	Angle := arm.Angle()
	restAngle := arm.InitAngle + node.Angle
	angleUnrest := Angle - restAngle
	unrestIncr := angleUnrest - arm.prevAngleUnrest
	arm.prevAngleUnrest = angleUnrest
	if unrestIncr > 0 {
		angleUnrest += ArmResist*d
	} else if unrestIncr < 0 {
		angleUnrest -= ArmResist*d
	}
	normalizeAndTorqueF := angleUnrest * arm.w / d
	forceX := (to.Y - node.Y) * normalizeAndTorqueF
	forceY := (node.X - to.X) * normalizeAndTorqueF
	node.accelerate(-forceX, -forceY, duration)
	to.accelerate(forceX, forceY, duration)
}

func (s *Spring) torque(node *Node, duration float64) {
	node.torque(&s.FromArm, s.To, duration)
	s.To.torque(&s.ToArm, node, duration)
}

func (node *Node) move(duration float64) {
	dMove := duration * distanceXY(node.VelocityX, node.VelocityY)
	rMove := node.R * .9
	if dMove > rMove {
		velocityCap := rMove / dMove
		node.VelocityX *= velocityCap
		node.VelocityY *= velocityCap
	}
	node.X += node.VelocityX * duration
	node.Y += node.VelocityY * duration
}

func (node *Node) avgRotationsPrepare() {
	node.Angle = 0
	node.wAvgSum = 0
}

func avgRotations(nodes []Node) {
	iLast := len(nodes) - 1
	for iForward, _ := range nodes {
		i := iLast - iForward
		n := &nodes[i]
		for j, _ := range n.Springs {
			s := &n.Springs[j]
			t := s.To
			s.FromArm.updateAngle(n.angle(t))
			s.ToArm.updateAngle(t.angle(n))

			n.Angle += (s.FromArm.Angle() - s.FromArm.InitAngle) * s.FromArm.w
			t.Angle += (s.ToArm.Angle() - s.ToArm.InitAngle) * s.ToArm.w
			n.wAvgSum += s.FromArm.w
			t.wAvgSum += s.ToArm.w
		}
		n.Angle /= n.wAvgSum
	}
}

func StepsPrepare(nodes []Node) {
	for i, _ := range nodes {
		nodes[i].Prepare()
	}
}

func Step(nodes []Node, duration float64) {
	iLast := len(nodes) - 1
	for iForward, _ := range nodes {
		i := iLast - iForward
		n := &nodes[i]
		for j, _ := range n.Springs {
			s := &n.Springs[j]
			s.bounce(n, duration)
			s.torque(n, duration)
		}
		n.move(duration)
		n.avgRotationsPrepare()
	}
	avgRotations(nodes)
}
