// Package llvm provides means to transform the Go syntax tree into LLVM IR for the system installed LLVM
// runtime.
package llvm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

import (
	"tinygo.org/x/go-llvm"
)

import (
	ast "vslc/src/ir"
	"vslc/src/util"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// symTab is a symbol table that implements a hash map and a read/write mutex for thread safe access.
type symTab struct {
	m map[string]llvm.Value
	sync.RWMutex
}

// ---------------------
// ----- Constants -----
// ---------------------

const mapSize = 16 // Predefined size for a decently sized symbol table hash table.

// -------------------
// ----- globals -----
// -------------------

var stringPrefix = "L_STR" // Prefix all global strings with this prefix.
var i = llvm.Int64Type()   // i defines the integer type for the target architecture.
var f = llvm.DoubleType()  // f defines the float type for the target architecture.

// globals is the global symbol table that keeps track of globally declared variables and functions for easy access.
var globals symTab

// reservedFunctionNames defines a list of function names that cannot be assigned to VSL functions.
var reservedFunctionNames = []string{
	"main",
	"printf",
	"atof",
	"atoi",
}

// ---------------------
// ----- functions -----
// ---------------------

// GenLLVM generates LLVM IR from the root ast.Node of the syntax tree.
func GenLLVM(opt util.Options, root *ast.Node) error {
	if root == nil {
		return errors.New("syntax tree node is <nil>")
	}
	if len(root.Children) < 1 {
		return errors.New("syntax tree node has no children")
	}

	if opt.TargetArch == util.Riscv32 {
		i = llvm.Int32Type()
		f = llvm.FloatType()
	}

	// funcWrapper wraps an ast.Node pointer and an LLVM function definition.
	type funcWrapper struct {
		ll   llvm.Value // LLVM function definition.
		node *ast.Node  // Syntax tree node pointer of function.
	}

	globals.m = make(map[string]llvm.Value, mapSize)
	ctx := llvm.NewContext()
	defer ctx.Dispose()

	// Builder constructs LLVM IR instructions on basic block level.
	b := ctx.NewBuilder()
	defer b.Dispose()

	// Set module name equal file name without file extension.
	m := ctx.NewModule(filepath.Base(opt.Src))
	defer m.Dispose()

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

		cerr := make(chan error, t) // One buffer per worker thread.
		errs := make([]error, 0, l) // Pre-allocate one error per global definition.

		// Error listener.
		go func(cerr chan error, errs *[]error) {
			defer close(cerr)
			for {
				err := <-cerr
				if err == nil {
					return
				} else {
					*errs = append(*errs, err)
				}
			}
		}(cerr, &errs)

		funcs := make([]funcWrapper, 0, len(root.Children))
		cfunc := make(chan []funcWrapper, t)

		// Generate global variables and function declarations.
		for i1 := 0; i1 < t; i1++ {
			// Spawn t threads.
			if i1 < res {
				// This thread should do one extra residual job.
				end++
			}
			go func(start, end int, cerr chan error, cfunc chan []funcWrapper, wg *sync.WaitGroup) {
				defer wg.Done()
				funcs := make([]funcWrapper, 0, end-start)
				for _, e1 := range root.Children[start:end] {
					if e1.Typ == ast.FUNCTION {
						if fun, err := genFuncHeader(m, e1); err != nil {
							cerr <- err
						} else {
							funcs = append(funcs, funcWrapper{ll: fun, node: e1})
						}
					} else if e1.Typ == ast.DECLARATION {
						if err := genDeclarationGlobal(m, e1); err != nil {
							cerr <- err
						}
					} else {
						cerr <- fmt.Errorf("line %d:%d: expected FUNCTION or DECLARATION, got %s",
							e1.Line, e1.Pos, e1.Type())
					}
				}
				cfunc <- funcs
			}(start, end, cerr, cfunc, &wg)

			start = end
			end += n
		}

		// Wait for generation of function declarations and global variables.
		wg.Wait()
		close(cfunc)
		for e1 := range cfunc {
			funcs = append(funcs, e1...)
		}

		// Stop error listener.
		cerr <- nil

		// Check for errors.
		if len(errs) > 0 {
			for _, e1 := range errs {
				fmt.Println(e1)
			}
			return errors.New("multiple errors during parallel compilation")
		}

		// Calculate worker threads for function body generation.
		l = len(funcs)
		t = opt.Threads
		if t > l {
			t = l
		}
		n = l / t
		res = l % t
		start = 0
		end = n

		cerr = make(chan error, t) // One buffer per worker thread.
		errs = make([]error, 0, l) // Pre-allocate one error per global definition.

		wg.Add(t)
		// Generate function bodies.
		for i1 := 0; i1 < t; i1++ {
			// Spawn t threads.
			if i1 < res {
				// This thread should do one extra residual job.
				end++
			}

			go func(start, end int, wg *sync.WaitGroup, cerr chan error) {
				defer wg.Done()
				// Give each thread its own builder, else there will be multiple threads writing different functions,
				// interchanging basic blocks concurrently.
				b := ctx.NewBuilder()
				defer b.Dispose()
				for _, e1 := range funcs[start:end] {
					if err := genFuncBody(b, m, e1.ll, e1.node); err != nil {
						cerr <- err
					}
				}
			}(start, end, &wg, cerr)

			start = end
			end += n
		}

		// Wait for generation of function bodies.
		wg.Wait()
	} else {
		// Sequential.
		funcs := make([]funcWrapper, 0, len(root.Children)) // Pre-allocate sufficient space for functions of root.
		for _, e1 := range root.Children {
			if e1.Typ == ast.FUNCTION {
				if fun, err := genFuncHeader(m, e1); err != nil {
					return err
				} else {
					funcs = append(funcs, funcWrapper{ll: fun, node: e1})
				}
			} else if e1.Typ == ast.DECLARATION {
				// Global variable declaration.
				if err := genDeclarationGlobal(m, e1); err != nil {
					return err
				}
			} else {
				return fmt.Errorf("expected node of type FUNCTION or DECLARATION, got %s", e1.Type())
			}
		}
		for _, e1 := range funcs {
			if err := genFuncBody(b, m, e1.ll, e1.node); err != nil {
				return err
			}
		}
	}
	if err := genMain(b, m, root); err != nil {
		return err
	}

	if opt.Verbose {
		fmt.Println("LLVM IR:")
		m.Dump()
	}

	// Initialise LLVM code generation.
	llvm.InitializeAllTargetInfos()
	llvm.InitializeAllTargetMCs()
	llvm.InitializeAllAsmParsers()
	llvm.InitializeAllAsmPrinters()

	// Construct target triple.
	t, tt, err := genTargetTriple(&opt)
	if err != nil {
		return err
	}

	// Configure hardware properties for target.
	var cpu string
	switch opt.TargetArch {
	case util.Riscv64:
		cpu = "generic-rv64" // TODO: Causes LLVM to crash.
	case util.Riscv32:
		cpu = "generic-rv32"
	default:
		cpu = "generic"
	}
	features := "" // Ignore extra features for this simple compiler.

	tm := t.CreateTargetMachine(tt, cpu, features,
		llvm.CodeGenLevelNone,
		llvm.RelocDefault,
		llvm.CodeModelDefault)
	defer tm.Dispose()

	td := tm.CreateTargetData()
	defer td.Dispose()

	m.SetDataLayout(td.String())
	m.SetTarget(tm.Triple())

	// Set target file type.
	ft := llvm.ObjectFile

	// Compile target and store in memory.
	buf, err := tm.EmitToMemoryBuffer(m, ft)
	if err != nil {
		return err
	} else if buf.IsNil() {
		return errors.New("could not emit compiled code to memory")
	}

	// Open/create file and write compiled code to output file.
	var out string
	if len(opt.Out) > 0 {
		out = opt.Out
	} else {
		out = fmt.Sprintf("./%s.o", strings.TrimSuffix(filepath.Base(opt.Src), filepath.Ext(opt.Src)))
	}

	// Write to file sequentially.
	if fd, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0755); err != nil {
		return err
	} else {
		defer func() {
			if err := fd.Close(); err != nil {
				fmt.Println(err)
			}
		}()
		if _, err2 := fd.Write(buf.Bytes()); err2 != nil {
			return err
		}
	}

	return nil
}

// gen recursively generates LLVM IR by iterating the sub-tree of ast.Node n.
//
// Parameters:
//	b	-	LLVM Builder.
//	m	-	Current LLVM module.
//	fun	-	Current LLVM function being generated.
//	n	-	Current node in syntax tree being generated.
//	st	-	Scope stack for looking up correct variables with respect to definition scopes.
//	ls	-	GlobalSeq stack for continuing/breaking correct loops.
//
// Returns:
//
// bool		-	Set true if the sub-tree generated a RETURN statement which terminates the current basic block.
// error	-	<nil> if everything went ok, error message if something went wrong.
func gen(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st, ls *util.Stack) (bool, error) {
	ret := false
	var err error
	switch n.Typ {
	case ast.BLOCK:
		// Add new scope.
		st.Push(&symTab{
			m:       make(map[string]llvm.Value, mapSize),
			RWMutex: sync.RWMutex{},
		})
		for _, e1 := range n.Children {
			if ret, err = gen(b, m, fun, e1, st, ls); err != nil {
				st.Pop()
				return ret, err
			}
		}
		st.Pop()
	case ast.PRINT_STATEMENT:
		if err = genPrint(b, m, fun, n, st); err != nil {
			return ret, err
		}
	case ast.ASSIGNMENT_STATEMENT:
		if err = genAssign(b, m, fun, n, st); err != nil {
			return ret, err
		}
	case ast.DECLARATION:
		if err = genDeclaration(b, n, st); err != nil {
			return ret, err
		}
	case ast.WHILE_STATEMENT:
		if err = genWhile(b, m, fun, n, st, ls); err != nil {
			return ret, err
		}
	case ast.IF_STATEMENT:
		if err = genIf(b, m, fun, n, st, ls); err != nil {
			return ret, err
		}
	case ast.NULL_STATEMENT:
		if err = genContinue(b, ls); err != nil {
			return ret, err
		}
	case ast.RETURN_STATEMENT:
		if err = genReturn(b, m, fun, n, st); err != nil {
			return true, err
		}
		return true, nil
	default:
		// Recursively generate LLVM IR.
		for _, e1 := range n.Children {
			if ret, err = gen(b, m, fun, e1, st, ls); err != nil {
				return ret, err
			}
		}
	}
	return ret, nil
}

// genFuncHeader generates the LLVM IR declaration of a function. The declaration defines a function's name, parameters
// and return type.
func genFuncHeader(m llvm.Module, n *ast.Node) (llvm.Value, error) {
	if n.Typ != ast.FUNCTION {
		return llvm.Value{}, fmt.Errorf("expected node type FUNCTION, got %s", n.String())
	}

	// Function's name.
	name := n.Children[0].Data.(string)
	for _, e1 := range reservedFunctionNames {
		if e1 == name {
			return llvm.Value{},
				fmt.Errorf("duplicate function name %q, %s is a reserved function name",
					name, name)
		}
	}

	// Define function's return type.
	ret, err := genType(n.Children[1])
	if err != nil {
		return llvm.Value{}, err
	}

	// Function's parameters.
	atyp := make([]llvm.Type, 0, 8) // Assume no more than 8 parameters.
	aname := make([]string, 0, 8)   // Assume no more than 8 parameters.
	for _, e1 := range n.Children[2].Children {
		// Typed variable list.
		typ, err := genType(n.Children[1])
		if err != nil {
			return llvm.Value{}, err
		}
		for _, e2 := range e1.Children {
			// Identifiers.
			atyp = append(atyp, typ)
			aname = append(aname, e2.Data.(string))
		}
	}
	ftyp := llvm.FunctionType(ret, atyp, false) // TODO: Sigseg during parallel.

	// Used mutex for parallel thread safety.
	globals.Lock()
	defer globals.Unlock()

	// Check for duplicate declarations.
	if val, ok := globals.m[name]; ok {
		if !val.IsAFunction().IsNil() {
			return llvm.Value{}, fmt.Errorf("duplicate declaration, function %q already declared", name)
		}
		return llvm.Value{}, fmt.Errorf("duplicate declaration, global identifer %q already declared", name)
	}

	// Declare function in module m.
	fun := llvm.AddFunction(m, name, ftyp) // TODO: sigseg during parallel run.

	// Set parameter names.
	for i1, e1 := range fun.Params() {
		e1.SetName(aname[i1])
	}

	// Add function to global symbol table.
	globals.m[name] = fun
	return fun, nil
}

// genFuncBody generates the LLVM IR definition fo a function. A function definition defines a function's executing
// instructions that's run when the function is called.
func genFuncBody(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node) error {
	st := util.Stack{} // Scope stack.
	ls := util.Stack{} // GlobalSeq stack for loops.

	// Create new basic block for function body.
	bb := llvm.AddBasicBlock(fun, "")
	b.SetInsertPointAtEnd(bb)

	// Allocate memory for the function's parameters.
	fscope := symTab{
		m:       make(map[string]llvm.Value),
		RWMutex: sync.RWMutex{},
	}
	for _, e1 := range fun.Params() {
		alloc := b.CreateAlloca(e1.Type(), "") // Allocate stack memory for parameter e1. TODO: Sigseg during parallel.
		b.CreateStore(e1, alloc)               // Store the value passed to parameter e1 to stack.
		fscope.Lock()
		fscope.m[e1.Name()] = alloc            // Put variable holding parameter e1 on scope stack.
		fscope.Unlock()
	}

	// Push the function parameters to the bottom of the stack.
	st.Push(&fscope)
	defer st.Pop()

	// Generate function body recursively.
	if _, err := gen(b, m, fun, n, &st, &ls); err != nil {
		return err
	}
	return nil
}

// genExpression generates LLVM IR from the expression ast.Node n.
func genExpression(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st *util.Stack) (llvm.Value, error) {
	c1 := n.Children[0]
	var res llvm.Value

	if n.Data == nil {
		// Function call.
		name := c1.Data.(string)
		var target llvm.Value

		// Find function in module.
		if target = m.NamedFunction(name); target.IsAFunction().IsNil() {
			return res, fmt.Errorf("undeclared function %q", name)
		}

		params := target.Params()
		args := make([]llvm.Value, len(params))

		if len(n.Children[1].Children) == 0 && len(params) != 0 {
			return llvm.Value{}, fmt.Errorf("function %q expects %d parameters, got %d",
				name, len(args), len(n.Children[1].Children))
		}
		if len(n.Children[1].Children) > 0 {

			c2 := n.Children[1].Children[0] // ast.EXPRESSION_LIST. List with all arguments.
			if len(args) != len(c2.Children) {
				return llvm.Value{}, fmt.Errorf("function %q expects %d parameters, got %d",
					name, len(args), len(c2.Children))
			}

			for i1, e1 := range c2.Children {
				// Load argument.
				switch e1.Typ {
				case ast.INTEGER_DATA:
					args[i1] = llvm.ConstInt(i, uint64(e1.Data.(int)), true)
				case ast.FLOAT_DATA:
					args[i1] = llvm.ConstFloat(f, e1.Data.(float64))
				case ast.EXPRESSION:
					if r, err := genExpression(b, m, fun, e1, st); err != nil {
						return llvm.Value{}, err
					} else {
						args[i1] = r
					}
				case ast.IDENTIFIER_DATA:
					if r, err := genLoad(e1.Data.(string), b, m, fun, st); err != nil {
						return llvm.Value{}, err
					} else {
						args[i1] = r
					}
				}
			}
		}

		return b.CreateCall(target, args, ""), nil
	}
	if len(n.Children) == 2 {
		// Binary expression.
		c2 := n.Children[1]
		var op1, op2 llvm.Value

		// Operand 1.
		switch c1.Typ {
		case ast.INTEGER_DATA:
			op1 = llvm.ConstInt(i, uint64(c1.Data.(int)), true)
		case ast.FLOAT_DATA:
			op1 = llvm.ConstFloat(f, c1.Data.(float64))
		case ast.EXPRESSION:
			if r, err := genExpression(b, m, fun, c1, st); err != nil {
				return res, err
			} else {
				op1 = r
			}
		case ast.IDENTIFIER_DATA:
			if r, err := genLoad(c1.Data.(string), b, m, fun, st); err != nil {
				return res, err
			} else {
				op1 = r
			}
		}

		// Operand 2.
		switch c2.Typ {
		case ast.INTEGER_DATA:
			op2 = llvm.ConstInt(i, uint64(c2.Data.(int)), true)
		case ast.FLOAT_DATA:
			op2 = llvm.ConstFloat(f, c2.Data.(float64))
		case ast.EXPRESSION:
			if r, err := genExpression(b, m, fun, c2, st); err != nil {
				return res, err
			} else {
				op2 = r
			}
		case ast.IDENTIFIER_DATA:
			if r, err := genLoad(c2.Data.(string), b, m, fun, st); err != nil {
				return res, err
			} else {
				op2 = r
			}
		}

		// Operator.
		switch n.Data.(string) {
		case "+":
			res = b.CreateAdd(op1, op2, "")
		case "-":
			res = b.CreateSub(op1, op2, "")
		case "*":
			res = b.CreateMul(op1, op2, "")
		case "/":
			res = b.CreateSDiv(op1, op2, "")
		case "%":
			res = b.CreateSRem(op1, op2, "")
		case "<<":
			res = b.CreateShl(op1, op2, "")
		case ">>":
			res = b.CreateLShr(op1, op2, "")
		case "|":
			res = b.CreateOr(op1, op2, "")
		case "&":
			res = b.CreateAnd(op1, op2, "")
		case "^":
			res = b.CreateXor(op1, op2, "")
		default:
			return res, fmt.Errorf("operator %q not defined for VSL", n.Data.(string))
		}
		return res, nil
	} else {
		// Unary expression.
		var op1 llvm.Value

		// Operand 1.
		switch c1.Typ {
		case ast.INTEGER_DATA:
			op1 = llvm.ConstInt(i, uint64(c1.Data.(int)), true)
		case ast.FLOAT_DATA:
			op1 = llvm.ConstFloat(f, c1.Data.(float64))
		case ast.EXPRESSION:
			if r, err := genExpression(b, m, fun, c1, st); err != nil {
				return llvm.Value{}, err
			} else {
				op1 = r
			}
		case ast.IDENTIFIER_DATA:
			if r, err := genLoad(c1.Data.(string), b, m, fun, st); err != nil {
				return res, err
			} else {
				op1 = r
			}
		}

		// Operator.
		switch n.Data.(string) {
		case "-":
			res = b.CreateSub(llvm.ConstInt(i, 0, false), op1, "")
		case "~":
			res = b.CreateXor(llvm.ConstInt(i, ^uint64(0), false), op1, "")
		default:
			return res, fmt.Errorf("line %d:%d: unsupported unary operator %q",
				n.Line, n.Pos, n.Data.(string))
		}
		return res, nil
	}
}

// genDeclaration generates LLVM IR that declares one or many new local variables in the inner-most scope.
func genDeclaration(b llvm.Builder, n *ast.Node, st *util.Stack) error {
	typ, err := genType(n)
	if err != nil {
		return fmt.Errorf("genDeclaration(): %s. Node was %s", err, n.String())
	}

	if scope := st.Peek().(*symTab); scope != nil {
		for _, e1 := range n.Children[0].Children {
			name := e1.Data.(string)
			if _, ok := scope.m[name]; ok {
				return fmt.Errorf("duplicate variable declaration, %q is already declared in the same scope",
					name)
			}
			val := b.CreateAlloca(typ, name) // TODO: Sigseg during parallel.
			scope.m[name] = val
		}
		return nil
	}
	return errors.New("compiler error, no scope on the scope stack")
}

// genDeclarationGlobal generates LLVM IR that declares a global variable and adds it to the global symbol table.
func genDeclarationGlobal(m llvm.Module, n *ast.Node) error {
	typ, err := genType(n)
	if err != nil {
		return fmt.Errorf("genDeclarationGlobal(): %s. Node was %s", err, n.String())
	}
	for _, e1 := range n.Children[0].Children {
		// Identifier names.
		name := e1.Data.(string)

		// Look in global symbol table for duplicate declaration.
		globals.Lock()
		if _, ok := globals.m[name]; ok {
			globals.Unlock()
			return fmt.Errorf("duplicate declaration, identifier %q already exists", name)
		}

		// Create global variable and add it to the global symbol table.
		g := llvm.AddGlobal(m, typ, name)
		g.SetInitializer(g)
		globals.m[name] = g
		globals.Unlock()
	}
	return nil
}

// genAssign generates LLVM IR that assigns a value to an existing variable.
func genAssign(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st *util.Stack) error {
	name := n.Children[0].Data.(string)
	c1 := n.Children[1]

	switch c1.Typ {
	case ast.INTEGER_DATA:
		cnst := llvm.ConstInt(i, uint64(c1.Data.(int)), true)
		if err := genStore(cnst, name, b, m, fun, st); err != nil {
			return err
		}
	case ast.FLOAT_DATA:
		cnst := llvm.ConstFloat(f, c1.Data.(float64))
		if err := genStore(cnst, name, b, m, fun, st); err != nil {
			return err
		}
	case ast.EXPRESSION:
		if tmp1, err := genExpression(b, m, fun, c1, st); err != nil {
			return err
		} else {
			if err = genStore(tmp1, name, b, m, fun, st); err != nil {
				return err
			}
		}
	case ast.IDENTIFIER_DATA:
		if src, err := genLoad(c1.Data.(string), b, m, fun, st); err != nil {
			return err
		} else {
			if err = genStore(src, name, b, m, fun, st); err != nil {
				return err
			}
		}
	}
	return nil
}

// genReturn generates LLVM IR that terminates the current basic block with a return statement.
func genReturn(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st *util.Stack) error {
	c1 := n.Children[0]
	switch c1.Typ {
	case ast.INTEGER_DATA:
		b.CreateRet(llvm.ConstInt(i, uint64(c1.Data.(int)), true))
	case ast.FLOAT_DATA:
		b.CreateRet(llvm.ConstFloat(f, c1.Data.(float64)))
	case ast.EXPRESSION:
		if val, err := genExpression(b, m, fun, c1, st); err != nil {
			return err
		} else {
			b.CreateRet(val)
		}
	case ast.IDENTIFIER_DATA:
		if val, err := genLoad(c1.Data.(string), b, m, fun, st); err != nil {
			return err
		} else {
			b.CreateRet(val)
		}
	}
	return nil
}

// genPrint generates LLVM IR that calls printf to print constants, identifiers or expressions.
func genPrint(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st *util.Stack) error {
	var pf llvm.Value

	// Check if printf is defined.
	globals.Lock()
	if pf = m.NamedFunction("printf"); pf.IsAFunction().IsNil() {
		pf = genPrintf(m)
	}
	globals.Unlock()

	// Build printf arguments.
	args := make([]llvm.Value, len(n.Children[0].Children)+1)
	sb := strings.Builder{}
	for i1, e1 := range n.Children[0].Children {
		switch e1.Typ {
		case ast.STRING_DATA:
			sb.WriteString("%s")
			globals.Lock()
			s := b.CreateGlobalStringPtr(e1.Data.(string), stringPrefix)
			globals.Unlock()
			args[i1+1] = s
		case ast.INTEGER_DATA:
			sb.WriteString("%d")
			args[i1+1] = llvm.ConstInt(i, uint64(e1.Data.(int)), true)
		case ast.FLOAT_DATA:
			sb.WriteString("%f")
			args[i1+1] = llvm.ConstFloat(f, e1.Data.(float64))
		case ast.EXPRESSION:
			if val, err := genExpression(b, m, fun, e1, st); err != nil {
				return err
			} else {
				if val.Type() == i {
					sb.WriteString("%d")
				} else {
					sb.WriteString("%f")
				}
				args[i1+1] = val
			}
		case ast.IDENTIFIER_DATA:
			if val, err := genLoad(e1.Data.(string), b, m, fun, st); err != nil {
				return err
			} else {
				if val.Type() == i {
					sb.WriteString("%d")
				} else {
					sb.WriteString("%f")
				}
				args[i1+1] = val
			}
		default:
			return fmt.Errorf("print statement expected node of type STRING, INTEGER, FLOAT, EXPRESSION or "+
				"IDENTIFIER, got %s", e1.Type())
		}

		// Add whitespace between print items.
		if i1 < len(n.Children[0].Children)-1 {
			sb.WriteRune(' ')
		}
	}

	// Add newline to string format.
	sb.WriteRune('\n')

	// Construct format string and store in globals.
	globals.Lock()
	frmt := b.CreateGlobalStringPtr(sb.String(), stringPrefix)
	globals.Unlock()

	// Prepend format string to arguments.
	args[0] = frmt

	// Call printf.
	b.CreateCall(pf, args, "")

	return nil
}

// genRelation generates LLVM IR that compares two operands with the given relation.
func genRelation(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st *util.Stack) (llvm.Value, error) {
	c1 := n.Children[0]
	c2 := n.Children[1]
	var op1, op2 llvm.Value

	// Operand 1.
	switch c1.Typ {
	case ast.INTEGER_DATA:
		op1 = llvm.ConstInt(i, uint64(c1.Data.(int)), true)
	case ast.FLOAT_DATA:
		op1 = llvm.ConstFloat(f, c1.Data.(float64))
	case ast.EXPRESSION:
		if r, err := genExpression(b, m, fun, c1, st); err != nil {
			return llvm.Value{}, err
		} else {
			op1 = r
		}
	case ast.IDENTIFIER_DATA:
		if r, err := genLoad(c1.Data.(string), b, m, fun, st); err != nil {
			return llvm.Value{}, err
		} else {
			op1 = r
		}
	}

	// Operand 2.
	switch c2.Typ {
	case ast.INTEGER_DATA:
		op2 = llvm.ConstInt(i, uint64(c2.Data.(int)), true)
	case ast.FLOAT_DATA:
		op2 = llvm.ConstFloat(f, c2.Data.(float64))
	case ast.EXPRESSION:
		if r, err := genExpression(b, m, fun, c2, st); err != nil {
			return llvm.Value{}, err
		} else {
			op2 = r
		}
	case ast.IDENTIFIER_DATA:
		if r, err := genLoad(c2.Data.(string), b, m, fun, st); err != nil {
			return llvm.Value{}, err
		} else {
			op2 = r
		}
	}

	// Operator.
	switch n.Data.(string) {
	case "=":
		if op1.Type() == i {
			return b.CreateICmp(llvm.IntEQ, op1, op2, ""), nil
		} else {
			return b.CreateFCmp(llvm.FloatOEQ, op1, op2, ""), nil
		}
	case "<":
		if op1.Type() == i {
			return b.CreateICmp(llvm.IntSLT, op1, op2, ""), nil
		} else {
			return b.CreateFCmp(llvm.FloatOLT, op1, op2, ""), nil
		}
	case ">":
		if op1.Type() == i {
			return b.CreateICmp(llvm.IntSGT, op1, op2, ""), nil
		} else {
			return b.CreateFCmp(llvm.FloatOGT, op1, op2, ""), nil
		}
	default:
		return llvm.Value{}, fmt.Errorf("undefined relation operator %q", n.Children[0].Data.(string))
	}
}

// genIf generates LLVM IR for either IF-THEN or IF-THEN-ELSE statements.
func genIf(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st, ls *util.Stack) error {
	// Generate relation.
	var conv llvm.BasicBlock
	var val llvm.Value
	var err error
	if val, err = genRelation(b, m, fun, n.Children[0], st); err != nil {
		return err
	}

	// Set up new basic block(s).
	thn := llvm.AddBasicBlock(fun, "")

	if len(n.Children) == 2 {
		// IF-THEN.
		conv = llvm.AddBasicBlock(fun, "")

		// Generate branch.
		b.CreateCondBr(val, thn, conv)

		// Generate THEN.
		b.SetInsertPointAtEnd(thn)
		for _, e1 := range n.Children[1].Children {
			if ret, err := gen(b, m, fun, e1, st, ls); err != nil {
				return err
			} else if !ret {
				b.CreateBr(conv)
			}
		}
		b.SetInsertPointAtEnd(conv)
	} else {
		// IF-THEN-ELSE.
		var retA, retB bool
		els := llvm.AddBasicBlock(fun, "")

		// Generate branch.
		b.CreateCondBr(val, thn, els)

		// Generate THEN.
		b.SetInsertPointAtEnd(thn)
		if retA, err = gen(b, m, fun, n.Children[1], st, ls); err != nil {
			return err
		}

		if !retA {
			conv = llvm.AddBasicBlock(fun, "")
			b.CreateBr(conv)
		}

		// Generate ELSE.
		b.SetInsertPointAtEnd(els)
		if retB, err = gen(b, m, fun, n.Children[2], st, ls); err != nil {
			return err
		}

		if !retB {
			if conv.IsNil() {
				conv = llvm.AddBasicBlock(fun, "")
			}
			b.CreateBr(conv)
		}

		// Check if either branch converges. If they do, start insert point at converging basic block.
		if !conv.IsNil() {
			b.SetInsertPointAtEnd(conv)
		}
	}
	return nil
}

// genWhile generates LLVM IR for loops of type WHILE(relation) DO.
func genWhile(b llvm.Builder, m llvm.Module, fun llvm.Value, n *ast.Node, st, ls *util.Stack) error {
	head := llvm.AddBasicBlock(fun, "")
	body := llvm.AddBasicBlock(fun, "")
	conv := llvm.AddBasicBlock(fun, "")

	// Push head to label stack for CONTINUE statement.
	ls.Push(head)

	// Generate relation and branch.
	b.CreateBr(head)
	b.SetInsertPointAtEnd(head)
	rel, err := genRelation(b, m, fun, n.Children[0], st)
	if err != nil {
		return err
	}
	b.CreateCondBr(rel, body, conv)

	// Generate WHILE body.
	b.SetInsertPointAtEnd(body)

	if ret, err := gen(b, m, fun, n.Children[1], st, ls); err != nil {
		return err
	} else if !ret {
		// Jump back to loop head.
		b.CreateBr(head)
	}

	// Converge.
	b.SetInsertPointAtEnd(conv)

	// Pop label stack.
	ls.Pop()
	return nil
}

// genContinue generates LLVM IR for a continue statement for loops.
func genContinue(b llvm.Builder, ls *util.Stack) error {
	var l interface{}
	if l = ls.Peek(); l == nil {
		return errors.New("label stack is empty")
	}

	b.CreateBr(l.(llvm.BasicBlock))
	return nil
}

// genStore generates LLVM IR store instruction that stores the src llvm.Value in the requested identifier with
// given name.
func genStore(src llvm.Value, name string, b llvm.Builder, m llvm.Module, fun llvm.Value, st *util.Stack) error {
	// Check local scopes. Function parameters are on the bottom of the scope stack.
	for i1 := 1; i1 <= st.Size(); i1++ {
		if symtab := st.Get(i1).(*symTab); symtab != nil {
			if dst, ok := symtab.m[name]; ok {
				if src.Type() != dst.Type() {
					if dst.Type() == i {
						src = b.CreateSIToFP(src, i, "")
					} else {
						src = b.CreateSIToFP(src, f, "")
					}
				}
				_ = b.CreateStore(src, dst)
				return nil
			}
		}
	}

	// Check global scope.
	if dst := m.NamedGlobal(name); dst.IsNil() {
		return fmt.Errorf("undeclared variable %q", name)
	} else {
		if src.Type() != dst.Type().ElementType() {
			if dst.Type() == i {
				src = b.CreateSIToFP(src, i, "")
			} else {
				src = b.CreateSIToFP(src, f, "")
			}
		}
		_ = b.CreateStore(src, dst)
		return nil
	}
}

// genLoad generates LLVM IR load instruction for the requested identifier with given name and returns the
// resulting llvm.Value.
func genLoad(name string, b llvm.Builder, m llvm.Module, fun llvm.Value, st *util.Stack) (llvm.Value, error) {
	// Check local scopes. Function parameters are on the bottom of the scope stack.
	for i1 := 1; i1 <= st.Size(); i1++ {
		if symtab := st.Get(i1).(*symTab); symtab != nil {
			if src, ok := symtab.m[name]; ok {
				return b.CreateLoad(src, ""), nil
			}
		}
	}

	// Check global scope.
	if val := m.NamedGlobal(name); val.IsNil() {
		return llvm.Value{}, fmt.Errorf("undeclared variable %q", name)
	} else {
		return b.CreateLoad(val, ""), nil
	}
}

// genType takes an ast.TYPED_VARIABLE_LIST or ast.DECLARATION and returns the type of the data variable(s).
func genType(n *ast.Node) (res llvm.Type, _ error) {
	if n == nil {
		return llvm.Type{}, errors.New("cannot generate LLVM type, node is <nil>")
	}
	if n.Data == nil {
		return res, errors.New("syntax tree node doesn't carry data")
	}
	switch n.Data.(string) {
	case "int":
		return i, nil
	case "float":
		return f, nil
	default:
		return res, fmt.Errorf("expected DECLARATION or TYPED_VARIABLE_LIST, got %s",
			n.Type())
	}
}

// genMain generates LLVM IR for the implicit main function. The main function takes the input arguments
// from the operating system and calls the first function defined in the syntax tree.
func genMain(b llvm.Builder, m llvm.Module, n *ast.Node) error {
	var callee *ast.Node
	var fun, atoi, atof llvm.Value

	// Find first declared function.
	for _, e1 := range n.Children {
		if e1.Typ == ast.FUNCTION {
			callee = e1
			break
		}
	}

	if callee == nil {
		return errors.New("no functions declared in syntax tree")
	}

	// Find the function's LLVM IR entry.
	if fun = m.NamedFunction(callee.Children[0].Data.(string)); fun.IsNil() {
		return errors.New("first function does not have LLVM IR global declaration")
	}

	// Define main function.
	var typ llvm.Type
	switch callee.Children[1].Data.(string) {
	case "int":
		typ = i
	case "float":
		typ = f
	default:
		return fmt.Errorf("undefined return data type of function %q, expected int or float, got %s",
			callee.Children[0].Data.(string), callee.Children[1].Data.(string))
	}
	params := []llvm.Type{i, llvm.PointerType(llvm.PointerType(llvm.Int8Type(), 0), 0)}
	ftyp := llvm.FunctionType(i, params, false)
	main := llvm.AddFunction(m, "main", ftyp)
	main.Param(0).SetName("argc")
	main.Param(1).SetName("argv")
	bb := llvm.AddBasicBlock(main, "")
	b.SetInsertPointAtEnd(bb)
	argcGood := llvm.AddBasicBlock(main, "argcGood")
	argcBad := llvm.AddBasicBlock(main, "argcBad")
	var argvBad llvm.BasicBlock

	// Verify arguments before calling VSL function.
	argc := b.CreateSub(main.Param(0), llvm.ConstInt(i, 1, true), "")
	cmp := b.CreateICmp(llvm.IntEQ, argc, llvm.ConstInt(i, uint64(len(fun.Params())), true), "")
	b.CreateCondBr(cmp, argcGood, argcBad)

	// Generate argc is ok.
	b.SetInsertPointAtEnd(argcGood)
	argv := main.Param(1)
	args := make([]llvm.Value, len(fun.Params()))

	// Verify argv by checking for successful int and/or float parses.
	// Based on: https://gist.github.com/Legacy25/d60f345c911748086443

	// argv[1] is the first argument to the called function.
	// i1 is the "iterator/incrementor" variable pointing to the right index of argv.
	i1 := llvm.ConstInt(i, 1, false)

	// Compile time indexer.
	idx := 0

	if len(callee.Children[2].Children) > 0 {
		argvBad = llvm.AddBasicBlock(main, "argvBad")
		for _, e1 := range callee.Children[2].Children {
			// Typed variable list.
			typ, err := genType(e1)
			if err != nil {
				return err
			}
			if typ == i && atoi.IsAFunction().IsNil() {
				atoi = genAtoi(m)
			} else if atof.IsAFunction().IsNil() {
				atof = genAtof(m)
			}

			for range e1.Children {
				// For all identifiers. Try parsing string to float or integer.

				// Create pointer to argv[i1].
				ptr := b.CreateGEP(
					argv,
					[]llvm.Value{
						i1,
					},
					"")

				var param llvm.Value
				newBB := llvm.AddBasicBlock(main, "")
				if typ == i {
					param = b.CreateCall(atoi, []llvm.Value{b.CreateLoad(ptr, "")}, "")
					cmp = b.CreateICmp(llvm.IntEQ, llvm.ConstInt(i, 0, false), param, "")
					b.CreateCondBr(cmp, argvBad, newBB)
				} else {
					param = b.CreateCall(atof, []llvm.Value{b.CreateLoad(ptr, "")}, "")
					cmp = b.CreateFCmp(llvm.FloatOEQ, llvm.ConstFloat(f, 0.0), param, "")
					b.CreateCondBr(cmp, argvBad, newBB)
				}
				b.SetInsertPointAtEnd(newBB)
				if idx < len(fun.Params())-1 {
					//ptr = b.CreateAdd(ptr, llvm.ConstInt(i, ib, false), "")
				}
				args[idx] = param
				idx++
				i1 = b.CreateAdd(i1, llvm.ConstInt(i, 1, false), "")
			}
		}
	}

	// Call function.
	ret := b.CreateCall(fun, args, "")

	// Check return value and exit.
	if typ == i {
		// Simply return the returned value.
		b.CreateRet(ret)
	} else {
		// Cast to integer and return.
		b.CreateRet(b.CreateFPToSI(ret, i, ""))
	}

	// Generate param parse mismatch.
	// Generate printf if it hasn't been generated already.
	pf := m.NamedFunction("printf")
	if pf.IsAFunction().IsNil() {
		genPrintf(m)
		pf = m.NamedFunction("printf")
	}

	if len(callee.Children[2].Children) > 0 {
		b.SetInsertPointAtEnd(argvBad)
		errMsg := b.CreateGlobalStringPtr(
			"failed to parse argument\n",
			stringPrefix)
		b.CreateCall(pf, []llvm.Value{errMsg}, "")
		b.CreateRet(llvm.ConstInt(i, 1, false))
	}

	// Generate argc mismatch.
	b.SetInsertPointAtEnd(argcBad)

	errMsg := b.CreateGlobalStringPtr(
		fmt.Sprintf("argument count mismatch, expected %d, got %%d\n", len(fun.Params())),
		stringPrefix)
	errArgs := []llvm.Value{errMsg, argc}
	b.CreateCall(pf, errArgs, "")
	b.CreateRet(llvm.ConstInt(i, 1, false))

	return nil
}

// genPrintf generates the LLVM IR printf definition.
func genPrintf(m llvm.Module) llvm.Value {
	// Declare printf.
	args := []llvm.Type{llvm.PointerType(llvm.Int8Type(), 0)}
	ftyp := llvm.FunctionType(llvm.Int32Type(), args, true)
	return llvm.AddFunction(m, "printf", ftyp)
}

// genAtof generates the Atoi function LLVM IR definition.
func genAtoi(m llvm.Module) llvm.Value {
	params := []llvm.Type{llvm.PointerType(llvm.Int8Type(), 0)}
	ftyp := llvm.FunctionType(llvm.Int32Type(), params, false)
	return llvm.AddFunction(m, "atoi", ftyp)
}

// genAtof generates the Atof function LLVM IR definition.
func genAtof(m llvm.Module) llvm.Value {
	params := []llvm.Type{llvm.PointerType(llvm.Int8Type(), 0)}
	ftyp := llvm.FunctionType(llvm.DoubleType(), params, false)
	return llvm.AddFunction(m, "atof", ftyp)
}

// genTargetTriple generates an LLVM target triple given the compiler options.
func genTargetTriple(opt *util.Options) (llvm.Target, string, error) {
	sb := strings.Builder{}
	var triple string

	// Target architecture. Revert to host system default if unknown.
	if opt.TargetArch == util.UnknownArch {
		// Used compiler host's default triple.
		triple = llvm.DefaultTargetTriple()
	} else {
		// Try generating target triple from CLI arguments.
		sb.Grow(20)

		switch opt.TargetArch {
		case util.Aarch64:
			sb.WriteString("aarch64")
		case util.Riscv64:
			sb.WriteString("riscv64")
		case util.Riscv32:
			sb.WriteString("riscv32")
		case util.X86_64:
			sb.WriteString("x86_64")
		case util.X86_32:
			sb.WriteString("x86")
		default:
			return llvm.Target{}, "", fmt.Errorf("unnsupported target architecture identifier %d",
				opt.TargetArch)
		}
		sb.WriteRune('-')

		// Target vendor. Defaults to PC.
		switch opt.TargetVendor {
		case util.PC:
			sb.WriteString("pc")
		case util.Apple:
			sb.WriteString("apple")
		case util.IBM:
			sb.WriteString("ibm")
		case util.UnknownVendor:
			sb.WriteString("pc")
		default:
			return llvm.Target{}, "", fmt.Errorf("unnsupported target vendor identifier %d",
				opt.TargetVendor)
		}
		sb.WriteRune('-')

		// Target operating system.
		if opt.TargetOS > 0 {
			switch opt.TargetOS {
			case util.Linux:
				sb.WriteString("linux")
			case util.Windows:
				sb.WriteString("win32")
			case util.MAC:
				sb.WriteString("darwin")
			default:
				return llvm.Target{}, "", fmt.Errorf("unnsupported target operating system identifier %d",
					opt.TargetArch)
			}
		} else {
			sb.WriteString("none")
		}

		// Target abi/environment.
		sb.WriteRune('-')
		sb.WriteString("gnu") // Default to GNU for now.

		triple = sb.String()
	}

	if opt.Verbose {
		fmt.Printf("compiling for target %s\n", triple)
	}
	llvm.InitializeAllTargets()
	if tt, err := llvm.GetTargetFromTriple(triple); err != nil {
		return llvm.Target{}, "", err
	} else {
		return tt, triple, nil
	}
}
