%{

// Definitions and imports.
// Define imports for section three (Inserted go code) below.
package frontend

import "vslc/src/ir"

%}
%left '|'
%left '^'
%left '&'
%left LSHIFT RSHIFT
%left '+' '-'
%left '*' '/'
%nonassoc UMINUS
%nonassoc THEN
%nonassoc ELSE
%right '~'

%union {
    typ int
    val string
    line int
    pos int
    data interface{}
    node *ir.Node
}

%token DEF BEGIN END RETURN PRINT IF THEN ELSE WHILE DO CONTINUE VAR    // Reserved words.
%token INTEGER FLOAT IDENTIFIER STRING                                  // Data 'terminals'.
%token LSHIFT RSHIFT                                                    // Bitwise operators left and right shift.
%token ASSIGN                                                           // The assignment operator (:=).

%start program  // Tell goyacc that we want to end up with a 'root' non-terminal when all tokens have been parsed.

%%

program           :   global_list                                     { ir.Root = nodeInit(ir.PROGRAM, nil, $1.line, $1.pos, $1).node }

global_list       :   global                                          { $$ = nodeInit(ir.GLOBAL_LIST, nil, $1.line, $1.pos, $1) }
                  |   global_list global                              { $$ = nodeInit(ir.GLOBAL_LIST, nil, $1.line, $1.pos, $1, $2) }

global            :   function                                        { $$ = nodeInit(ir.GLOBAL, nil, $1.line, $1.pos, $1) }
                  |   declaration                                     { $$ = nodeInit(ir.GLOBAL, nil, $1.line, $1.pos, $1) }

statement_list    :   statement                                       { $$ = nodeInit(ir.STATEMENT_LIST, nil, $1.line, $1.pos, $1) }
                  |   statement_list statement                        { $$ = nodeInit(ir.STATEMENT_LIST, nil, $1.line, $1.pos, $1, $2) }

print_list        :   print_item                                      { $$ = nodeInit(ir.PRINT_LIST, nil, $1.line, $1.pos, $1) }
                  |   print_list ',' print_item                       { $$ = nodeInit(ir.PRINT_LIST, nil, $1.line, $1.pos, $1, $3) }

expression_list   :   expression                                      { $$ = nodeInit(ir.EXPRESSION_LIST, nil, $1.line, $1.pos, $1) }
                  |   expression_list ',' expression                  { $$ = nodeInit(ir.EXPRESSION_LIST, nil, $1.line, $1.pos, $1, $3) }

variable_list     :   identifier                                      { $$ = nodeInit(ir.VARIABLE_LIST, nil, $1.line, $1.pos, $1) }
                  |   variable_list ',' identifier                    { $$ = nodeInit(ir.VARIABLE_LIST, nil, $1.line, $1.pos, $1, $3) }

argument_list     :   expression_list                                 { $$ = nodeInit(ir.ARGUMENT_LIST, nil, $1.line, $1.pos, $1) }
                  |                                                   { $$ = nodeInit(ir.PARAMETER_LIST, nil, 0, 0) }

parameter_list    :   variable_list                                   { $$ = nodeInit(ir.PARAMETER_LIST, nil, $1.line, $1.pos, $1) }
                  |                                                   { $$ = nodeInit(ir.PARAMETER_LIST, nil, 0, 0) }

declaration_list  :   declaration                                     { $$ = nodeInit(ir.DECLARATION_LIST, nil, $1.line, $1.pos, $1) }
                  |   declaration_list declaration                    { $$ = nodeInit(ir.DECLARATION_LIST, nil, $1.line, $1.pos, $1, $2) }

function          :   DEF identifier '(' parameter_list ')' statement { $$ = nodeInit(ir.FUNCTION, nil, $1.line, $1.pos, $2, $4, $6) }

statement         :   assign_statement                                { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }
                  |   return_statement                                { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }
                  |   print_statement                                 { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }
                  |   if_statement                                    { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }
                  |   while_statement                                 { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }
                  |   null_statement                                  { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }
                  |   block                                           { $$ = nodeInit(ir.STATEMENT, nil, $1.line, $1.pos, $1) }

block             :   BEGIN declaration_list statement_list END       { $$ = nodeInit(ir.BLOCK, nil, $1.line, $1.pos, $2, $3) }
                  |   BEGIN statement_list END                        { $$ = nodeInit(ir.BLOCK, nil, $1.line, $1.pos, $2) }

assign_statement  :   identifier ASSIGN expression                    { $$ = nodeInit(ir.ASSIGNMENT_STATEMENT, nil, $1.line, $1.pos, $1, $3) }

return_statement  :   RETURN expression                               { $$ = nodeInit(ir.RETURN_STATEMENT, nil, $1.line, $1.pos, $2) }

print_statement   :   PRINT print_list                                { $$ = nodeInit(ir.PRINT_STATEMENT, nil, $1.line, $1.pos, $2) }

null_statement    :   CONTINUE                                        { $$ = nodeInit(ir.NULL_STATEMENT, nil, $1.line, $1.pos) }

if_statement      :   IF relation THEN statement                      { $$ = nodeInit(ir.IF_STATEMENT, nil, $1.line, $1.pos, $2, $4) }
                  |   IF relation THEN statement ELSE statement       { $$ = nodeInit(ir.IF_STATEMENT, nil, $1.line, $1.pos, $2, $4, $6) }

while_statement   :   WHILE relation DO statement                     { $$ = nodeInit(ir.WHILE_STATEMENT, nil, $1.line, $1.pos, $2, $4) }

relation          :   expression '=' expression                       { $$ = nodeInit(ir.RELATION, "=", $1.line, $1.pos, $1, $3) }
                  |   expression '<' expression                       { $$ = nodeInit(ir.RELATION, "<", $1.line, $1.pos, $1, $3) }
                  |   expression '>' expression                       { $$ = nodeInit(ir.RELATION, ">", $1.line, $1.pos, $1, $3) }

expression        :   expression '+' expression                       { $$ = nodeInit(ir.EXPRESSION, "+", $1.line, $1.pos, $1, $3) }
                  |   expression '-' expression                       { $$ = nodeInit(ir.EXPRESSION, "-", $1.line, $1.pos, $1, $3) }
                  |   expression '*' expression                       { $$ = nodeInit(ir.EXPRESSION, "*", $1.line, $1.pos, $1, $3) }
                  |   expression '/' expression                       { $$ = nodeInit(ir.EXPRESSION, "/", $1.line, $1.pos, $1, $3) }
                  |   expression '|' expression                       { $$ = nodeInit(ir.EXPRESSION, "|", $1.line, $1.pos, $1, $3) }
                  |   expression '^' expression                       { $$ = nodeInit(ir.EXPRESSION, "^", $1.line, $1.pos, $1, $3) }
                  |   expression '&' expression                       { $$ = nodeInit(ir.EXPRESSION, "&", $1.line, $1.pos, $1, $3) }
                  |   expression LSHIFT expression                    { $$ = nodeInit(ir.EXPRESSION, "<<", $1.line, $1.pos, $1, $3) }
                  |   expression RSHIFT expression                    { $$ = nodeInit(ir.EXPRESSION, ">>", $1.line, $1.pos, $1, $3) }
                  |   '-' expression %prec UMINUS                     { $$ = nodeInit(ir.EXPRESSION, "-", $1.line, $1.pos, $2) }
                  |   '~' expression                                  { $$ = nodeInit(ir.EXPRESSION, "~", $1.line, $1.pos, $2) }
                  |   '(' expression ')'                              { $$ = nodeInit(ir.EXPRESSION, nil, $2.line, $2.pos, $2) }
                  |   number                                          { $$ = nodeInit(ir.EXPRESSION, nil, $1.line, $1.pos, $1) }
                  |   identifier                                      { $$ = nodeInit(ir.EXPRESSION, nil, $1.line, $1.pos, $1) }
                  |   identifier '(' argument_list ')'                { $$ = nodeInit(ir.EXPRESSION, nil, $1.line, $1.pos, $1, $3) }

declaration       :   VAR variable_list                               { $$ = nodeInit(ir.DECLARATION, nil, $2.line, $2.pos, $2) }

print_item        :   expression                                      { $$ = nodeInit(ir.PRINT_ITEM, nil, $1.line, $1.pos, $1) }
                  |   string                                          { $$ = nodeInit(ir.PRINT_ITEM, nil, $1.line, $1.pos, $1) }

identifier        :   IDENTIFIER                                      { $$ = nodeInit(ir.IDENTIFIER_DATA, $1.val, $1.line, $1.pos) }

number            :   INTEGER                                         { $$ = nodeInit(ir.INTEGER_DATA, $1.val, $1.line, $1.pos) }
                  |   FLOAT                                           { $$ = nodeInit(ir.FLOAT_DATA, $1.val, $1.line, $1.pos) }

string            :   STRING                                          { $$ = nodeInit(ir.STRING_DATA, $1.val, $1.line, $1.pos) }

%%
