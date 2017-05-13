package neat

import (
	"fmt"
	"math/rand"
)

// NodeGene is an implementation of each node in the graph representation of a
// genome. Each node consists of a node ID, its type, and the activation type.
type NodeGene struct {
	ID         int             // node ID
	Type       string          // node type
	Activation *ActivationFunc // activation function
}

// NewNodeGene returns a new instance of NodeGene, given its ID, its type, and
// the activation function of this node.
func NewNodeGene(id int, ntype string, activation *ActivationFunc) *NodeGene {
	return &NodeGene{id, ntype, activation}
}

// String returns a string representation of the node.
func (n *NodeGene) String() string {
	return fmt.Sprintf("[%s(%d, %s)]", n.Type, n.ID, n.Activation.Name)
}

// ConnGene is an implementation of a connection between two nodes in the graph
// representation of a genome. Each connection includes its input node, output
// node, connection weight, and an indication of whether this connection is
// disabled
type ConnGene struct {
	From     *NodeGene // input node
	To       *NodeGene // output node
	Weight   float64   // connection weight
	Disabled bool      // true if disabled
}

// NewConnGene returns a new instance of ConnGene, given the input and output
// node genes. By default, the connection is enabled.
func NewConnGene(from, to *NodeGene, weight float64) *ConnGene {
	return &ConnGene{from, to, weight, false}
}

// String returns the string representation of this connection.
func (c *ConnGene) String() string {
	connectivity := fmt.Sprintf("%.3f", c.Weight)
	if c.Disabled {
		connectivity = "/"
	}
	return fmt.Sprintf("%s--%s--%s", c.From.String(), connectivity, c.To.String())
}

// Genome encodes the weights and topology of the output network as a collection
// of nodes and connection genes.
type Genome struct {
	ID        int         // genome ID
	NodeGenes []*NodeGene // nodes in the genome
	ConnGenes []*ConnGene // connections in the genome
}

// NewGenome returns an instance of initial Genome with fully connected input
// and output layers.
func NewGenome(id, numInputs, numOutputs int) *Genome {
	nodeGenes := make([]*NodeGene, 0, numInputs+numOutputs)
	connGenes := make([]*ConnGene, 0, numInputs*numOutputs)

	for i := 0; i < numInputs; i++ {
		inputNode := NewNodeGene(i, "input", ActivationSet["identity"])
		nodeGenes = append(nodeGenes, inputNode)
	}
	for i := numInputs; i < numInputs+numOutputs; i++ {
		outputNode := NewNodeGene(i, "output", ActivationSet["sigmoid"])
		nodeGenes = append(nodeGenes, outputNode)

		for j := 0; j < numInputs; j++ {
			conn := NewConnGene(nodeGenes[j], outputNode, rand.NormFloat64()*6.0)
			connGenes = append(connGenes, conn)
		}
	}

	return &Genome{
		ID: id,
		NodeGenes: func() []*NodeGene {
		}(),
		ConnGenes: make([]*ConnGene, 0),
	}
}

// String returns the string representation of the genome.
func (g *Genome) String() string {
	str := fmt.Sprintf("Genome(%d):\n", g.ID)
	for _, conn := range g.ConnGenes {
		str += conn.String() + "\n"
	}
	return str[:len(str)-1]
}

// Mutate mutates the genome in three ways, by perturbing each connection's
// weight, by adding a node between two connected nodes, and by adding a
// connection between two nodes that are not connected.
func Mutate(g *Genome, ratePerturb, rateAddNode, rateAddConn float64) {
	// perturb connection weights
	for _, conn := range g.ConnGenes {
		if rand.Float64() < ratePerturb {
			conn.Weight += rand.NormFloat64()
		}
	}

	// add node between two connected nodes, by randomly selecting a connection;
	// only applied if there are connections in the genome
	if rand.Float64() < rateAddNode && len(g.ConnGenes) != 0 {
		selected := g.ConnGenes[rand.Intn(len(g.ConnGenes))]
		newNode := NewNodeGene(len(g.NodeGenes), "hidden", ActivationSet["sigmoid"])

		g.NodeGenes = append(g.NodeGenes, newNode)
		g.ConnGenes = append(g.ConnGenes, NewConnGene(selected.From, newNode, 1.0),
			NewConnGene(newNode, selected.To, selected.Weight))
		selected.Disabled = true
	}

	// add connection between two disconnected nodes; only applied if the selected
	// nodes are not connected yet, and the resulting connection doesn't make the
	// phenotype network recurrent
	if rand.Float64() < rateAddConn {
		selectedNode0 := g.NodeGenes[rand.Intn(len(g.NodeGenes))]
		selectedNode1 := g.NodeGenes[rand.Intn(len(g.NodeGenes))]

		for _, conn := range g.ConnGenes {
			if conn.From == selectedNode0 && conn.To == selectedNode1 {
				return
			}
		}

		newConn := NewConnGene(selectedNode0, selectedNode1, rand.NormFloat64()*6.0)
		g.ConnGenes = append(g.ConnGenes, newConn)
	}
}

// Crossover returns a new child genome by performing crossover between the two
// argument genomes.
//
// innovations is a temporary dictionary for the child genome's connection
// genes; it essentially stores all connection genes that will be contained
// in the child genome.
//
// Initially, all of one parent genome's connections are recorded to
// innovations. Then, as the other parent genome's connections are added, it
// checks if each connection already exists; if it does, swap with the other
// parent's connection by 50% chance. Otherwise, append the new connection.
func Crossover(id int, g0, g1 *Genome) *Genome {
	innovations := make(map[[2]int]*ConnGene)
	for _, conn := range g0.ConnGenes {
		innovations[[2]int{conn.From.ID, conn.To.ID}] = conn
	}
	for _, conn := range g1.ConnGenes {
		innov := [2]int{conn.From.ID, conn.To.ID}
		if innovations[innov] != nil {
			if rand.Float64() < 0.5 {
				innovations[innov] = conn
			}
		} else {
			innovations[innov] = conn
		}
	}

	// copy node genes
	largerParent := g0
	if len(g0.NodeGenes) < len(g1.NodeGenes) {
		largerParent = g1
	}
	nodeGenes := make([]*NodeGene, len(largerParent.NodeGenes))
	for i := range largerParent.NodeGenes {
		n := largerParent.NodeGenes[i]
		nodeGenes[i] = &NodeGene{n.ID, n.Type, n.Activation}
	}

	// copy connection genes
	connGenes := make([]*ConnGene, 0, len(innovations))
	for _, conn := range innovations {
		connGenes = append(connGenes, &ConnGene{
			From:     nodeGenes[conn.From.ID],
			To:       nodeGenes[conn.To.ID],
			Weight:   conn.Weight,
			Disabled: conn.Disabled,
		})
	}

	return &Genome{
		ID:        id,
		NodeGenes: nodeGenes,
		ConnGenes: connGenes,
	}
}
