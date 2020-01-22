# Golang VTL engine
## Differences from Apache VTL
These are deliberate decisions to make syntax less confusing
1. [Whitespace gobbling](http://velocity.apache.org/engine/devel/developer-guide.html#space-gobbling) is always `lines`, and is the same for single-line comments and directives

   If directive or comment is alone on the line, possibly surrounded by whitespace, then parser ignores leading spaces and tabs (`' '` and `'\t'`) and trailing newline (`'\n'`) and any spaces and tabs preceeding this newline.
2. [Strict rendering](https://velocity.apache.org/engine/devel/user-guide.html#strict-rendering-mode) mode cannot be disabled and effective for both variables and directives

   That means any undefined directive will trigger parsing error and any undefiened variable, method or property will trigger runtime error.
3. String interpolation

   Only [references](https://velocity.apache.org/engine/devel/user-guide.html#references) will be interpolated in double-quoted strings, not directives or comments
4. [Varaibles](https://velocity.apache.org/engine/devel/user-guide.html#variables)

   There is no option to enable hyphen in variables
5. [Escaping](https://velocity.apache.org/engine/devel/user-guide.html#getting-literal) `$` and `#`

   There is no difference in behaviour when escaping defined or undefined variable - it always works as if variable was defined. Escaping works consistently for variables and directives. If `$` or `#` is escaped go-vtl will never try to parse what is left as variable or directive.
7. [Alternate values](https://velocity.apache.org/engine/devel/user-guide.html#alternate-values) not supported (yet)

8. Consistent usage of comma and space

   Any arguments should be separated by comma, be it method call, include, macro signature or macro call
9. Maps are iterated in sorted key order

## Deficiencies
1. Huge ranges will trigger OOM, e.g. [1 .. 67000000]
