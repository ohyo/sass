package strops

import (
	"strconv"

	"github.com/wellington/sass/ast"
	"github.com/wellington/sass/builtin"
	"github.com/wellington/sass/strops"
	"github.com/wellington/sass/token"
)

func init() {
	builtin.Register("unquote($string)", unquote)
	builtin.Register("length($value)", length)
}

func unquote(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	in := *args[0]
	lit := &ast.BasicLit{
		Kind:     token.STRING,
		ValuePos: in.ValuePos,
		Value:    strops.Unquote(in.Value),
	}
	// Because in Ruby Sass, there is no failure though libSass fails
	// very easily
	return lit, nil
}

func length(call *ast.CallExpr, args ...*ast.BasicLit) (*ast.BasicLit, error) {
	in := *args[0]
	lit := &ast.BasicLit{
		Kind:     token.INT,
		Value:    strconv.Itoa(len(in.Value)),
		ValuePos: in.ValuePos,
	}
	return lit, nil
}