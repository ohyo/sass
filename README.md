# sass
Pure Go sass scanner, ast, and parser

Cross platform compiler for Sass

This project is currently in alpha, and contains no compiler. A scanner and parser are being developed to support a future compiler.

To help, check out [parser](https://github.com/wellington/sass/tree/master/parser). This project contains tests that iterate through sass-spec running the parser against example inputs. Errors detected by the parser are reported. However, you could also set the Parser mode to `Trace` and verify proper ast trees are being built from the input. As the parser matures, output can automatically be verified by the example outputs in these directories.

Glossary
- Partial Support :question:
- [x] Full Support
- [ ] No Support

### Parser Status
- [x] Nested Rules
- [ ] Referencing Parent Selectors: &
- [ ] Nested Properties
- [ ] Placeholder Selectors: %foo
- [x] Comments: /* */ and //
- SassScript :question:
- Variables: $ :question:
- Data Types :question:
- [ ] Strings
- [ ] Lists
- [ ] Maps
- [x] Colors
- Operations
  - [x] Number Operations
  - [x] Division and /
  - [x] Subtraction, Negative Numbers, and -
  - [ ] Color Operations
  - [ ] String Operations
  - [ ] Boolean Operations
  - [ ] List Operations
  - Parentheses :question:
- [x] Functions
- [x] Keyword Arguments
- Interpolation: #{} (Limited support for these) :question:
- [ ] & in SassScript
- [ ] Variable Defaults: !default
- @-Rules and Directives
  - [x] @import
  - [x] @media
  - [ ] @extend
    - [ ] Extending Complex Selectors
    - [ ] Multiple Extends
    - [ ] Chaining Extends
- [ ] Selector Sequences
  - [ ] Merging Selector Sequences
- [ ] @extend-Only Selectors
- [ ] The !optional Flag
- [ ] @extend in Directives
- [ ] @at-root
- [ ] @at-root (without: ...) and @at-root (with: ...)
- [ ] @debug
- [ ] @warn
- [ ] @error
- Control Directives & Expressions
  - [ ] if()
  - [ ] @if
  - [ ] @for
  - [ ] @each
    - [ ] Multiple Assignment
  - [ ] @while
  - [ ] url(/local/path)
  - [ ] url(http://remote/path)
- Mixin Directives
  - [x] Defining a Mixin: @mixin
  - [ ] Including a Mixin: @include
- Arguments
  - [x] Keyword Arguments
  - Variable Arguments :question:
- Passing Content Blocks to a Mixin
- Variable Scope and Content Blocks
- Function Directives :question:
- [ ] Extending Sass
- [ ] Defining Custom Sass Functions
