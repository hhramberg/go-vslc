// tree.go provides functions for starting the parsing using goyacc and transforming the goyacc yySymTypes
// into a syntax tree of ir.Nodes. The scanner runs concurrently to the parser which lets one thread scan
// source strings for lexemes while the other parses the syntax tree using the grammar rules defined in parser.y.

package frontend

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	"vslc/src/ir"
	"vslc/src/util"
)

// Parse parses the syntax tree from the source code.
func Parse(src string) error {
	l := newLexer(src, lexGlobal)

	yyErrorVerbose = true

	// Start scanner and run it concurrently to the parser.
	go l.run()

	// Start parser.
	if a := yyParse(l); a != 0 {
		close(l.items)
		return fmt.Errorf("parser returned %d", a)
	}

	// Check if parser successfully created the syntax tree.
	if ir.Root == nil {
		return errors.New("root node is <nil>")
	}
	return nil
}

// TokenStream outputs the token stream from the given source string.
func TokenStream(src string) error {
	l := newLexer(src, lexGlobal)
	go l.run()

	wr := util.NewWriter()
	defer wr.Close()
	sb := strings.Builder{}
	tw := tabwriter.NewWriter(&sb, 10, 20, 2, ' ', 0)
	_, _ = fmt.Fprintf(tw, "Value\tType\tPosition\n")
	for {
		t := l.nextItem()
		switch t.typ {
		case itemEOF:
			var err error = nil
			if err2 := tw.Flush(); err2 != nil {
				err = err2
			}
			wr.WriteString(sb.String())
			return err
		case itemError:
			wr.WriteString(sb.String())
			return errors.New(t.val)
		default:
			if len(t.val) > 20 {
				_, _ = fmt.Fprintf(tw, "%.17q...\t%s\tline: %d:%d\n", t.val, yyTokname(int(t.typ)), t.line, t.pos)
			} else {
				_, _ = fmt.Fprintf(tw, "%q\t%s\tline: %d:%d\n", t.val, yyTokname(int(t.typ)), t.line, t.pos)
			}
		}
	}
}

// nodeInit creates a yySymType struct which holds an ir.Node datatype.
func nodeInit(typ ir.NodeType, data interface{}, line, pos int, args ...yySymType) yySymType {
	n := ir.Node{Typ: typ, Line: line, Pos: pos, Children: make([]*ir.Node, len(args))}
	switch typ {
	case ir.INTEGER_DATA:
		if num, err := parseInteger(data); err == nil {
			n.Data = num
		} else {
			fmt.Println(err)
			n.Data = data.(string)
		}
	case ir.FLOAT_DATA:
		if num, err := parseFloat(data); err == nil {
			n.Data = float32(num)
		} else {
			fmt.Println(err)
			n.Data = data.(string)
		}
	default:
		n.Data = data
	}
	for i1, e := range args {
		n.Children[i1] = e.node
	}
	return yySymType{typ: int(typ), val: "N/A", node: &n}
}

// parseInteger parses an interface{} as an integer. This function returns a 32-bit integer value.
func parseInteger(n interface{}) (int, error) {
	if s, ok := n.(string); ok {
		i, err := strconv.Atoi(s)
		return int(int32(i)), err
	}
	return -1, fmt.Errorf("could not parse integer number from %v", n)
}

// parseFloat parses an interface{} as an integer. This function returns a 32-bit floating point value.
func parseFloat(n interface{}) (float64, error) {
	if s, ok := n.(string); ok {
		return strconv.ParseFloat(s, 32)
	}
	return -1.0, fmt.Errorf("could not parse float from %v", n)
}
