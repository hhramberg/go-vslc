# Simple paralleliseable VSL compiler made in Go

This Git repository contains all source code materials for my master thesis with the ambition to explore
the [Go programming language](https://go.dev/)
as a replacement for the C programming language in educational and introductory compiler construction course, such
as [TDT4205](https://www.ntnu.edu/studies/courses/TDT4205), for
the [Norwegian University of Science and Technology](https://www.ntnu.no) (NTNU).

In addition to representing a replacement implementation of
the [VSL](http://www.jeremybennett.com/publications/index.html)
compiler the project examines *functional level parallelism* and other features that benefits from the ease of parallel
code generation of the Go programming language.

**If you are looking for how to use/install the compiler, please see
[USAGE.md](doc/USAGE.md).**

**[scrips.md](doc/scripts.md) contain descriptions of the bundled test and benchmarking scripts.**

**View benchmarking results in [bench.md](doc/bench.md).**

## Very Simple Language

Very Simple Language, (VSL), was defined by [Jeremy Bennett](http://www.jeremybennett.com/publications/index.html)
in 1990. A [follow-up book](https://isbnsearch.org/isbn/9780077092214)
was published in 1996. The work of Bennett uses [ANSI C](https://en.wikipedia.org/wiki/ANSI_C)
, [YACC](https://en.wikipedia.org/wiki/Yacc)
and [Lex](https://en.wikipedia.org/wiki/Lex_(software)).

### Typed VSL

One of the additional features provided by this particular VSL compiler is the addition of data types. Two types
exists: `int`; the Integer type and `float`; the decimal floating point type. Typings can't be statically inferred at
compile time, so I have made some alterations to the existing VSL language. I introduced the keywords `int` and `float`,
added return data type on function declarations, added data type for variable declarations and added data type for
parameter declarations.

In the original VSL language a function definitions was something like this:

```VSL
def improve ( n, estimate )
begin
    // function body ...
end
```

In my type extended VSL I proposed the following syntax, showcasing both *parameter*
and function *return* type. In the below example parameters `n` and `estimate` are of type `int`, as is the return type.
This is very similar to the Go language syntax.

```VSL
def improve ( n, estimate int ) int
begin
    // function body ...
end
```

The initial declaration syntax is shown below.

```VSL
var a,b,c
var x,y,c
```

The proposed type declaration syntax is shown below, also similar to the Go syntax.

```VSL
var a,b,c int
var x,y,z float
```

A list of comma separated identifiers is assigned the type at the end of the list. A single declaration and parameter
declaration can be made as well.

```VSL
var score int
```

```VSL
def newton ( n int ) int
begin
    // function body ...
end
```

More on VSL type compatibility and assignment in [types.md](doc/types.md).

## Go features

### State function scanner

The scanner of this compiler was heavily inspired by Rob Pike's
[talk](https://www.youtube.com/watch?v=HxaD_trXwRE) on Lexical Scanning in Go. The slides shown in the video are
available [here](https://talks.golang.org/2011/lex.slide#1).

In the video Rob Pike suggests that regular expressions are overkill for simple language scanners. He proposes the state
function model where a scanner searches for lexemes by moving through the source stream, character by character, and
acting based on its internal state.

### Function level parallelism

Functions are inherently separate units of code. One function may reference other functions or variables not defined in
its local scope. However, at compile time this does not matter, as the instruction stream generated by the function is
defined in the function itself. I propose *function level parallelism* for faster optimisation, symbol table generation,
syntax tree validation and output generation. Function level parallelism lets threads work on functions in parallel
because of functions' inherent independence.

Dependencies lay in references, which can be safely verified by symbol tables using mutexes and synchronisation
barriers. Dependencies are resolved during symbol table creation and syntax tree validation. Symbol tables hold all
identifiers, while validation traverses the syntax tree and looks up any reference to the appropriate symbol table. We
can thus conclude that dependencies does not prevent functions from processed in parallel.

This is practically implemented such that all threads wait until all global references have been inserted into the
global symbol table before proceeding to process their local function bodies. This is the synchronisation barrier.

What makes this even more prevalent is the fact that validation requires only read access to shared resources. The only
shared resources between functions are global variables and the functions themselves. A synchronisation barrier is
adequate to validate inter-function dependencies.

## LLVM

[LLVM](https://llvm.org/) is a compiler toolchain, its statically typed intermediate representation (IR) and compiler
backend being its most prominent features. It is widely used in industry and academia and is interoperable with
different computer architectures, operating systems and vendors. It uses static single assignment (SSA) form to run
optimisation passes and register allocation that preferably improves compiled code quality.

In this project LLVM is one of two ways of generating near machine code output, the other being the built-in syntax tree
to assembler backend. Function level parallelism is implemented in LLVM IR generation the same way it is implemented in
the built-in syntax tree based assembler generators. When using LLVM for code generation, the output writer won't write
to the output file in parallel. This is because writing object file code in parallel is outside the scope of this
thesis.

From the viewpoint of this project, LLVM is the backend, when enabled, while the frontend serves as a VSL-to-LLVM
parser.
