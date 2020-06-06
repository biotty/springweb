package springweb

import (
	"math"
)

type Spring struct {
	To *Node
	Distance, K float64
}

type Node struct {
	X, Y, InvMass  float64
	VelocityX float64
	VelocityY float64
	Springs []Spring
}

func distance(a *Node, b *Node) float64 {
	return math.Sqrt(math.Pow(a.X-b.X, 2)+math.Pow(a.Y-b.Y, 2))
}

func NewNode(x, y, mass float64) Node {
	return Node{x, y, 1 / mass, 0, 0, nil}
}

func (node *Node) NewSpring(to *Node, k float64) {
	node.Springs = append(node.Springs, Spring{to, distance(node, to), k})
}

func (node *Node) accelerate(forceX, forceY, duration float64) {
	node.VelocityX += node.InvMass * forceX * duration
	node.VelocityY += node.InvMass * forceY * duration
}

func (s Spring) accelerate(node *Node, duration float64) {
	actualDistance := distance(s.To, node)
	contractF := s.K * (actualDistance - s.Distance)
	forceX := contractF * (s.To.X - node.X) / actualDistance
	forceY := contractF * (s.To.Y - node.Y) / actualDistance
	node.accelerate(forceX, forceY, duration)
	s.To.accelerate(-forceX, -forceY, duration)
}

func (node *Node) move(duration float64) {
	node.X += node.VelocityX * duration
	node.Y += node.VelocityY * duration
}

func Step(nodes []Node, duration float64) {
	iLast := len(nodes) - 1
	for iForward, _ := range nodes {
		i := iLast - iForward
		for _, s := range nodes[i].Springs {
			s.accelerate(&nodes[i], duration)
		}
		nodes[i].move(duration)
	}
}
