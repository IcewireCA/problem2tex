package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
)

// runs a line of code that may have several statements in it (separated by ";")
// returns assignVar where assignVar is the assigned variable.  it is last assigned var if multiple statements
// also returns answer (float64) where it is the last answer calculated if there are multiple statements in one line
func runCode(inString string, varAll map[string]varSingle, configParam map[string]string) (assignVar, outString string, answer float64, errCode string) {
	var result, lineCode []string
	var allOptions []option
	var infixCode, rpnCode, optionStr, prefix, units string
	var errorInfix, errorRpn string
	var tmp2 varSingle
	var reAssignment = regexp.MustCompile(`(?m)^\s*(?P<res1>\w*)\s*=\s*(?P<res2>.*)$`)
	var reGetFirst = regexp.MustCompile(`(?mU)^(?P<res1>.*);(?P<res2>.*)$`)
	var reBeforeComment = regexp.MustCompile(`(?mU)^(?P<res1>.*)#.*$`)
	var reOptions = regexp.MustCompile(`(?mU)^.*#(?P<res1>.*)$`)

	if reOptions.MatchString(inString) {
		optionStr = reOptions.FindStringSubmatch(inString)[1]
	}
	if reBeforeComment.MatchString(inString) {
		inString = reBeforeComment.FindStringSubmatch(inString)[1] // strip off comments after #
	}
	outString = inString
	for reGetFirst.MatchString(inString) { // break inString into lines of code between ";"
		result = reGetFirst.FindStringSubmatch(inString)
		lineCode = append(lineCode, result[1])
		inString = result[2]
	}
	lineCode = append(lineCode, inString) // anything left over is final line code
	// run each line code
	for i := range lineCode {
		errCode = syntaxError(lineCode[i], "runCode")
		if errCode != "" {
			assignVar = "dummy"
			outString = errCode
			return
		}
		// using switch statement so other types of code can be run
		// first type is assignment
		// might add if statement or ...
		switch {
		case reAssignment.MatchString(lineCode[i]):
			result = reAssignment.FindStringSubmatch(lineCode[i])
			assignVar = result[1]
			infixCode = result[2]
			assignVar, errCode = checkReserved(assignVar, errCode)
			if errCode != "" {
				outString = assignVar
				return
			}
			rpnCode, errorInfix = infix2rpn(infixCode)
			if errorInfix != "" {
				errCode = errorInfix
				return
			}
			answer, errorRpn = rpnEval(rpnCode, varAll)
			if errorRpn != "" {
				errCode = errorRpn
				outString = errCode
				return
			}
			_, ok := varAll[assignVar]
			if !ok { // assignVar is a new variable and needs to be added to varAll map
				varAll[assignVar] = varSingle{} // adding blank new variable to varAll map
				tmp2 = varAll[assignVar]
				tmp2.units = defaultUnitsVar(assignVar, configParam) // add default units which may be overwritten later
				tmp2.latex = latexifyVar(assignVar)                  // add latex version of assignVar
			} else {
				tmp2 = varAll[assignVar] // use existing map location
			}

			// Now deal with options in the RUN command
			allOptions = getAllOptions(optionStr)
			for i := 0; i < len(allOptions); i++ {
				switch allOptions[i].name {
				case "units":
					prefix, units = getPrefixUnits(allOptions[i].value) // separate preUnits into prefix and units
					if prefix != "" {                                   // there should be NO prefix in RUN commands (it would not make sense since many variables are allowed on right side)
						outString = "option units CAN NOT have prefix in RUN commands"
						errCode = outString
					}
					tmp2.units = units
				case "symbol":
					tmp2.latex = allOptions[i].value
				default:
					errCode = allOptions[i].name + " is not a valid option"
					return
				}
			} // done with options

			tmp2.value = answer
			varAll[assignVar] = tmp2

		default: // if no assignment or other command, just evaluate the expression and update varAll["_ans_"]
			assignVar = ""
			rpnCode, errorInfix = infix2rpn(lineCode[i])
			if errorInfix != "" {
				errCode = errorInfix
				return
			}
			answer, errorRpn = rpnEval(rpnCode, varAll)
			if errorRpn != "" {
				errCode = errorRpn
				return
			}
			return
		}
	}
	return
}

// used to get next token from an infix equation
func getNextToken(inString, lastTokenType string, infix bool) (token, tokenType, remainder, errorToken string) {
	var firstChar, nextChar string
	var result []string

	isAlpha := regexp.MustCompile(`^[A-Za-z]$`).MatchString // to check if a single char is alpha character
	isDigit := regexp.MustCompile(`^[0-9]$`).MatchString    // to check if a single char is a digit
	isSomething := regexp.MustCompile(`\S`).MatchString     // to check that there is something in the string that is non-whitespace
	var reFirstChar = regexp.MustCompile(`(?m)^\s*(?P<res1>.)`)
	var reWord = regexp.MustCompile(`(?m)^\s*(?P<res1>\w+)(?P<res2>.?)(?P<res3>.*)$`)
	var reNumber = regexp.MustCompile(`(?m)^\s*(?P<res1>\d*\.?\d*([eE][-+]?[0-9]+)?)(?P<res2>.*)$`)
	var reRemoveFirstChar = regexp.MustCompile(`(?m)^\s*.(?P<res1>.*)$`)

	if !isSomething(inString) { // if there is only whitespace in inString, then return and end
		token = ""
		remainder = ""
		tokenType = "end"
		return
	}
	firstChar = reFirstChar.FindStringSubmatch(inString)[1]
	switch firstChar {
	case "+": // either add operator or unary pos function
		tokenType = "operator"
		token = firstChar
		if infix { // if input is an infix equation, then "+" may be a unary function
			switch lastTokenType {
			case "rightBrac", "number", "variable": // do nothing as already correct
			default: // if anything else, then token is a unary function (not an operator)
				tokenType = "function"
				token = "pos"
			}
		}
		remainder = reRemoveFirstChar.FindStringSubmatch(inString)[1]
		return
	case "-": // either subtract operator or unary neg function
		tokenType = "operator"
		token = firstChar
		if infix {
			switch lastTokenType {
			case "rightBrac", "number", "variable": // do nothing as already correct
			default: // if anything else, then token is a unary function (not an operator)
				tokenType = "function"
				token = "neg"
			}
		}
		remainder = reRemoveFirstChar.FindStringSubmatch(inString)[1]
		return
	case "*", "/", "^": // operator
		token = firstChar
		remainder = reRemoveFirstChar.FindStringSubmatch(inString)[1]
		tokenType = "operator"
		return
	case "#": // comment symbol so this and rest of line can be ignored
		token = ""
		remainder = ""
		tokenType = "end"
		return
	case "(": // left bracket
		token = firstChar
		remainder = reRemoveFirstChar.FindStringSubmatch(inString)[1]
		tokenType = "leftBrac"
		return
	case ")": // right bracket
		token = firstChar
		remainder = reRemoveFirstChar.FindStringSubmatch(inString)[1]
		tokenType = "rightBrac"
		return
	case ",": // used for function arguments
		token = ""
		remainder = reRemoveFirstChar.FindStringSubmatch(inString)[1]
		tokenType = ""
		return
	default:
		switch {
		case isAlpha(firstChar): // is an alphbetic char so token is either function or variable
			result = reWord.FindStringSubmatch(inString)
			token = result[1]
			nextChar = result[2] // look right after token to see if there is "("
			remainder = nextChar + result[3]
			_, ok1 := func1[token]
			_, ok2 := func2[token]
			if ok1 || ok2 { // if word is in function maps, then it is a function
				tokenType = "function"
			} else {
				tokenType = "variable"
			}
			if tokenType == "function" && nextChar != "(" && infix { // error if function not followed by "(" when infix = true
				errorToken = token + ": is a function and should be followed by ("
			}
			return
		case isDigit(firstChar) || firstChar == ".": // is a digit or . (both mean token is a number)
			result = reNumber.FindStringSubmatch(inString)
			token = result[1]
			remainder = result[3]
			tokenType = "number"
			return
		default:
			// if here then don't know what the token is
			token = ""
			tokenType = "end"
			remainder = ""
			errorToken = firstChar + " ** not a valid character in equation"
			return
		}
	}
}

func rpnEval(inString string, varAll map[string]varSingle) (answer float64, errorRpn string) {
	var stack []float64
	var token, tokenType, remainder, errorToken string
	for tokenType != "end" {
		token, tokenType, remainder, errorToken = getNextToken(inString, "", false)
		inString = remainder
		if errorToken != "" {
			answer = math.NaN() // returns NaN as math result
			errorRpn = errorToken
			return
		}
		switch tokenType {
		case "end":
		case "operator":
			if len(stack) > 1 {
				stack[len(stack)-2] = funcTwo(stack[len(stack)-2], stack[len(stack)-1], func2[token].name) // replace 2nd from top element
				stack = stack[:len(stack)-1]                                                               // remove top of stack
			} else {
				errorRpn = "  ** equation error: too many operators or functions"
				answer = math.NaN() // returns NaN as math result
				return
			}
		case "function":
			_, ok := func2[token]
			if ok {
				stack[len(stack)-2] = funcTwo(stack[len(stack)-2], stack[len(stack)-1], func2[token].name)
				stack = stack[:len(stack)-1]
			}
			_, ok = func1[token]
			if ok {
				stack[len(stack)-1] = funcOne(stack[len(stack)-1], func1[token])
			}

		case "number":
			x, _ := strconv.ParseFloat(token, 64)
			stack = append(stack, x)
		case "variable":
			_, ok := varAll[token]
			if !ok {
				errorRpn = token + ": ** not defined"
				answer = math.NaN() // returns NaN as math result
				return
			}
			stack = append(stack, varAll[token].value)
		default:
		}
	}
	switch {
	case len(stack) == 1:
		answer = stack[0]
		errorRpn = ""
	case len(stack) < 1:
		answer = math.NaN() // returns NaN as math result
		errorRpn = " ** equation not parsed **"
	case len(stack) > 1:
		answer = math.NaN() // returns NaN as math result
		errorRpn = " ** can not parse equation **"
	default:
	}
	return
}

func infix2rpn(inString string) (rpn, errorInfix string) {
	var stack []tokenAndType // holds operators and left parenthesis (holds both token and tokenType for each element)
	var token, tokenType, remainder, errorToken, input string
	var rpnSlice []string
	input = inString
	_ = input
	for tokenType != "end" {
		token, tokenType, remainder, errorToken = getNextToken(inString, tokenType, true)
		inString = remainder
		if errorToken != "" {
			errorInfix = errorToken
			rpn = ""
			return
		}
		switch tokenType {
		case "end":
		case "number", "variable":
			rpnSlice = append(rpnSlice, token) // push to output
			if len(stack) > 0 {                // pop off stack if stack hold either "pos" or "neg" as top element
				opTop := stack[len(stack)-1]
				switch opTop.token {
				case "neg", "pos":
					stack = stack[:len(stack)-1] // pop top off stack
					rpnSlice = append(rpnSlice, opTop.token)
				default:
				}
			}

		case "function":
			stack = append(stack, tokenAndType{token, tokenType})
		case "operator":
			opNew := tokenAndType{token, tokenType}
			for len(stack) > 0 {
				// consider top item on stack and compare to opNew
				opTop := stack[len(stack)-1]
				if (func2[opTop.token].prec < func2[opNew.token].prec) || (opTop.tokenType != "operator") {
					break
				}
				stack = stack[:len(stack)-1] // pop top off stack
				rpnSlice = append(rpnSlice, opTop.token)
			}
			stack = append(stack, tokenAndType{token, tokenType})
		case "leftBrac":
			stack = append(stack, tokenAndType{token, tokenType})
		case "rightBrac":
			if len(stack) == 0 {
				errorInfix = " ** unmatched parenthesis"
				rpn = ""
				return
			}
			for len(stack) > 0 {
				opTop := stack[len(stack)-1]
				if opTop.tokenType == "leftBrac" {
					stack = stack[:len(stack)-1]
					if len(stack) > 0 {
						opTop = stack[len(stack)-1]        // now check if stack top is a function and if so, pop it off as well
						if opTop.tokenType == "function" { // this fix is not on the wiki page for shunting-yard algorithm
							stack = stack[:len(stack)-1]             // pop top off stack
							rpnSlice = append(rpnSlice, opTop.token) // write function out
						}
					}
					break
				}
				stack = stack[:len(stack)-1] // pop top off stack
				rpnSlice = append(rpnSlice, opTop.token)
			}
		}
	}
	// tokens all read so pop any remaining operators on the stack
	for len(stack) > 0 {
		if stack[len(stack)-1].token == "(" {
			errorInfix = " ** parenthesis do not match"
			rpn = ""
			return
		}
		rpnSlice = append(rpnSlice, stack[len(stack)-1].token)
		stack = stack[:len(stack)-1]

	}
	rpn = strings.Join(rpnSlice, " ")
	return
}

// used to make x3_45 into x3_{45}
// takes simple variables with underscore and makes them latex valid
func latexifyVar(inVar string) (outVar string) {
	var reLatex = regexp.MustCompile(`(?m)_(?P<result>.*)`)
	outVar = reLatex.ReplaceAllString(inVar, "_{$result}")
	return
}
