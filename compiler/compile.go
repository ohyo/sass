package compiler

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/parser"
	"github.com/wellington/sass/token"
)

type Context struct {
	buf      *bytes.Buffer
	fileName *ast.Ident

	err error
	// Records the current level of selectors
	// Each time a selector is encountered, increase
	// by one. Each time a block is exited, remove
	// the last selector
	sels      [][]*ast.Ident
	activeSel *ast.BasicLit
	firstRule bool
	level     int
	printers  map[ast.Node]func(*Context, ast.Node)
	fset      *token.FileSet
	scope     Scope
}

func File(path string, out string) error {
	s, err := Run(path)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(out, []byte(s), 0666)
}

func Run(path string) (string, error) {
	ctx := &Context{}
	ctx.Init()
	out, err := ctx.Run(path)
	if err != nil {
		log.Fatal(err)
	}
	return out, err
}

func (ctx *Context) run(path string, src interface{}) (string, error) {
	// func ParseFile(fset *token.FileSet, filename string, src interface{}, mode Mode) (f *ast.File, err error) {
	ctx.fset = token.NewFileSet()
	// pf, err := parser.ParseFile(ctx.fset, path, src, parser.ParseComments)
	pf, err := parser.ParseFile(ctx.fset, path, src, parser.ParseComments|parser.Trace)
	if err != nil {
		return "", err
	}

	ast.Walk(ctx, pf)
	lr, _ := utf8.DecodeLastRune(ctx.buf.Bytes())
	_ = lr
	if ctx.buf.Len() > 0 && lr != '\n' {
		ctx.out("\n")
	}
	// ctx.printSels(pf.Decls)
	return ctx.buf.String(), nil
}

// Run takes a single Sass file and compiles it outputing a string
func (ctx *Context) Run(path string) (string, error) {
	return ctx.run(path, nil)
}

// out prints with the appropriate indention, selectors always have indent
// 0
func (ctx *Context) out(v string) {
	fr, _ := utf8.DecodeRuneInString(v)
	if fr == '\n' {
		fmt.Fprintf(ctx.buf, v)
		return
	}
	ws := []byte("                                              ")
	lvl := ctx.level

	format := append(ws[:lvl*2], "%s"...)
	fmt.Fprintf(ctx.buf, string(format), v)
}

// This needs a new name, it prints on every stmt
func (ctx *Context) blockIntro() {

	// this isn't a new block
	if !ctx.firstRule {
		fmt.Fprint(ctx.buf, "\n")
		return
	}

	ctx.firstRule = false

	// Only print newlines if there is text in the buffer
	if ctx.buf.Len() > 0 {
		if ctx.level == 0 {
			fmt.Fprint(ctx.buf, "\n")
		} else {

		}
	}
	sel := "MISSING"
	if ctx.activeSel != nil {
		sel = ctx.activeSel.Value
	}

	ctx.out(fmt.Sprintf("%s {\n", sel))
}

func (ctx *Context) blockOutro() {
	// Remove the innermost selector scope
	// if len(ctx.sels) > 0 {
	// 	ctx.sels = ctx.sels[:len(ctx.sels)-1]
	// }
	// Don't print } if there are no rules at this level
	if ctx.firstRule {
		return
	}

	ctx.firstRule = true
	// if !skipParen {
	fmt.Fprintf(ctx.buf, " }\n")
	// }
}

func (ctx *Context) Visit(node ast.Node) ast.Visitor {
	if ctx.err != nil {
		fmt.Println(ctx.err)
		return nil
	}
	var key ast.Node
	switch v := node.(type) {
	case *ast.BlockStmt:
		if ctx.scope.RuleLen() > 0 {
			ctx.level = ctx.level + 1
			if !ctx.firstRule {
				fmt.Fprintf(ctx.buf, " }\n")
			}
		}
		ctx.scope = NewScope(ctx.scope)
		ctx.firstRule = true
		for _, node := range v.List {
			ast.Walk(ctx, node)
		}
		if ctx.level > 0 {
			ctx.level = ctx.level - 1
		}
		ctx.scope = CloseScope(ctx.scope)
		ctx.blockOutro()
		ctx.firstRule = true
		// ast.Walk(ctx, v.List)
		// fmt.Fprintf(ctx.buf, "}")
		return nil
	case *ast.SelDecl:
	case *ast.File, *ast.GenDecl, *ast.Value:
		// Nothing to print for these
	case *ast.Ident:
		// The first IDENT is always the filename, just preserve
		// it somewhere
		key = ident
	case *ast.PropValueSpec:
		key = propSpec
	case *ast.DeclStmt:
		key = declStmt
	case *ast.IncludeSpec:
		// panic("not supported")
		// ast.Print(ctx.fset, node)
	case *ast.ValueSpec:
		key = valueSpec
	case *ast.RuleSpec:
		key = ruleSpec
	case *ast.SelStmt:
		// We will need to combine parent selectors
		// while printing these
		key = selStmt
		// Nothing to do
	case *ast.CommStmt:
	case *ast.CommentGroup:
	case *ast.Comment:
		key = comment
	case *ast.FuncDecl:
		ctx.printers[funcDecl](ctx, node)
		// Do not traverse mixins in the regular context
		return nil
	case *ast.BasicLit:
		return ctx
	case *ast.CallExpr:
	case nil:
		return ctx
	case *ast.EmptyStmt:
	case *ast.AssignStmt:
		key = assignStmt
	default:
		fmt.Printf("add printer for: %T\n", v)
		fmt.Printf("% #v\n", v)
	}
	ctx.printers[key](ctx, node)
	return ctx
}

var (
	ident       *ast.Ident
	expr        ast.Expr
	declStmt    *ast.DeclStmt
	assignStmt  *ast.AssignStmt
	valueSpec   *ast.ValueSpec
	ruleSpec    *ast.RuleSpec
	selDecl     *ast.SelDecl
	selStmt     *ast.SelStmt
	propSpec    *ast.PropValueSpec
	typeSpec    *ast.TypeSpec
	comment     *ast.Comment
	funcDecl    *ast.FuncDecl
	includeSpec *ast.IncludeSpec
)

func (ctx *Context) Init() {
	ctx.buf = bytes.NewBuffer(nil)
	ctx.printers = make(map[ast.Node]func(*Context, ast.Node))
	ctx.printers[valueSpec] = visitValueSpec
	ctx.printers[funcDecl] = visitFunc
	ctx.printers[assignStmt] = visitAssignStmt

	ctx.printers[ident] = printIdent
	ctx.printers[includeSpec] = printInclude
	ctx.printers[declStmt] = printDecl
	ctx.printers[ruleSpec] = printRuleSpec
	ctx.printers[selStmt] = printSelStmt
	ctx.printers[propSpec] = printPropValueSpec
	ctx.printers[expr] = printExpr
	ctx.printers[comment] = printComment
	ctx.scope = NewScope(empty)
	// ctx.printers[typeSpec] = visitTypeSpec
	// assign printers
}

func printComment(ctx *Context, n ast.Node) {
	ctx.blockIntro()
	cmt := n.(*ast.Comment)
	// These additional spaces should be handled by out()
	ctx.out("  " + cmt.Text)
}

func printExpr(ctx *Context, n ast.Node) {
	switch v := n.(type) {
	case *ast.File:
	case *ast.BasicLit:
		fmt.Fprintf(ctx.buf, "%s;", v.Value)
	case *ast.Value:
	case *ast.GenDecl:
		// Ignoring these for some reason
	default:
		// fmt.Printf("unmatched expr %T: % #v\n", v, v)
	}
}

func printSelStmt(ctx *Context, n ast.Node) {
	stmt := n.(*ast.SelStmt)
	ctx.activeSel = stmt.Resolved
}

func printRuleSpec(ctx *Context, n ast.Node) {
	// Inspect the sel buffer and dump it
	// Also need to track what level was last dumped
	// so selectors don't get printed twice
	ctx.blockIntro()

	spec := n.(*ast.RuleSpec)
	ctx.scope.RuleAdd(spec)
	ctx.out(fmt.Sprintf("  %s: ", spec.Name))
	var s string
	s, ctx.err = simplifyExprs(ctx, spec.Values)
	fmt.Fprintf(ctx.buf, "%s;", s)
}

func printPropValueSpec(ctx *Context, n ast.Node) {
	spec := n.(*ast.PropValueSpec)
	fmt.Fprintf(ctx.buf, spec.Name.String()+";")
}

// Variable assignments inside blocks ie. mixins
func visitAssignStmt(ctx *Context, n ast.Node) {
	fmt.Println("visit Assign")
	return
	stmt := n.(*ast.AssignStmt)
	var key, val *ast.Ident
	_, _ = key, val
	switch v := stmt.Lhs[0].(type) {
	case *ast.Ident:
		key = v
	default:
		log.Fatalf("unsupported key: % #v", v)
	}

	switch v := stmt.Rhs[0].(type) {
	case *ast.Ident:
		val = v
	default:
		log.Fatalf("unsupported key: % #v", v)
	}

}

// Variable declarations
func visitValueSpec(ctx *Context, n ast.Node) {
	return
}

func exprString(expr ast.Expr) string {
	switch v := (expr).(type) {
	case *ast.Ident:
		return v.String()
	case *ast.BasicLit:
		return v.Value
	default:
		panic(fmt.Sprintf("exprString %T: % #v\n", v, v))
	}
	return ""
}

func calculateExprs(ctx *Context, bin *ast.BinaryExpr) (string, error) {
	x := bin.X
	y := bin.Y

	var err error
	// Convert CallExpr to BasicLit
	if cx, ok := x.(*ast.CallExpr); ok {
		fmt.Printf("cx (%p) % #v\n", cx.Fun, cx.Fun.(*ast.Ident))
		x = cx.Fun.(*ast.Ident).Obj.Decl.(ast.Expr)
	}
	if cy, ok := y.(*ast.CallExpr); ok {
		fmt.Printf("cy (%p) % #v\n", cy.Fun, cy.Fun)
		y = cy.Fun.(*ast.Ident).Obj.Decl.(ast.Expr)
	}

	if err != nil {
		return "", err
	}

	bx := x.(*ast.BasicLit)
	by := y.(*ast.BasicLit)

	if bx == nil || by == nil {
		return "", fmt.Errorf("operand is nil % #v: % #v", bx, by)
	}

	// Attempt color math
	if bx.Kind == token.COLOR {
		z := bx.Op(bin.Op, by)
		if z == nil {
			// Op failed, just do string math
			z = &ast.BasicLit{
				Kind:  token.STRING,
				Value: bx.Value + by.Value,
			}
			// panic(fmt.Sprintf("invalid return op: %q x: % #v y: % #v",
			// 	bin.Op, bx, by,
			// ))
		}
		return z.Value, nil
	}

	// We're looking at INT and non-INT, treat as strings
	if bx.Kind == token.INT && by.Kind != token.INT {
		// Treat everything as strings
		return bx.Value + bin.Op.String() + by.Value, nil
	}

	return "", nil
}

func resolveIdent(ctx *Context, ident *ast.Ident) (out string) {
	v := ident
	if ident.Obj == nil {
		out = ident.Name
		return
	}
	switch vv := v.Obj.Decl.(type) {
	case *ast.Ident:
		out = resolveIdent(ctx, vv)
	case *ast.ValueSpec:
		var s []string
		for i := range vv.Values {
			if ident, ok := vv.Values[i].(*ast.Ident); ok {
				// If obj is set, resolve Obj and report
				if ident.Obj != nil {
					spec := ident.Obj.Decl.(*ast.ValueSpec)
					for _, val := range spec.Values {
						s = append(s, fmt.Sprintf("%s", val))
					}
				} else {
					// fmt.Printf("basic ident: % #v\n", ident)
					s = append(s, fmt.Sprintf("%s", ident))
				}
				continue
			}
			lit := vv.Values[i].(*ast.BasicLit)
			if len(lit.Value) > 0 {
				s = append(s, lit.Value)
			}
		}
		out = strings.Join(s, " ")
	case *ast.AssignStmt:
		lits := resolveAssign(ctx, vv)
		out = joinLits(lits, " ")
	case *ast.BasicLit:
		fmt.Printf("assigning %s: % #v\n", ident, vv)
		ident.Obj.Decl = vv
	default:
		fmt.Printf("unsupported VarDecl: % #v\n", vv)
		// Weird stuff here, let's just push the Ident in
		out = v.Name
	}
	return
}

// joinLits acts like strings.Join
func joinLits(a []*ast.BasicLit, sep string) string {
	s := make([]string, len(a))
	for i := range a {
		s[i] = a[i].Value
	}
	return strings.Join(s, sep)
}

func resolveAssign(ctx *Context, astmt *ast.AssignStmt) (lits []*ast.BasicLit) {

	for _, rhs := range astmt.Rhs {
		switch v := rhs.(type) {
		case *ast.Ident:
			assign := v.Obj.Decl.(*ast.AssignStmt)
			// Replace Ident with underlying BasicLit
			lits = append(lits, resolveAssign(ctx, assign)...)
		case *ast.CallExpr:
			lits = append(lits, v.Fun.(*ast.Ident).Obj.Decl.(*ast.BasicLit))
		case *ast.BasicLit:
			lits = append(lits, v)
		default:
			log.Fatalf("default rhs %s % #v\n", rhs, rhs)
		}
	}
	return
}

func resolveExpr(ctx *Context, expr ast.Expr) (out string, err error) {
	switch v := expr.(type) {
	case *ast.Value:
		panic("ast.Value")
	case *ast.BinaryExpr:
		out, err = calculateExprs(ctx, v)
	case *ast.CallExpr:
		expr := v.Fun.(*ast.Ident).Obj.Decl.(*ast.BasicLit)
		if expr == nil {
			return "", errors.New("call return was nil")
		}
		out = expr.Value
	case *ast.ParenExpr:
		out, ctx.err = simplifyExprs(ctx, []ast.Expr{v.X})
	case *ast.Ident:
		out = resolveIdent(ctx, v)
	case *ast.BasicLit:
		switch v.Kind {
		case token.VAR:
			// s, ok := ctx.scope.Lookup(v.Value).(string)
			// if ok {
			// 	sums = append(sums, s)
			// }
		default:
			out = v.Value
		}
	default:
		panic(fmt.Sprintf("unhandled expr: % #v\n", v))
	}
	return
}

func simplifyExprs(ctx *Context, exprs []ast.Expr) (string, error) {

	sums := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		s, err := resolveExpr(ctx, expr)
		if err != nil {
			return "", err
		}
		sums = append(sums, s)
	}
	start, end := 0, len(exprs)
	if lit, ok := exprs[0].(*ast.BasicLit); ok {
		if lit.Kind == token.QSTRING {
			start = 1
		}
	}
	if lit, ok := exprs[end-1].(*ast.BasicLit); ok {
		if lit.Kind == token.QSTRING {
			end = end - 2
		}
	}
	if start == 1 {
		sums = sums[start:end]
		return `"` + strings.Join(sums, " ") + `"`, nil
	}
	return strings.Join(sums, " "), nil
}

func printDecl(ctx *Context, node ast.Node) {
	// I think... nothing to print we'll see
}

func printIdent(ctx *Context, node ast.Node) {
	// ident := node.(*ast.Ident)
	// don't print these
	// fmt.Printf("ignoring % #v\n", ident)
}

func (c *Context) makeStrings(exprs []ast.Expr) (list []string) {
	list = make([]string, 0, len(exprs))
	for _, expr := range exprs {
		switch e := expr.(type) {
		case *ast.Ident:
			list = append(list, e.Name)
		}
	}
	return
}
