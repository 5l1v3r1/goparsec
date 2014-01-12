// Package parsec implements a library of parser-combinators using basic
// recognizers like - And, OrdChoice, Kleene, Many and Maybe.
//
// parsec tool's combinators can be used to construct higher-order parsers
// using basic parser function. All parser functions are expected to follow
// the `Parser` type signature, accepting a `Scanner` interface and returning
// a `ParsecNode` and a new scanner. If a parser fails match the input string
// according to its rules, then it must return nil, for ParsecNode, and a new
// Scanner.
//
// ParsecNode can either be a Terminal structure or NonTerminal structure or a
// list of Terminal/NonTerminal structure. The AST output is expected to be
// made up of ParseNode.
//
// Nodify is a callback function that every combinators use as a callback
// to construct a ParsecNode.

package parsec

type ParsecNode interface{}                     // Used to construct AST.
type Parser func(Scanner) (ParsecNode, Scanner) // combinable parsers
type Nodify func([]ParsecNode) ParsecNode

// Terminal structure can be used to construct a terminal ParsecNode.
type Terminal struct {
	Name     string // contains terminal's token type
	Value    string // value of the terminal
	Position int    // Offset into the text stream where token was identified
}

// NonTerminal structure can be used to construct a non-terminal ParsecNode.
type NonTerminal struct {
	Name     string       // contains terminal's token type
	Value    string       // value of the terminal
	Children []ParsecNode // list of children to this node.
}

// Scanner interface supplies necessary methods to match the input stream.
type Scanner interface {
	// Clone will return new clone of the underlying scanner structure. This
	// will be used by combinators to _backtrack_.
	Clone() Scanner

	// GetCursor gets the current cursor position inside input text.
	GetCursor() int

	// Match the input stream with `pattern` and return matching string after
	// advancing the cursor.
	Match(pattern string) ([]byte, Scanner)

	// SkipWs skips white space characters in the input stream. Return skipped
	// whitespaces as byte-slice and advance the cursor.
	SkipWS() ([]byte, Scanner)

	// Endof detects whether end-of-file is reached in the input stream and
	// return a boolean indicating the same.
	Endof() bool
}

// And combinator accepts a list of `Parser` that must match the input string,
// atleast until the last Parser argument, and returns a parser function that
// can be used to construct higher-level parsers.
//
// If all parser matches, then returned ParsecNode will be a list of
// ParseNode - as returned by the individual parser function.
//
// Even if one of the input parser function fails, then ParseNode will be an
// empty slice.
func And(callb Nodify, parsers ...Parser) Parser {
	return func(s Scanner) (ParsecNode, Scanner) {
		var ns = make([]ParsecNode, 0, len(parsers))
		var n ParsecNode
		news := s.Clone()
		for _, parser := range parsers {
			n, news = parser(news)
			if n == nil {
				return nil, s
			}
			ns = append(ns, n)
		}
		return docallback(callb, ns), news
	}
}

// OrdChoice combinatore accepts a list of `Parser`, where atleast one of the
// parser must match the input string, and returns a parser that can be used
// to construct higher level parsers.
//
// The first matching parser function's output is returned as ParsecNode.
//
// If non of the parsers match the input, then empty slice is retunred as
// ParsecNode.
func OrdChoice(callb Nodify, parsers ...Parser) Parser {
	return func(s Scanner) (ParsecNode, Scanner) {
		for _, parser := range parsers {
			n, news := parser(s.Clone())
			if n != nil {
				return docallback(callb, []ParsecNode{n}), news
			}
		}
		return nil, s
	}
}

// Kleen combinator accepts two parsers, namely opScan and sepScan, where
// opScan parser will be used to match input string and contruct ParsecNode
// and sepScan parser will be used to match input string and ignore the
// matched string. If sepScan parser is not supplied, then opScan parser will
// be applied on the input until it fails.
//
// The process of matching opScan parser and sepScan parser will continue in a
// loop until either one of them fails on the input stream.
//
// For every successful match of opScan, the returned ParseNode from the
// parser will be accumulated and returned as the final result.
//
// If there is not a single match for opScan, then an empty slice (not nil)
// will be returned as ParsecNode
func Kleene(callb Nodify, parsers ...Parser) Parser {
	var opScan, sepScan Parser
	opScan = parsers[0]
	if len(parsers) == 2 {
		sepScan = parsers[1]
	} else {
		panic("Kleene parser does not accept more than 2 parsers")
	}
	return func(s Scanner) (ParsecNode, Scanner) {
		var n ParsecNode
		ns := make([]ParsecNode, 0)
		news := s.Clone()
		for {
			n, news = opScan(news)
			if n == nil {
				break
			}
			ns = append(ns, n)
			if sepScan != nil {
				if n, news = sepScan(news); n == nil {
					break
				}
			}
		}
		return docallback(callb, ns), news
	}
}

// Many combinator accepts two parsers, namely opScan and sepScan, where
// opScan parser will be used to match input string and contruct ParsecNode
// and sepScan parser will be used to match input string and ignore the
// matched string. If sepScan parser is not supplied, then opScan parser will
// be applied on the input until it fails.
//
// The process of matching opScan parser and sepScan parser will continue in a
// loop until either one of them fails on the input stream.
//
// For every successful match of opScan, the returned ParseNode from the
// parser will be accumulated and returned as the final result.
//
// If there is not a single match for opScan, then the parser returns nil for
// ParsecNode.
func Many(callb Nodify, parsers ...Parser) Parser {
	var opScan, sepScan Parser
	opScan = parsers[0]
	if len(parsers) >= 2 {
		sepScan = parsers[1]
	}
	return func(s Scanner) (ParsecNode, Scanner) {
		var n ParsecNode
		ns := make([]ParsecNode, 0)
		news := s.Clone()
		for {
			n, news = opScan(news)
			if n != nil {
				ns = append(ns, n)
				if sepScan != nil {
					if n, news = sepScan(news); n == nil {
						break
					}
				}
			} else {
				break
			}
		}
		if len(ns) == 0 {
			return nil, s
		}
		return docallback(callb, ns), news
	}
}

// Maybe combinator accepts a single parser, and tries to match the input
// stream with it.
func Maybe(callb Nodify, parser Parser) Parser {
	return func(s Scanner) (ParsecNode, Scanner) {
		n, news := parser(s.Clone())
		if n == nil {
			return nil, s
		}
		return docallback(callb, []ParsecNode{n}), news
	}
}

//---- Local function

func docallback(callb Nodify, ns []ParsecNode) ParsecNode {
	if callb != nil {
		return callb(ns)
	} else {
		return ns
	}
}
