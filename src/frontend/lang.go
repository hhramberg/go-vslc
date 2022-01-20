package frontend

type reservedItem struct {
	val string
	typ itemType
}

// rw contains the set of all reserved VSL keywords.
// The first dimension equals the length of the word.
// The second dimension is the slice of all words of that length.
// Indexing by length and searching should be faster than using a hash table.
var rw = [...][]reservedItem{
	// One-grams
	{},
	// Two-grams
	{
		{val: "do", typ: DO},
		{val: "if", typ: IF},
	},
	// Three-grams
	{
		{val: "def", typ: DEF},
		{val: "end", typ: END},
		{val: "var", typ: VAR},
		{val: "int", typ: TYPE},
	},
	// Four-grams
	{
		{val: "then", typ: THEN},
		{val: "else", typ: ELSE},
	},
	// Five-grams
	{
		{val: "begin", typ: BEGIN},
		{val: "while", typ: WHILE},
		{val: "print", typ: PRINT},
		{val: "float", typ: TYPE},
	},
	// Six-grams
	{
		{val: "return", typ: RETURN},
	},
	// Seven-grams
	{},
	// Eight-grams
	{
		{val: "continue", typ: CONTINUE},
	},
}

// isKeyword returns true if the string s is a reserved VSL keyword.
// On the return of true the itemType of the keyword is returned.
// On the return of false the itemType is either IDENTIFIER or itemError.
func isKeyword(s string) (bool, itemType) {
	if len(s) == 0 {
		return false, itemError
	}
	if len(s) > len(rw) {
		return false, IDENTIFIER
	}

	// Check if string s is a reserved word by iterating over all words in rw of length len(s).
	for _, e1 := range rw[len(s)-1] {
		if e1.val == s {
			return true, e1.typ
		}
	}
	return false, IDENTIFIER
}
