# Simple paralleliseable VSL compiler made in Go

This Git repository contains all source code materials for my master thesis
with the ambition to explore the [Go programming language](https://go.dev/) 
as a replacement for the C programming language in educational and 
introductory compiler construction course, such as [TDT4205](https://www.ntnu.edu/studies/courses/TDT4205), 
for the [Norwegian University of Science and Technology](https://www.ntnu.no) (NTNU).

In addition to representing a replacement implementation of the [VSL](http://www.jeremybennett.com/publications/index.html) 
compiler the project examines *functional level parallelism* and other 
features that benefits from the ease of parallel code generation of the 
Go programming language.

**If you are looking for how to use the compiler please see 
[USAGE.md](usage.md).**

## Very Simple Language
Very Simple Language, (VSL), was defined by [Jeremy Bennett](http://www.jeremybennett.com/publications/index.html) 
in 1990. A [follow-up book](https://isbnsearch.org/isbn/9780077092214) 
was published in 1996. The work of Bennett uses [ANSI C](https://en.wikipedia.org/wiki/ANSI_C)
, [YACC](https://en.wikipedia.org/wiki/Yacc) 
and [Lex](https://en.wikipedia.org/wiki/Lex_(software)).

## Go features

### State function scanner
The scanner of this compiler was heavily inspired by Rob Pike's 
[talk](https://www.youtube.com/watch?v=HxaD_trXwRE) on Lexical Scanning in Go.
The slides shown in the video are available [here](https://talks.golang.org/2011/lex.slide#1).

In the video Rob Pike suggests that regular expressions are overkill for simple 
language scanners. He proposes the state function model where a scanner check for lexemes
by moving through the source stream, character by character, and acting based on
its internal state.


### Functional level parallelism
