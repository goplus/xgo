package main

import "github.com/goplus/xgo/tpl"

func main() {
	tpl.NewEx(`
file = stmts => {
	return self
}

stmts = *(stmt ";") => {
	return [n.([]any)[0] for n in self]
}

stmt = varStmt | constStmt | outputStmt | inputStmt | ifStmt | whileStmt | untilStmt | assignStmt

varStmt = "DECLARE" namelist ":" typeExpr

constStmt = "CONSTANT" IDENT "<-" expr

assignStmt = IDENT "<-" expr

outputStmt = "OUTPUT" exprlist

inputStmt = "INPUT" namelist

ifStmt = "IF" expr "THEN" ";" stmts ?("ELSE" ";" stmts) "ENDIF"

whileStmt = "WHILE" expr "DO" ";" stmts "ENDWHILE"

untilStmt = "REPEAT" ";" stmts "UNTIL" expr

typeExpr = "INTEGER" | "REAL" | "STRING" | "BOOLEAN"

expr = binaryExpr2 % ("<" | "<=" | ">" | ">=" | "=" | "<>")

binaryExpr2 = binaryExpr1 % ("+" | "-")

binaryExpr1 = operand % ("*" | "/")

operand = basicLit | ident | parenExpr | unaryExpr

unaryExpr = "-" operand

basicLit = INT | FLOAT | STRING

ident = IDENT

parenExpr = "(" expr ")"

exprlist = expr % ","

namelist = IDENT % ","
`, "cl/_testgop/domaintpl/in.xgo", 1, 4, "file", func(self interface{}) interface{} {
		return self
	}, "stmts", func(self []interface{}) interface{} {
		return func() (_xgo_ret []interface{}) {
			for _, n := range self {
				_xgo_ret = append(_xgo_ret, n.([]interface{})[0])
			}
			return
		}()
	})
}
