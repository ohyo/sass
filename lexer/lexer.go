/*
The lexer processes text flagging any sass extended commands
sprite* as commands
*/
package lexer

import (
	"container/list"
	"errors"
	"fmt"
	"log"
	"strings"
	"unicode"
	"unicode/utf8"

	. "github.com/wellington/wellington/token"
)

const EOF rune = 0x04

// IsEOF returns true if n is zero.
func IsEOF(c rune, n int) bool {
	return n == 0
}

// IsInvalid returns true if c is utf8.RuneError and n is 1.
func IsInvalid(c rune, n int) bool {
	return c == utf8.RuneError && n == 1
}

func (l *Lexer) dequeue() *Item {
	head := l.items.Front()
	if head == nil {
		return nil
	}
	return l.items.Remove(head).(*Item)
}

// StateFn functions scan runes from the lexer's input and emit items.  A StateFn
// is responsible for emitting ItemEOF after input has been consumed.
type StateFn func(*Lexer) StateFn

// Lexer contains an input string and state associate with the lexing the
// input.
type Lexer struct {
	input string     // string being scanned
	start int        // start position for the current lexeme
	pos   int        // current position
	width int        // length of the last rune read
	last  rune       // the last rune read
	state StateFn    // the current state
	items *list.List // Buffer of lexed items
}

// Create a new lexer. Must be given a non-nil state.
func New(state StateFn, input string) *Lexer {

	if state == nil {
		return nil //panic("nil start state")
	}
	return &Lexer{
		state: state,
		input: input,
		items: list.New(),
	}
}

// Input returns the input string being lexed by the l.
func (l *Lexer) Input() string {
	return l.input
}

// Start marks the first byte of item currently being lexed.
func (l *Lexer) Start() int {
	return l.start
}

// Pos marks the next byte to be read in the input string.  The behavior of Pos
// is unspecified if an error previously occurred or if all input has been
// consumed.
func (l *Lexer) Pos() int {
	return l.pos
}

// Current returns the contents of the item currently being lexed.
func (l *Lexer) Current() string {
	return l.input[l.start:l.pos]
}

// Last return the last rune read from the input stream.
func (l *Lexer) Last() (r rune, width int) {
	return l.last, l.width
}

// Advance adds one rune of input to the current lexeme,
// increments the lexer's position, and returns the input
// rune with its size in bytes (encoded as UTF-8).  Invalid
// UTF-8 codepoints cause the current call and all subsequent
// calls to return (utf8.RuneError, 1).  If there is no input
// the returned size is zero.
func (l *Lexer) Advance() (rune, int) {
	if l.pos >= len(l.input) {
		l.width = 0
		return EOF, l.width
	}
	l.last, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	if l.last == utf8.RuneError && l.width == 1 {
		return l.last, l.width
	}
	l.pos += l.width
	return l.last, l.width
}

// Backup removes the last rune from the current lexeme and moves l's position
// back in the input string accordingly. Backup should only be called after a
// call to Advance.
func (l *Lexer) Backup() {
	l.pos -= l.width
}

// Peek returns the next rune in the input stream without adding it to the
// current lexeme.
func (l *Lexer) Peek() (rune, int) {
	c, n := l.Advance()
	l.Backup()
	return c, n
}

// Ignore throws away the current lexeme.
func (l *Lexer) Ignore() {
	l.start = l.pos
}

// Accept advances the lexer if the next rune is in valid.
func (l *Lexer) Accept(valid string) (ok bool) {
	r, _ := l.Advance()
	ok = strings.IndexRune(valid, r) >= 0
	if !ok {
		l.Backup()
	}
	return
}

// AcceptFunc advances the lexer if fn return true for the next rune.
func (l *Lexer) AcceptFunc(fn func(rune) bool) (ok bool) {
	switch r, n := l.Advance(); {
	case IsEOF(r, n):
		return false
	case IsInvalid(r, n):
		return false
	case fn(r):
		return true
	default:
		l.Backup()
		return false
	}
}

// AcceptRange advances l's position if the current rune is in tab.
func (l *Lexer) AcceptRange(tab *unicode.RangeTable) (ok bool) {
	r, _ := l.Advance()
	ok = unicode.Is(tab, r)
	if !ok {
		l.Backup()
	}
	return
}

// AcceptRun advances l's position as long as the current
// rune is in valid.
func (l *Lexer) AcceptRun(valid string) (n int) {
	for l.Accept(valid) {
		n++
	}
	return
}

// AcceptRunFunc advances l's position as long as fn returns
// true for the next input rune.
func (l *Lexer) AcceptRunFunc(fn func(rune) bool) int {
	var n int
	for l.AcceptFunc(fn) {
		n++
	}
	return n
}

// AcceptRunRange advances l's possition as long as the current
// rune is in tab.
func (l *Lexer) AcceptRunRange(tab *unicode.RangeTable) (n int) {
	for l.AcceptRange(tab) {
		n++
	}
	return
}

// AcceptString advances the lexer len(s) bytes if the next
// len(s) bytes equal s. AcceptString returns true if l advanced.
func (l *Lexer) AcceptString(s string) (ok bool) {
	if len(l.input)-l.pos < len(s) {
		return false
	}
	if strings.HasPrefix(l.input[l.pos:l.pos+len(s)], s) {
		l.pos += len(s)
		return true
	}
	return false
}

// Errorf causes an error item to be emitted from l.Next().  The item's value
// (and its error message) are the result of evaluating format and vs with
// fmt.Sprintf.
func (l *Lexer) Errorf(format string, vs ...interface{}) StateFn {
	l.enqueue(&Item{
		ItemError,
		l.start,
		fmt.Sprintf(format, vs...),
	})
	return nil
}

// Emit the current value as an Item with the specified type.
func (l *Lexer) Emit(t ItemType) {
	l.enqueue(&Item{
		t,
		l.start,
		l.input[l.start:l.pos],
	})
	l.start = l.pos
}

// The method by which items are extracted from the input.
// Returns nil if the lexer has entered a nil state.
func (l *Lexer) Next() (i *Item) {
	for {
		if head := l.dequeue(); head != nil {
			return head
		}
		if l.state == nil {
			return &Item{ItemEOF, l.start, ""}
		}
		l.state = l.state(l)
	}
}

func (l *Lexer) enqueue(i *Item) {
	l.items.PushBack(i)
}

const (
	Symbols = `/\.*-_`
)

func IsAllowedRune(r rune) bool {
	return unicode.IsNumber(r) ||
		unicode.IsLetter(r) ||
		strings.ContainsRune(Symbols, r)
}

// An individual scanned item (a lexeme).
type Item struct {
	Type  ItemType
	Pos   int
	Value string
}

func (i Item) Error() error {
	if i.Type == ItemError {
		return errors.New("Error reading input")
	}
	return nil
}

// String returns the raw lexeme of i.
func (i Item) String() string {
	switch i.Type {
	case ItemError:
		return i.Value
	case ItemEOF:
		return "EOF"
	}
	if len(i.Value) > 10 {
		return fmt.Sprintf("%s", i.Value)
	}
	return i.Value
}

func (l *Lexer) Action() StateFn {
	for {
		switch r, _ := l.Advance(); {
		case r == EOF: // || r == '\n':
			l.enqueue(&Item{
				ItemEOF,
				l.start,
				"",
			})
			return nil
		case IsSpace(r):
			l.Ignore()
		case IsSymbol(r):
			l.Backup()
			if len(l.Current()) > 0 {
				l.Emit(TEXT)
			}
			return l.Paren()
		case r == '/':
			if ok := l.Accept("/*"); ok {
				return l.Comment()
			}
			fallthrough
		case strings.IndexRune("*-+/", r) > -1: //l.Accept("*-+"):
			l.Backup()
			return l.Math()
		case r == '@':
			l.Backup()
			return l.Directive()
		case r == '"' || r == '\'':
			return l.File()
		case r == '$':
			return l.Var()
		case IsAllowedRune(r):
			l.Backup()
			return l.Text()
		default:
			//l.Advance()
			//l.Emit(EXTRA)
		}
	}
}

func IsSymbol(r rune) bool {
	return strings.ContainsRune("(),;{}#:", r)
}

func IsSpace(r rune) bool {
	return unicode.IsSpace(r)
}

func IsPrintable(r rune) bool {
	return true
}

func (l *Lexer) Math() StateFn {
	switch {
	case l.Accept("*"):
		l.Emit(MULT)
	case l.Accept("+"):
		l.Emit(PLUS)
	case l.Accept("-"):
		l.Emit(MINUS)
	case l.Accept("/"):
		l.Emit(MULT)
	}
	return l.Action()
}

func (l *Lexer) Directive() StateFn {
	switch {
	case l.AcceptString("@import"):
		l.Emit(IMPORT)
	case l.AcceptString("@include"):
		l.Emit(INCLUDE)
	case l.AcceptString("@each"):
		l.Emit(EACH)
	case l.AcceptString("@function"):
		l.Emit(FUNC)
	case l.AcceptString("@mixin"):
		l.Emit(MIXIN)
	case l.AcceptString("@if"):
		l.Emit(IF)
	case l.AcceptString("@else"):
		l.Emit(ELSE)
	default:
		// Unknown commands, write out as text
		// Be sure to write off unknown commands as text
		l.Accept("@")
		return l.Text()
	}

	return l.Action()
}

func (l *Lexer) Paren() StateFn {
	switch {
	case l.AcceptString("#{"):
		l.Emit(INTP)
	case l.Accept("("):
		last := l.items.Back()
		l.Emit(LPAREN)

		ok := l.Accept("$")
		if ok {
			l.AcceptRunFunc(IsAllowedRune)
			l.Emit(SUB)
			l.Accept(", ")
			return l.File()
		} else {
			// Special case for image-[width|height]
			switch fmt.Sprintf("%s", last.Value) {
			case "image-height", "image-width":
				return l.File()
			}
		}
	case l.Accept(")"):
		l.Emit(RPAREN)
	case l.Accept("{"):
		l.Emit(LBRACKET)
	case l.Accept("}"):
		l.Emit(RBRACKET)
	case l.Accept(":"):
		l.Emit(COLON)
	case l.Accept(";"):
		l.Emit(SEMIC)
	default:
		l.Advance()
	}
	return l.Action()
}

func (l *Lexer) Comment() StateFn {
	switch l.Current() {
	case "/*":
		// Look for the next '/' preceded with '*'
		var last rune
		for {
			r, _ := l.Advance()
			if last == '*' && r == '/' {
				break
			} else if r == EOF {
				// TODO: Surface language parsing errors
				log.Println("Invalid comment found", l.Current())
				break
			}
			last = r
		}
		l.Emit(CMT)
	case "//":
		// Single line comments
		for {
			r, _ := l.Advance()
			if !unicode.IsGraphic(r) {
				break
			}
		}
		l.Emit(CMT)
	}
	return l.Action()
}

// $images: sprite-map("*.png");
func (l *Lexer) Var() StateFn {
	l.Accept("$")
	l.AcceptRunFunc(IsAllowedRune)
	r, _ := l.Peek()

	if r == ':' {
		l.Emit(VAR)
	} else {
		l.Emit(SUB)
	}
	return l.Action()
}

func (l *Lexer) Text() StateFn {
	// Edge case when background:sprite();
	switch l.Current() {
	case ":":
		l.Ignore()
	}
	if ok := l.AcceptString("sprite-map"); ok {
		l.Emit(CMDVAR)
		return l.Action()
	}
	cmds := []string{
		// Supported commands
		"sprite-width", "sprite-height", "sprite-file",
		"sprite-height", "sprite-path", "sprite-position",
		"sprite-width", "sprite-url", "sprite-dimensions",
		// Future Support
		"sprite-map-name", "sprite-names",
		// Other commands
		"image-url", "inline-image",
		"image-width", "image-height",
	}

	for _, cmd := range cmds {
		if ok := l.AcceptString(cmd); ok {
			l.Emit(CMD)
			return l.Action()
		}
	}
	// Since this is a greedy algo, commands must be unique.
	// Many commands start with sprite, so do this after checking
	// all sprite... commands
	if ok := l.AcceptString("sprite"); ok {
		l.Emit(CMD)
		return l.Action()
	}

	if ok := l.Accept("-"); ok {
		l.Ignore()
		return l.Text()
	}
	// For unknown directives
	// Give up on searching for commands guess it is text
	l.AcceptRunFunc(IsAllowedRune)
	l.Emit(TEXT)

	return l.Action()
}

func (l *Lexer) File() StateFn {
	l.AcceptFunc(IsSpace)
	l.Ignore()
	l.AcceptRunFunc(IsAllowedRune)
	if len(l.Current()) > 0 {
		if c := Lookup(l.Current()); c > 0 {
			l.Emit(CMD)
		} else {
			l.Emit(FILE)
		}
	}
	return l.Action()
}
