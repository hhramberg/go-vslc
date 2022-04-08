package ir

import "fmt"

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// NodeType differentiates the types of nodes in the intermediate syntax tree.
type NodeType int

// Node represents a single node in the intermediate syntax tree representation.
type Node struct {
	Typ      NodeType    // The type of Node, i.e. string data, relation node or number data.
	Line     int         // Line in source code Node is declared.
	Pos      int         // Position on the line in source code Node is declared.
	Data     interface{} // Data node is holding: used for strings, number data and identifier data.
	Entry    *Symbol     // Symbol table entry for this node, if it exists.
	Children []*Node     // Children of this node that constitutes its local sub-tree.
}

// ---------------------
// ----- Constants -----
// ---------------------

// Root node of program.
var Root *Node

const (
	PROGRAM NodeType = iota
	GLOBAL_LIST
	GLOBAL
	STATEMENT_LIST
	PRINT_LIST
	EXPRESSION_LIST
	VARIABLE_LIST
	TYPED_VARIABLE_LIST
	ARGUMENT_LIST
	PARAMETER_LIST
	DECLARATION_LIST
	FUNCTION
	STATEMENT
	BLOCK
	ASSIGNMENT_STATEMENT
	RETURN_STATEMENT
	PRINT_STATEMENT
	NULL_STATEMENT
	IF_STATEMENT
	WHILE_STATEMENT
	EXPRESSION
	RELATION
	DECLARATION
	PRINT_ITEM
	IDENTIFIER_DATA
	INTEGER_DATA
	FLOAT_DATA
	STRING_DATA
	TYPE_DATA
)

// nt provides an array of strings used for printing NodeType in a print friendly manner.
var nt = [...]string{
	"PROGRAM",
	"GLOBAL_LIST",
	"GLOBAL",
	"STATEMENT_LIST",
	"PRINT_LIST",
	"EXPRESSION_LIST",
	"VARIABLE_LIST",
	"TYPED_VARIABLE_LIST",
	"ARGUMENT_LIST",
	"PARAMETER_LIST",
	"DECLARATION_LIST",
	"FUNCTION",
	"STATEMENT",
	"BLOCK",
	"ASSIGNMENT_STATEMENT",
	"RETURN_STATEMENT",
	"PRINT_STATEMENT",
	"NULL_STATEMENT",
	"IF_STATEMENT",
	"WHILE_STATEMENT",
	"EXPRESSION",
	"RELATION",
	"DECLARATION",
	"PRINT_ITEM",
	"IDENTIFIER_DATA",
	"INTEGER_DATA",
	"FLOAT_DATA",
	"STRING_DATA",
	"TYPE_DATA",
}

// ----------------------
// ----- functions ------
// ----------------------

// String returns a print friendly string of Node n.
func (n *Node) String() string {
	if n == nil {
		return "---> [NIL POINTER]"
	}
	typ := int(n.Typ)
	if typ > len(nt) || typ < 0 {
		// This Node has been mis-configured.
		return fmt.Sprintf("---> MISCONFIGURED NODE [Node.Typ = %d]", typ)
	}
	if n.Data == nil {
		return fmt.Sprintf("%s", nt[n.Typ])
	}

	switch n.Typ {
	case STRING_DATA:
		if n.Entry == nil {
			// BEFORE syntax table creation. Get string data from node.
			return fmt.Sprintf("%s [%q]", nt[n.Typ], n.Data)
		} else {
			// AFTER syntax table creation. Get string data from global string table.
			return fmt.Sprintf("%s [%s]", nt[n.Typ], Strings.St[n.Data.(int)])
		}
	case INTEGER_DATA:
		return fmt.Sprintf("%s [%d]", nt[n.Typ], n.Data)
	//case IDENTIFIER_DATA:
	//	return fmt.Sprintf("%s [%q] bound: %t", nt[n.Typ], n.Data, n.Entry != nil)
	default:
		return fmt.Sprintf("%s [%q]", nt[n.Typ], n.Data)
	}
}

// Type returns a print friendly string of the Node n' type.
func (n *Node) Type() string {
	return nt[n.Typ]
}

// Print recursively prints this Node and all its Children while indenting for every recursive call.
// depth is the number of times nodes are padded to the right, having the root node with padding 0.
// If showDepth is true the method also prints the depths of the nodes.
func (n *Node) Print(depth int, showDepth bool) {
	if depth < 0 {
		depth = 0
	}

	if n == nil {
		if showDepth {
			fmt.Printf("%d %*c%s\n", depth, depth<<1, 0, "---> NIL")
		} else {
			fmt.Printf("%*c%s\n", depth<<1, 0, "---> NIL")
		}
		return
	}
	if showDepth {
		fmt.Printf("%d %*c%s\n", depth, depth<<1, 0, n.String())
	} else {
		fmt.Printf("%*c%s\n", depth<<1, 0, n.String())
	}

	for _, e := range n.Children {
		e.Print(depth+1, showDepth)
	}
}
