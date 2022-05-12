package lir

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	tree "vslc/src/ir"
	"vslc/src/ir/lir/types"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// funcWrapper wraps a LIR Function and its source node in the syntax tree.
type funcWrapper struct {
	node  *tree.Node
	entry *Function
}

// symTab is a symbol table that implements a hash map and a read/write mutex for thread safe access.
type symTab struct {
	m map[string]Value
	sync.RWMutex
}

// ---------------------
// ----- Constants -----
// ---------------------

// mapSize defines a pre-defined size of the globals hash map. 16 is thought to be reasonable for small VSL programs.
const mapSize = 16

// -------------------
// ----- Globals -----
// -------------------

// reservedFunctionNames defines a list of function names that cannot be assigned to VSL functions.
var reservedFunctionNames = []string{
	"main",
	"printf",
	"atof",
	"atoi",
}

// ---------------------
// ----- Functions -----
// ---------------------

// GenLIR generates lightweight intermediate representation from the syntax tree.
func GenLIR(opt util.Options, root *tree.Node) (*Module, error) {
	m := CreateModule(filepath.Base(opt.Src)) // The LIR module.
	if opt.Threads > 1 {
		// Parallel.
		t := opt.Threads
		l := len(root.Children)
		if t > l {
			t = l
		}
		n := l / t
		res := l % t

		start := 0
		end := n

		wg := sync.WaitGroup{}
		wg.Add(t)

		perr := util.NewPerror(t)

		// funcs hold LIR function wrappers.
		funcs := make([]funcWrapper, 0, t)

		// cfunc recieves function wrappers from worker go routines.
		cfuncs := make(chan []funcWrapper, t)

		// Spawn t worker go routines.
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				// This worker go routine should perform one residual job.
				end++
			}

			// Spawn go routine.
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done()
				funcs := make([]funcWrapper, 0, end-start)
				for _, e1 := range root.Children[start:end] {
					if e1.Typ == tree.DECLARATION {
						// Variable declaration.
						if err := genDeclarationGlobal(e1, m); err != nil {
							perr.Append(err)
							continue
						}
					} else {
						// Function declaration.
						f, err := genFunctionHeader(e1, m)
						if err != nil {
							perr.Append(err)
							continue
						}
						funcs = append(funcs, funcWrapper{
							node:  e1,
							entry: f,
						})
					}
				}
				cfuncs <- funcs
			}(start, end, &wg)

			start = end
			end += n
		}

		// Wait for worker go routines.
		wg.Wait()

		// Check for errors.
		if perr.Len() > 0 {
			close(cfuncs)
			for e1 := range perr.Errors() {
				fmt.Println(e1)
			}
			return nil, fmt.Errorf("%d errors during parallel LIR generation", perr.Len())
		}
		perr.Flush()

		// Retrieve all functions wrappers and append them to function definition slice.
		close(cfuncs)
		for e1 := range cfuncs {
			funcs = append(funcs, e1...)
		}

		// Generate LIR function bodies.
		t = opt.Threads
		l = len(funcs)
		if t > l {
			t = l
		}
		n = l / t
		res = l % t

		start = 0
		end = n

		// Spawn t worker go routines.
		wg.Add(t)
		for i1 := 0; i1 < t; i1++ {
			if i1 < res {
				end++
			}

			// Spawn worker go routine.
			go func(start, end int, wg *sync.WaitGroup) {
				defer wg.Done()
				for _, e2 := range funcs[start:end] {
					if err := genFunctionBody(e2.node, e2.entry); err != nil {
						perr.Append(err)
					}
				}
			}(start, end, &wg)
			start = end
			end += n
		}

		// Wait for worker threads to finish,
		wg.Wait()

		// Check for errors.
		if perr.Len() > 0 {
			for e1 := range perr.Errors() {
				fmt.Println(e1)
			}
			return nil, fmt.Errorf("%d errors during parallel LIR generation", perr.Len())
		}

		perr.Stop()
	} else {
		// Sequential.
		funcs := make([]funcWrapper, 0, len(root.Children))
		for _, e1 := range root.Children {
			if e1.Typ == tree.DECLARATION {
				// Global variable declaration.
				if err := genDeclarationGlobal(e1, m); err != nil {
					return nil, err
				}
			} else {
				// Function declaration.
				f, err := genFunctionHeader(e1, m)
				if err != nil {
					return nil, err
				}
				funcs = append(funcs, funcWrapper{
					node:  e1,
					entry: f,
				})
			}
		}

		// Generate function bodies.
		for _, e1 := range funcs {
			if err := genFunctionBody(e1.node, e1.entry); err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

// genFunctionHeader generates a new Function in Module m from the ir.Node n.
func genFunctionHeader(n *tree.Node, m *Module) (*Function, error) {
	// Function's name.
	name := n.Children[0].Data.(string)
	for _, e1 := range reservedFunctionNames {
		if e1 == name {
			return nil,
				fmt.Errorf("line %d:%d: duplicate function name %q, %s is a reserved function name",
					n.Children[0].Line, n.Children[0].Pos, name, name)
		}
	}

	// Generate return data type.
	ret, err := genType(n.Children[1])
	if err != nil {
		return nil, err
	}

	// Generate function.
	f := m.CreateFunction(name, ret)
	if err != nil {
		return nil, err
	}

	// Generate function's parameters.
	for _, e1 := range n.Children[2].Children {
		// Typed variable lists.
		typ, err := genType(e1)
		if err != nil {
			return nil, err
		}
		if typ == types.Int {
			// Integer parameter list.
			for _, e2 := range e1.Children {
				// Identifier names.
				f.CreateParam(e2.Data.(string), types.Int)
			}
		} else {
			// Float parameter list.
			for _, e2 := range e1.Children {
				// Identifier names.
				f.CreateParam(e2.Data.(string), types.Float)
			}
		}
	}
	return f, nil
}

// genFunctionBody recursively generates the instructions of the Function f starting at ir.Node n.
func genFunctionBody(n *tree.Node, f *Function) error {
	st := util.Stack{} // Scope stack.
	ls := util.Stack{} // GlobalSeq stack for loops.

	// Create new basic block for function body.
	bb := f.CreateBlock()

	// Generate function body recursively.
	if _, err := gen(bb, n, &st, &ls); err != nil {
		return err
	}
	return nil
}

// gen recursively generates LIR instructions in Block b. The returned Block is the block into which
// the next sequential instructions is to be inserted.
func gen(b *Block, n *tree.Node, st, ls *util.Stack) (*Block, error) {
	if b == nil {
		return nil, fmt.Errorf("line %d:%d: unreacheable code",
			n.Line, n.Pos)
	}
	var err error
	switch n.Typ {
	case tree.BLOCK:
		// Add new scope.
		st.Push(&symTab{
			m:       make(map[string]Value, mapSize),
			RWMutex: sync.RWMutex{},
		})
		for _, e1 := range n.Children {
			if b, err = gen(b, e1, st, ls); err != nil {
				st.Pop()
				return b, err
			}
		}
		st.Pop()
	case tree.PRINT_STATEMENT:
		if err := genPrint(b, n, st); err != nil {
			return nil, err
		}
	case tree.ASSIGNMENT_STATEMENT:
		if err := genAssign(b, n, st); err != nil {
			return nil, err
		}
	case tree.DECLARATION:
		if err := genDeclaration(b, n, st); err != nil {
			return nil, err
		}
	case tree.WHILE_STATEMENT:
		if b, err = genWhile(b, n, st, ls); err != nil {
			return nil, err
		}
	case tree.IF_STATEMENT:
		if conv, err := genIf(b, n, st, ls); err != nil {
			return nil, err
		} else {
			if conv != nil {
				// Set insertion point to the converging basic block after IF-THEN(-ELSE) statement.
				b = conv
			}
		}
	case tree.RETURN_STATEMENT:
		if err := genReturn(b, n, st); err != nil {
			return nil, err
		}
		b = nil
	case tree.NULL_STATEMENT:
		if err := genContinue(b, ls); err != nil {
			return nil, err
		}
		b = nil
	default:
		// Recursively generate LIR.
		for _, e1 := range n.Children {
			if b, err = gen(b, e1, st, ls); err != nil {
				return nil, err
			}
		}
	}
	return b, nil
}

// genDeclaration generates LIR instructions for declaring a local variable in the current scope of the
// scope stack.
func genDeclaration(b *Block, n *tree.Node, st *util.Stack) error {
	typ, err := genType(n)
	if err != nil {
		return err
	}
	if scope := st.Peek().(*symTab); scope != nil {
		for _, e1 := range n.Children[0].Children {
			name := e1.Data.(string)
			if _, ok := scope.m[name]; ok {
				return fmt.Errorf("line %d:%d: duplicate variable declaration, %q is already declared in the same scope",
					e1.Line, e1.Pos, name)
			}
			val := b.CreateDeclare(name, typ)
			scope.m[name] = val
		}
		return nil
	}
	return errors.New("compiler error: no scope on the scope stack")
}

// genDeclarationGlobal generates a globally declared variable for Module m.
func genDeclarationGlobal(n *tree.Node, m *Module) error {
	typ, err := genType(n)
	if err != nil {
		return err
	}
	for _, e1 := range n.Children[0].Children {
		// Identifier names.
		name := e1.Data.(string)

		// Check for duplicate declaration.
		m.Lock()
		if m.GetGlobalVariable(name) != nil {
			m.Unlock()
			return fmt.Errorf("duplicate declaration, global identifier %q already exists", name)
		}
		m.Unlock()

		// Create global.
		if typ == types.Int {
			m.CreateGlobalInt(name)
		} else {
			m.CreateGlobalFloat(name)
		}
	}
	return nil
}

// genAssign creates LIR assignment procedure of value calculation and store instructions. An error is returned
// if something went wrong.
func genAssign(b *Block, n *tree.Node, st *util.Stack) error {
	name := n.Children[0].Data.(string)
	c1 := n.Children[1]
	switch c1.Typ {
	case tree.INTEGER_DATA:
		return genStore(name, b.CreateConstantInt(c1.Data.(int)), b, st)
	case tree.FLOAT_DATA:
		return genStore(name, b.CreateConstantFloat(c1.Data.(float64)), b, st)
	case tree.EXPRESSION:
		if r, err := genExpression(b, c1, st); err != nil {
			return err
		} else {
			return genStore(name, r, b, st)
		}
	case tree.IDENTIFIER_DATA:
		if r, err := genLoad(c1.Data.(string), b, st); err != nil {
			return err
		} else {
			return genStore(name, r, b, st)
		}
	}
	return fmt.Errorf("line %d:%d: compiler error: unexpected node type %q",
		n.Line, n.Pos, n.Type())
}

// genExpression generates an LIR arithmetic expression defined by ir.Node n. An error is returned if something went
// wrong.
func genExpression(b *Block, n *tree.Node, st *util.Stack) (Value, error) {
	c1 := n.Children[0]
	var res Value

	if n.Data == nil {
		// Function call.
		name := c1.Data.(string)
		var target *Function

		// Find function in module.
		if target = b.f.m.GetFunction(name); target == nil {
			return res, fmt.Errorf("undeclared function %q", name)
		}

		params := target.params
		args := make([]Value, len(params))
		if len(n.Children[1].Children) == 0 && len(params) != 0 {
			return nil, fmt.Errorf("function %q expects %d parameters, got %d",
				name, len(args), len(n.Children[1].Children))
		}

		if len(n.Children[1].Children) > 0 {
			c2 := n.Children[1].Children[0] // tree.EXPRESSION_LIST. List with all arguments.
			if len(args) != len(c2.Children) {
				return nil, fmt.Errorf("function %q expects %d parameters, got %d",
					name, len(args), len(c2.Children))
			}

			for i1, e1 := range c2.Children {
				// Load argument.
				switch e1.Typ {
				case tree.INTEGER_DATA:
					args[i1] = b.CreateConstantInt(e1.Data.(int))
				case tree.FLOAT_DATA:
					args[i1] = b.CreateConstantFloat(e1.Data.(float64))
				case tree.EXPRESSION:
					if r, err := genExpression(b, e1, st); err != nil {
						return nil, err
					} else {
						args[i1] = r
					}
				case tree.IDENTIFIER_DATA:
					if r, err := genLoad(e1.Data.(string), b, st); err != nil {
						return nil, err
					} else {
						args[i1] = r
					}
				}
			}
		}
		return b.CreateFunctionCall(target, args), nil
	}
	if len(n.Children) == 2 {
		// Binary expression.
		c2 := n.Children[1]
		var op1, op2 Value

		// Operand 1.
		switch c1.Typ {
		case tree.INTEGER_DATA:
			op1 = b.CreateConstantInt(c1.Data.(int))
		case tree.FLOAT_DATA:
			op1 = b.CreateConstantFloat(c1.Data.(float64))
		case tree.EXPRESSION:
			if r, err := genExpression(b, c1, st); err != nil {
				return res, err
			} else {
				op1 = r
			}
		case tree.IDENTIFIER_DATA:
			if r, err := genLoad(c1.Data.(string), b, st); err != nil {
				return res, err
			} else {
				op1 = r
			}
		}

		// Operand 2.
		switch c2.Typ {
		case tree.INTEGER_DATA:
			op2 = b.CreateConstantInt(c2.Data.(int))
		case tree.FLOAT_DATA:
			op2 = b.CreateConstantFloat(c2.Data.(float64))
		case tree.EXPRESSION:
			if r, err := genExpression(b, c2, st); err != nil {
				return res, err
			} else {
				op2 = r
			}
		case tree.IDENTIFIER_DATA:
			if r, err := genLoad(c2.Data.(string), b, st); err != nil {
				return res, err
			} else {
				op2 = r
			}
		}

		// Operator.
		switch n.Data.(string) {
		case "+":
			res = b.CreateAdd(op1, op2)
		case "-":
			res = b.CreateSub(op1, op2)
		case "*":
			res = b.CreateMul(op1, op2)
		case "/":
			res = b.CreateDiv(op1, op2)
		case "%":
			res = b.CreateRem(op1, op2)
		case "<<":
			res = b.CreateLShift(op1, op2)
		case ">>":
			res = b.CreateRShift(op1, op2)
		case "|":
			res = b.CreateOr(op1, op2)
		case "&":
			res = b.CreateAnd(op1, op2)
		case "^":
			res = b.CreateXor(op1, op2)
		default:
			return res, fmt.Errorf("line %d:%d: operator %q not defined for VSL",
				n.Line, n.Pos, n.Data.(string))
		}
		return res, nil
	} else {
		// Unary expression.
		var op1 Value

		// Operand 1.
		switch c1.Typ {
		case tree.INTEGER_DATA:
			op1 = b.CreateConstantInt(c1.Data.(int))
		case tree.FLOAT_DATA:
			op1 = b.CreateConstantFloat(c1.Data.(float64))
		case tree.EXPRESSION:
			if r, err := genExpression(b, c1, st); err != nil {
				return nil, err
			} else {
				op1 = r
			}
		case tree.IDENTIFIER_DATA:
			if r, err := genLoad(c1.Data.(string), b, st); err != nil {
				return res, err
			} else {
				op1 = r
			}
		}

		// Operator.
		switch n.Data.(string) {
		case "-":
			res = b.CreateSub(b.CreateConstantInt(0), op1)
		case "~":
			res = b.CreateXor(b.CreateConstantInt(^0), op1)
		default:
			return res, fmt.Errorf("line %d:%d: unsupported unary operator %q",
				n.Line, n.Pos, n.Data.(string))
		}
		return res, nil
	}
}

// genReturn generates an LIR return statement with the return value being generated recursively from ir.Node n's
// children. An error is returned if something went wrong.
func genReturn(b *Block, n *tree.Node, st *util.Stack) error {
	c1 := n.Children[0]
	switch c1.Typ {
	case tree.INTEGER_DATA:
		b.CreateReturn(b.CreateConstantInt(c1.Data.(int)))
	case tree.FLOAT_DATA:
		b.CreateReturn(b.CreateConstantFloat(c1.Data.(float64)))
	case tree.EXPRESSION:
		if r, err := genExpression(b, c1, st); err != nil {
			return err
		} else {
			b.CreateReturn(r)
		}
	case tree.IDENTIFIER_DATA:
		if r, err := genLoad(c1.Data.(string), b, st); err != nil {
			return err
		} else {
			b.CreateReturn(r)
		}
	}
	return nil
}

// genRelation generates a LIR arithmetic relation. The relation loads both operands into virtual registers and performs
// an arithmetic subtraction and returns the result in a new virtual register. An error is returned if something went
// wrong.
func genRelation(b *Block, n *tree.Node, st *util.Stack) (Value, error) {
	c1 := n.Children[0]
	c2 := n.Children[1]
	var op1, op2 Value

	// Operand 1.
	switch c1.Typ {
	case tree.INTEGER_DATA:
		op1 = b.CreateConstantInt(c1.Data.(int))
	case tree.FLOAT_DATA:
		op1 = b.CreateConstantFloat(c1.Data.(float64))
	case tree.EXPRESSION:
		if r, err := genExpression(b, c1, st); err != nil {
			return nil, err
		} else {
			op1 = r
		}
	case tree.IDENTIFIER_DATA:
		if r, err := genLoad(c1.Data.(string), b, st); err != nil {
			return nil, err
		} else {
			op1 = r
		}
	}

	// Operand 2.
	switch c2.Typ {
	case tree.INTEGER_DATA:
		op2 = b.CreateConstantInt(c2.Data.(int))
	case tree.FLOAT_DATA:
		op2 = b.CreateConstantFloat(c2.Data.(float64))
	case tree.EXPRESSION:
		if r, err := genExpression(b, c2, st); err != nil {
			return nil, err
		} else {
			op2 = r
		}
	case tree.IDENTIFIER_DATA:
		if r, err := genLoad(c2.Data.(string), b, st); err != nil {
			return nil, err
		} else {
			op2 = r
		}
	}

	return b.CreateSub(op1, op2), nil
}

// genIf generates LIR IF-THEN or IF-THEN-ELSE statement. If the statement is an IF-THEN-ELSE, and both
// branches terminate their respective blocks using RETURN, the returned Block will be <nil>, else the
// returning Block is the converging block following the IF-THEN-ELSE statement.
func genIf(b *Block, n *tree.Node, st, ls *util.Stack) (*Block, error) {
	thn := b.f.CreateBlock()
	var conv *Block

	// Generate relation.
	rel, err := genRelation(b, n.Children[0], st)
	if err != nil {
		return nil, err
	}
	var op types.RelationalOperation
	switch n.Children[0].Data.(string) {
	case "=":
		op = types.Eq
	case "<":
		op = types.LessThan
	case ">":
		op = types.GreaterThan
	default:
		return nil, fmt.Errorf("undefined relation operator %q", n.Children[0].Data.(string))
	}

	// Generate branches.
	if len(n.Children) == 2 {
		// IF-THEN

		// Must create converging Block.
		conv = b.f.CreateBlock()

		// Create branch instruction.
		if rel.DataType() == types.Int {
			b.CreateConditionalBranch(op, rel, b.CreateConstantInt(0), thn, conv)
		} else {
			b.CreateConditionalBranch(op, rel, b.CreateConstantFloat(0.0), thn, conv)
		}

		// Generate THEN body.
		if ret, err := gen(thn, n.Children[1], st, ls); err != nil {
			return nil, err
		} else if ret != nil {
			// If branch body does not call return, terminate with jump to converge.
			ret.CreateBranch(conv)
		}
	} else {
		// IF-THEN-ELSE
		els := b.f.CreateBlock()

		// Create branch instruction.
		if rel.DataType() == types.Int {
			b.CreateConditionalBranch(op, rel, b.CreateConstantInt(0), thn, els)
		} else {
			b.CreateConditionalBranch(op, rel, b.CreateConstantFloat(0.0), thn, els)
		}

		// Generate THEN body.
		if ret, err := gen(thn, n.Children[1], st, ls); err != nil {
			return nil, err
		} else if ret != nil {
			// If branch body does not call return, terminate with jump to converge.
			conv = b.f.CreateBlock()
			ret.CreateBranch(conv)
		}

		// Generate ELSE body.
		if ret, err := gen(els, n.Children[2], st, ls); err != nil {
			return nil, err
		} else if ret != nil {
			// If branch body does not call return, terminate with jump to converge.
			if conv == nil {
				conv = b.f.CreateBlock()
			}
			ret.CreateBranch(conv)
		}
	}
	return conv, nil
}

// genWhile generates LIR for a while statement and its body.
func genWhile(b *Block, n *tree.Node, st, ls *util.Stack) (*Block, error) {
	head := b.f.CreateBlock()
	body := b.f.CreateBlock()
	conv := b.f.CreateBlock()

	// Push head to lseq stack.
	ls.Push(head)

	// Generate relation and branch to check if to jump to while body or converge.
	b.CreateBranch(head)
	b = head
	rel, err := genRelation(b, n.Children[0], st)
	if err != nil {
		return nil, err
	}
	var op types.RelationalOperation
	switch n.Children[0].Data.(string) {
	case "=":
		op = types.Eq
	case "<":
		op = types.LessThan
	case ">":
		op = types.GreaterThan
	default:
		return nil, fmt.Errorf("undefined relation operator %q", n.Children[0].Data.(string))
	}
	if rel.DataType() == types.Int {
		b.CreateConditionalBranch(op, rel, b.CreateConstantInt(0), body, conv)
	} else {
		b.CreateConditionalBranch(op, rel, b.CreateConstantFloat(0.0), body, conv)
	}

	// Create while body.
	if ret, err := gen(body, n.Children[1], st, ls); err != nil {
		return nil, err
	} else if ret != nil {
		// Jump back to loop head if while statement doesn't call function return.
		ret.CreateBranch(head)
	}

	return conv, nil
}

// genContinue generates an LIR continue statement in Block b.
func genContinue(b *Block, ls *util.Stack) error {
	var l interface{}
	if l = ls.Peek(); l == nil {
		return errors.New("continue without while-statement")
	}
	b.CreateBranch(l.(*Block))
	return nil
}

// genPrint generates LIR print instructions using calls to Linux standard C library function printf. An error is
// returned if something went wrong.
func genPrint(b *Block, n *tree.Node, st *util.Stack) error {
	m := b.f.m
	args := make([]Value, len(n.Children[0].Children))

	// Build printf arguments.
	for i1, e1 := range n.Children[0].Children {
		switch e1.Typ {
		case tree.STRING_DATA:
			s := m.CreateGlobalString(e1.Data.(string))
			load := b.CreateLoad(s)
			args[i1] = load
		case tree.INTEGER_DATA:
			c := b.CreateConstantInt(e1.Data.(int))
			args[i1] = c
		case tree.FLOAT_DATA:
			s := m.CreateGlobalString(fmt.Sprintf("%x", e1.Data.(float64)))
			load := b.CreateLoad(s)
			args[i1] = load
		case tree.EXPRESSION:
			val, err := genExpression(b, e1, st)
			if err != nil {
				return err
			}
			args[i1] = val
		case tree.IDENTIFIER_DATA:
			val, err := genLoad(e1.Data.(string), b, st)
			if err != nil {
				return err
			}
			args[i1] = val
		default:
			return fmt.Errorf("print statement expected node of type STRING, INTEGER, FLOAT, EXPRESSION or "+
				"IDENTIFIER, got %s", e1.Type())
		}
	}

	// Prepend format and create print.
	b.CreatePrint(args)

	return nil
}

// genLoad generates a load of the named variable. The local scopes are searched first, followed by function parameters,
// and lastly global variables. An error is returned if something went wrong.
func genLoad(name string, b *Block, st *util.Stack) (Value, error) {
	// Start by searching through local scopes, inner-most to outer-most, first.
	for i1 := 1; i1 <= st.Size(); i1++ {
		if scope := st.Get(i1).(*symTab); scope != nil {
			if v, ok := scope.m[name]; ok {
				ld := b.CreateLoad(v)
				return ld, nil
			}
		} else {
			return nil, errors.New("compiler error: scope from scope stack is <nil>")
		}
	}

	// Search function parameters second.
	if v := b.f.GetParam(name); v != nil {
		ld := b.CreateLoad(v)
		return ld, nil
	}

	// Lastly, try searching global variables.
	if v := b.f.m.GetGlobalVariable(name); v != nil {
		return b.CreateLoad(v), nil
	}

	return nil, fmt.Errorf("undeclared variable %q", name)
}

// genStore generates a store to the named variable dst. Variables are looked up by local scopes first, function
// parameters second and global variables last. An error is returned if something went wrong.
func genStore(dst string, src Value, b *Block, st *util.Stack) error {
	// Start by searching local scopes first, top-to-bottom.
	for i1 := 1; i1 <= st.Size(); i1++ {
		if scope := st.Get(i1).(*symTab); scope != nil {
			if v, ok := scope.m[dst]; ok {
				b.CreateStore(src, v)
				return nil
			}
		} else {
			return errors.New("compiler error: scope from scope stack is <nil>")
		}
	}

	// Check function parameters next.
	if v := b.f.GetParam(dst); v != nil {
		b.CreateStore(src, v)
		return nil
	}

	// Lastly, check global variables.
	if v := b.f.m.GetGlobalVariable(dst); v != nil {
		b.CreateStore(src, v)
		return nil
	}
	return nil
}

// genType takes an ir.TYPED_VARIABLE_LIST or ir.DECLARATION and returns the type of the data variable(s).
func genType(n *tree.Node) (res types.DataType, _ error) {
	if n == nil {
		return types.Int, errors.New("cannot generate LIR type, node is <nil>")
	}
	if n.Data == nil {
		return res, fmt.Errorf("line %d:%d: syntax tree node of type %s doesn't carry data",
			n.Line, n.Pos, n.Type())
	}
	switch n.Data.(string) {
	case "int":
		return types.Int, nil
	case "float":
		return types.Float, nil
	default:
		return res, fmt.Errorf("expected DECLARATION or TYPED_VARIABLE_LIST, got %s",
			n.Type())
	}
}
