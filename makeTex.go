package main

import (
	"bytes"
	"fmt"
	"math"
	"math/rand"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type option struct {
	name  string
	value string
}

type strEqn struct {
	partial string
	eqn     int
}

func makeTex(problemInput, randomStr, outFlag, version string, inFile, outFile fileInfo) (string, string) {
	var inLines []string
	var inLine string
	var verbatim bool
	var logOut, errorHeader, commentSymbol string
	var texOut, fileRunInkscape string
	var reRemoveEndStuff = regexp.MustCompile(`(?m)\s*$`) // to delete blank space \r \n tabs etc at end of line
	var reOrgComment = regexp.MustCompile(`(?m)^# `)
	var reLatexComment = regexp.MustCompile(`(?m)^\s*%`)
	var svgList []string // list of all svg files that need inkscape to make pdf and pdf_tex files
	// done as a list so that inkscape only needs to be called once using inkscape --shell (makes it much faster than calling inkscape multiple times)

	orgHeader := `
#+OPTIONS: toc:nil author:nil email:nil creator:nil timestamp:nil
#+OPTIONS: html-postamble:nil num:nil
`
	orgBegSol := `
-----
* Solution
:PROPERTIES:
:CUSTOM_ID: problem-solution
:END:
`
	orgBegAns := `
-----
* Answer
:PROPERTIES:
:CUSTOM_ID: problem-answer
:END:
`
	texBegSol := `
\beginSolution
`
	texBegAns := `
\beginAnswer
`

	// using map here as I want to be able to iterate over key names as well
	// as looking at value for each key
	// these are configuration parameters
	// for format, the number represents the number of significant digits
	// in the case of DL, digits is forced to 2 after the decimal sign.
	// fmtVal: when \val is used
	// fmtRun(): the values inside brackets when RUN() is used
	// fmtRunEQ: when RUN= is used
	var configParam = map[string]string{ // defaults shown below
		"random":       randomStr, // can be false, true, any positive integer, min, max, minMax
		"nomVar":       "1.3:5",   // variation from x/k to kx : number of choices
		"fmtVal":       "U4",      // can be E, S, D, DL, or U (engineering, sci, decimal, dollar or SI Units)
		"fmtRun()":     "E4",      // can be E, S, D, DL (engineering, sci, decimal, dollar)
		"fmtRunEQ":     "U4",      // can be E, S, D, DL, or U (engineering, sci, decimal, dollar or SI Units)
		"verbose":      "false",   // can be true or false
		"defaultUnits": "",        // place defaultUnits here [[iI:A][vV:V][rR:\Omega]]  etc
		// if first letter of a variable is i or I then default units is A
	}
	// when DL (dollar) is used, the symbol "$" is NOT automatically added and can be added by user.
	// DL (dollar) results in number being nnn.nn (two digits after decimal point)

	varAll := make(map[string]varSingle) // IMPORTANT to use this type of map assignment - tried another and it worked for a while
	// til hash table memory changed and then memory errors on run that could not be traced by debugger
	// the downside of this approach is when changing a key value struct element, need to copy struct first then change
	// struct element then copy it back into hash table.  Can not change just a single struct element without this copy first

	// WITH VARIABLE MAP, keywords variation/random/sigDigits are used and defaults set below
	// stored in MAP so user can change them using \runParam{random=true}... during run of program

	// var key string  ... test code
	// for key = range configParam {
	// 	fmt.Println(configParam[key])
	// }
	verbatim = false
	switch outFile.ext {
	case ".org", ".otex":
		commentSymbol = "# "
		texOut = orgHeader + "\n"
	case ".tex":
		commentSymbol = "% "
	default: // should never be here
		fmt.Println("should not be here 01")
	}
	errorHeader = commentSymbol + "Created with problem2tex: version = " + version + "\n\n"
	inLines = strings.Split(problemInput, "\n")
	for i := range inLines {
		inLine = reRemoveEndStuff.ReplaceAllString(inLines[i], "")
		switch inLine {
		case "#+BEGIN_EXAMPLE":
			verbatim = true
			texOut = texOut + inLine + "\n"
			continue // skip to end of for lines and continue
			// different than break... break ends for loop
		case "#+END_EXAMPLE":
			verbatim = false
			texOut = texOut + inLine + "\n"
			continue // skip to end of for lines and continue
		default: // do nothing if here
		}
		if verbatim { // if true we are in verbatim mode so just skip the lines til out of verbatim
			texOut = texOut + inLine + "\n"
			continue
		}
		switch outFile.ext {
		case ".org", ".otex":
			if reOrgComment.MatchString(inLine) || inLine == "" { // either a comment line or a blank line so add \n and skip
				texOut = texOut + inLine + "\n"
				continue
			}
		case ".tex":
			if reLatexComment.MatchString(inLine) || inLine == "" { // either a comment line or a blank line so add \n and skip
				texOut = texOut + inLine + "\n"
				continue
			}
		default: // should never be here as option can only be orgMode or latex
			fmt.Println("should not be here 02")
		}
		inLine, svgList, logOut = commandReplace(inLine, inFile, outFile, varAll, configParam, false, svgList)
		if logOut != "" {
			errorHeader = errorHeader + logOutError(logOut, i)
		}
		inLine, logOut = fixHilite(inLine, outFile.ext)
		inLine = changeDollarDelimiters(inLine)
		if logOut != "" {
			errorHeader = errorHeader + logOutError(logOut, i)
		}
		if inLine != "" {
			texOut = texOut + inLine + "\n"
		}
	}
	switch outFlag {
	case "flagQuestion": // get rid of solution and answer
		var reSol = regexp.MustCompile(`(?msU)^\s*BEGIN{SOLUTION}.*^\s*END{SOLUTION}\s*\n`)
		var reAns = regexp.MustCompile(`(?msU)^\s*BEGIN{ANSWER}.*^\s*END{ANSWER}\s*\n`)
		texOut = reSol.ReplaceAllString(texOut, "")
		texOut = reAns.ReplaceAllString(texOut, "")
	case "flagSolAns": // keep all
		var reBegSol = regexp.MustCompile(`(?mU)^\s*BEGIN{SOLUTION}`)
		var reBegAns = regexp.MustCompile(`(?mU)^\s*BEGIN{ANSWER}`)
		var reEndSol = regexp.MustCompile(`(?mU)^\s*END{SOLUTION}`)
		var reEndAns = regexp.MustCompile(`(?mU)^\s*END{ANSWER}`)
		switch outFile.ext { // add headers for solution and answer
		case ".org", ".otex":
			texOut = reBegSol.ReplaceAllString(texOut, orgBegSol)
			texOut = reBegAns.ReplaceAllString(texOut, orgBegAns)
		case ".tex":
			texOut = reBegSol.ReplaceAllString(texOut, texBegSol)
			texOut = reBegAns.ReplaceAllString(texOut, texBegAns)
		default: // should never be here
			fmt.Println("should not be here 03")
		}
		texOut = reEndSol.ReplaceAllString(texOut, "")
		texOut = reEndAns.ReplaceAllString(texOut, "")
	case "flagAnswer": // get rid of solution
		var reBegAns = regexp.MustCompile(`(?mU)^\s*BEGIN{ANSWER}`)
		var reEndAns = regexp.MustCompile(`(?mU)^\s*END{ANSWER}`)
		var reSol = regexp.MustCompile(`(?msU)^\s*BEGIN{SOLUTION}.*^\s*END{SOLUTION}\s*\n`)
		texOut = reSol.ReplaceAllString(texOut, "")
		switch outFile.ext { // add headers for answer
		case ".org", ".otex":
			texOut = reBegAns.ReplaceAllString(texOut, orgBegAns)
		case ".tex":
			texOut = reBegAns.ReplaceAllString(texOut, texBegAns)
		default: // should never be here
			fmt.Println("should not be here 04")
		}
		texOut = reEndAns.ReplaceAllString(texOut, "")
	case "flagSolution": // get rid of answer
		var reBegSol = regexp.MustCompile(`(?mU)^\s*BEGIN{SOLUTION}`)
		var reEndSol = regexp.MustCompile(`(?mU)^\s*END{SOLUTION}`)
		var reAns = regexp.MustCompile(`(?msU)^\s*BEGIN{ANSWER}.*^\s*END{ANSWER}\s*\n`)
		texOut = reAns.ReplaceAllString(texOut, "")
		switch outFile.ext { // add headers for solution
		case ".org", ".otex":
			texOut = reBegSol.ReplaceAllString(texOut, orgBegSol)
		case ".tex":
			texOut = reBegSol.ReplaceAllString(texOut, texBegSol)
		default: // should never be here
			fmt.Println("should not be here 05")
		}
		texOut = reEndSol.ReplaceAllString(texOut, "")
	default:
		errorHeader = errorHeader + "ERROR: " + outFlag + " is not a valid outFlag parameter\n"
	}
	switch outFile.ext {
	case ".org":
	case ".tex", ".otex":
		// Now we need to create pdf and pdf_tex files for svg files if svgList has elements in it
		if len(svgList) > 0 {
			fileRunInkscape = `#!/bin/bash
echo "export-latex; export-area-drawing;`
			for i := range svgList {
				fileRunInkscape = fileRunInkscape + svgList[i]
			}
			fileRunInkscape = fileRunInkscape + `" | inkscape --shell
`
			fileWriteString(fileRunInkscape, "runInkscape.sh")
			logOut = runCommand("chmod", false, "+x", "runInkscape.sh")
			if logOut != "" {
				errorHeader = errorHeader + logOut
			}
			logOut = runCommand("./runInkscape.sh", true)
			if logOut != "" {
				errorHeader = errorHeader + logOut
			}
		}
		texOut = org2texFix(texOut)
	default: // should never be here
		fmt.Println("should not be here 06")
	}
	return texOut, errorHeader
}

func fixHilite(inString, fileExt string) (outString, logOut string) {
	var inStrSlice []strEqn
	var reTex = regexp.MustCompile(`(?m)HILITE{`)
	var reMark = regexp.MustCompile(`(?mU)HILITE{(?P<res1>.*)}`)
	outString = ""
	inStrSlice, logOut = findEqn(inString)
	switch fileExt {
	case ".tex":
		for i := range inStrSlice {
			switch inStrSlice[i].eqn {
			case 1, 2: // an equation phrase
				inStrSlice[i].partial = reTex.ReplaceAllString(inStrSlice[i].partial, "\\hiliteEQN{")
			default:
				inStrSlice[i].partial = reTex.ReplaceAllString(inStrSlice[i].partial, "\\hilite{")
			}
		}
	case ".org", ".otex":
		for i := range inStrSlice {
			switch inStrSlice[i].eqn {
			case 1, 2: // an equation phrase
				inStrSlice[i].partial = reTex.ReplaceAllString(inStrSlice[i].partial, "\\bbox[yellow]{")
			default:
				inStrSlice[i].partial = reMark.ReplaceAllString(inStrSlice[i].partial, "@@html:<mark>@@$res1@@html:</mark>@@")
			}
		}
	default: // should never be here
		fmt.Println("should not be here 07")
	}
	for i := range inStrSlice {
		outString = outString + inStrSlice[i].partial
	}
	return
}

func changeDollarDelimiters(inString string) (outString string) {
	var inStrSlice []strEqn
	var reInline = regexp.MustCompile(`(?m)^\$(?P<res1>.*)\$$`)
	var reEqn = regexp.MustCompile(`(?m)^\$\$(?P<res1>.*)\$\$$`)
	outString = ""
	inStrSlice, _ = findEqn(inString)
	for i := range inStrSlice {
		switch inStrSlice[i].eqn {
		case 1: // an inline equation
			inStrSlice[i].partial = reInline.ReplaceAllString(inStrSlice[i].partial, "\\($res1\\)")
		case 2:
			inStrSlice[i].partial = reEqn.ReplaceAllString(inStrSlice[i].partial, "\\[$res1\\]")
		default:
		}
	}
	for i := range inStrSlice {
		outString = outString + inStrSlice[i].partial
	}
	return
}

// recall... with string="1234" has len(string)=4
// first char is string[0:1] while first 3 char are string[0:3]

// breaks string into normal strings and equation strings and indicates with strEqn.eqn
// inStrSlice.eqn = 0, 1, or 2.  0 no eqn: 1 is inline eqn: 2 is non-inline eqn
func findEqn(inString string) (inStrSlice []strEqn, logOut string) {
	var temp strEqn
	var phraseEnd int // phrase end position index (either non-eqn or eqn phrase)
	var eqnNum int    // 0 means normal, 1 means inline eqn, 2 means non-inline eqn
	var tmpStr string
	inString = "X" + inString + "X" // add first and last char so that \$ might be at start of string
	// and can check for $$ at end of string
	// now must start i at 1 so it skips first string char and end at len(string)-1
	eqnNum = 0
	phraseEnd = 0
	for i := 1; i < len(inString)-1; i++ {
		tmpStr = inString[i-1 : i+2]
		switch {
		case tmpStr[0:1] != "\\" && tmpStr[1:2] == "$" && tmpStr[2:3] != "$":
			eqnNum = 1           // found start of an inline equation
			if i > phraseEnd+1 { // found a non-equation string so write it to inStrSlice
				temp.partial = inString[phraseEnd+1 : i]
				temp.eqn = 0
				inStrSlice = append(inStrSlice, temp)
			}
			// now look for end of inline equation
			for j := i + 1; j < len(inString); j++ {
				tmpStr = inString[j-1 : j+1]
				if j == len(inString)-1 { // got to end of line without finding end of equation
					temp.partial = inString[i : j+1]
					temp.eqn = 1
					inStrSlice = append(inStrSlice, temp)
					phraseEnd = j
					i = phraseEnd
					break
				}
				if tmpStr[0:1] != "\\" && tmpStr[1:2] == "$" { // found end of equation
					eqnNum = 0
					temp.partial = inString[i : j+1]
					temp.eqn = 1
					inStrSlice = append(inStrSlice, temp)
					phraseEnd = j
					i = phraseEnd
					break
				}
			}
		case tmpStr[0:1] != "\\" && tmpStr[1:2] == "$" && tmpStr[2:3] == "$":
			eqnNum = 2           // found start of a non-inline equation
			if i > phraseEnd+1 { // found a non-equation string so write it to inStrSlice
				temp.partial = inString[phraseEnd+1 : i]
				temp.eqn = 0
				inStrSlice = append(inStrSlice, temp)
			}
			i++
			// now look for end of non-inline equation
			for j := i + 1; j < len(inString)-1; j++ {
				tmpStr = inString[j-1 : j+2]
				if j == len(inString)-2 { // got to end of line without finding end of equation
					temp.partial = inString[i-1 : j+2]
					temp.eqn = 2
					inStrSlice = append(inStrSlice, temp)
					phraseEnd = j + 1
					i = phraseEnd
					break
				}
				if tmpStr[0:1] != "\\" && tmpStr[1:2] == "$" && tmpStr[2:3] == "$" { // found end of equation
					eqnNum = 0
					temp.partial = inString[i-1 : j+2]
					temp.eqn = 2
					inStrSlice = append(inStrSlice, temp)
					phraseEnd = j + 1
					i = phraseEnd
					break
				}
			}
		default:
		}

	}
	if phraseEnd+1 < len(inString)-1 {
		temp.partial = inString[phraseEnd+1 : len(inString)-1]
		temp.eqn = 0
		inStrSlice = append(inStrSlice, temp)
	}
	if eqnNum != 0 { // if 1 or 2, then unbalanced equation delimiters
		logOut = "Unbalanced equation delimiters"
	}
	return
}

func org2texFix(inString string) (outString string) {
	// for simpler syntax, replace "$$stuff_here$$" with \begin{equation}stuff_here\end{equation}"
	var reReplaceDoubleDollar = regexp.MustCompile(`(?m)\$\$(?P<res1>.*)\$\$`)
	outString = reReplaceDoubleDollar.ReplaceAllString(inString, "\\[$res1\\]")
	// now need to make inserted comments latex proper...so change "# " to "% "
	var reFindOrgComments = regexp.MustCompile(`(?m)^# `)
	outString = reFindOrgComments.ReplaceAllString(outString, "% ")
	var reFindSectionHead = regexp.MustCompile(`(?m)^\* (?P<res1>.*)`)
	outString = reFindSectionHead.ReplaceAllString(outString, "\\section*{$res1}")
	var reFindSubSectionHead = regexp.MustCompile(`(?m)^\*\* (?P<res1>.*)`)
	outString = reFindSubSectionHead.ReplaceAllString(outString, "\\subsection*{$res1}")
	var reFindSubSubSectionHead = regexp.MustCompile(`(?m)^\*\*\* (?P<res1>.*)`)
	outString = reFindSubSubSectionHead.ReplaceAllString(outString, "\\subsubsection*{$res1}")
	var reHorizLine = regexp.MustCompile(`(?m)^-----.*`)
	outString = reHorizLine.ReplaceAllString(outString, "\\noindent\\rule{\\textwidth}{1pt}")
	var reSkipLine = regexp.MustCompile(`(?m)^\s*\\\\\s*$`)
	outString = reSkipLine.ReplaceAllString(outString, "\n\\vspace{\\baselineskip}\n")
	var reBegVerb = regexp.MustCompile(`(?m)^\#\+BEGIN_EXAMPLE$`)
	outString = reBegVerb.ReplaceAllString(outString, "\\begin{verbatim}")
	var reEndVerb = regexp.MustCompile(`(?m)^\#\+END_EXAMPLE$`)
	outString = reEndVerb.ReplaceAllString(outString, "\\end{verbatim}")
	return
}

// Print out the logOut
// Function used to write and error line
func logOutError(logOut string, lineNum int) string {
	var outString string
	if lineNum != -1 { // dont include line number if lineNum = -1
		logOut = logOut + " - Line number: " + strconv.Itoa(lineNum+1)
	}
	outString = "ERROR" + ": " + logOut + "\n\n"
	fmt.Print(outString)
	return outString
}

// commandReplace looks for VAL and RUN  commands, executes those commands and replaces each with appropriate output
// it is a bit complicated as it needs to do VAL and RUN commands in order in case they are all on same line
// Example RUN{x=1} VAL{x} RUN{x=2} VAL{x} should give 1 for first VAL and 2 for second VAL
func commandReplace(inString string, inFile, outFile fileInfo, varAll map[string]varSingle, configParam map[string]string, graphic bool, svgList []string) (string, []string, string) {
	// graphic is a bool that if true implies we are replacing things in a graphic file (instead of a .prb file)
	// in this case, only VAL commands are done (no RUN commands)
	var head, tail, replace, logOut, newLog, leftMost, tmpCmd string
	var reFirstValMatch = regexp.MustCompile(`(?mU)VAL{`)
	var reFirstRunMatch = regexp.MustCompile(`(?mU)RUN{`)
	if !graphic {
		tmpCmd, newLog = getInsideStuff(inString, "CONFIG") // get the string INSIDE of { }
		if newLog != "" {
			logOut = logOut + " " + newLog
		}
		if tmpCmd != "" { // found a CONFIG command
			replace, logOut = runConfigFunc(tmpCmd, configParam)
			return replace, svgList, logOut
		}
		tmpCmd, newLog = getInsideStuff(inString, "PARAM")
		if newLog != "" {
			logOut = logOut + " " + newLog
		}
		if tmpCmd != "" { // found a PARAM command
			replace, logOut = runParamFunc(tmpCmd, varAll, configParam)
			return replace, svgList, logOut
		}
		tmpCmd, newLog = getInsideStuff(inString, "INCLUDE")
		if newLog != "" {
			logOut = logOut + " " + newLog
		}
		if tmpCmd != "" { // found an INCLUDE command
			replace, svgList, logOut = runInclude(tmpCmd, inFile, outFile, varAll, configParam, svgList) //need inFile/outFile to know where to get/put files
			return replace, svgList, logOut
		}
	}
	// If here, then no CONFIG, PARAM, or INCLUDE command found so perhaps VAL or RUN command
	for reFirstValMatch.MatchString(inString) || reFirstRunMatch.MatchString(inString) { // chec for VAL or RUN command
		// keep doing this loop until all VAL, RUN commands are done
		if reFirstValMatch.MatchString(inString) && reFirstRunMatch.MatchString(inString) {
			// if true, then both VAL and RUN are in inString and must do the most left one first
			locateVal := reFirstValMatch.FindStringIndex(inString)
			locateRun := reFirstRunMatch.FindStringIndex(inString)
			if locateVal[0] < locateRun[0] {
				leftMost = "VAL"
			} else {
				leftMost = "RUN"
			}
		} else { // if only one of VAL or RUN found then do the most leftmost for that one
			locateVal := reFirstValMatch.FindStringIndex(inString)
			if locateVal == nil {
				leftMost = "RUN"
			} else {
				leftMost = "VAL"
			}
		}
		switch leftMost {
		case "VAL":
			head, tail, replace, newLog = valReplace(inString, varAll, configParam)
			if newLog != "" {
				logOut = logOut + " " + newLog
			}
		case "RUN":
			if !graphic { // also run these commands below if graphic is false
				head, tail, replace, newLog = runReplace(inString, varAll, configParam)
				if newLog != "" {
					logOut = logOut + " " + newLog
				}
			}
		default: // should not get here
			fmt.Println("should not be here 08")
		}
		leftMost = ""
		inString = head + replace + tail
	}
	return inString, svgList, logOut
}

func getInsideStuff(inString, command string) (string, string) { // get the stuff between brackets ...  command{inside stuff}
	var reWordCmd = regexp.MustCompile(`(?mU)^\s*` + command + `(?P<res1>{.*)$`)
	var insideStuff, logOut string
	if !reWordCmd.MatchString(inString) {
		return "", logOut
	}
	insideStuff, _, logOut = matchBrackets(reWordCmd.FindStringSubmatch(inString)[1], "{")
	return insideStuff, logOut
}

func runReplace(inString string, varAll map[string]varSingle, configParam map[string]string) (string, string, string, string) {
	var head, tail, logOut, runCmd, replace, assignVar, format string
	var answer float64
	var result []string
	var reFirstRunCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)RUN(?P<res2>{.*)$`)
	result = reFirstRunCmd.FindStringSubmatch(inString)
	head = result[1]
	runCmd, tail, _ = matchBrackets(result[2], "{")
	assignVar, runCmd, answer, format, logOut = runCode(runCmd, varAll, configParam)
	if logOut != "" {
		return head, tail, replace, logOut
	}
	switch format {
	case "eqnVal":
		replace = valSymReplace(runCmd, "val", varAll, configParam)
	case "eqnSym":
		replace = valSymReplace(runCmd, "symbol", varAll, configParam)
	case "silent": // run statement but do not print anything
		replace = ""
	case "short2": // run statement and print statement (ex: v_2 = 3*V_t)
		if assignVar == "" {
			replace = value2Str(answer, "", configParam["fmtVal"]) // not an assignment statment so just return  answer
		} else {
			replace = valSymReplace(runCmd, "symbol", varAll, configParam)
		}
	case "short1": // run statement and print out statement = result (with units) (ex: v_2 = 3*V_t = 75mV)
		if assignVar == "" {
			replace = "error: not an assignment statement"
		} else {
			replace = valSymReplace(runCmd, "symbol", varAll, configParam) + " = " + value2Str(varAll[assignVar].value, varAll[assignVar].units, configParam["fmtRunEQ"])
		}
	case "long", "": // same as () but include result (ex: v_2 = 3*V_t = 3*(25e-3)=75mV) // THIS IS THE DEFAULT
		replace = valSymReplace(runCmd, "symbol", varAll, configParam) + " = " + valSymReplace(rightSide(runCmd), "()val", varAll, configParam) + " = " + value2Str(varAll[assignVar].value, varAll[assignVar].units, configParam["fmtRunEQ"])
	default:
		logOut = "Not a valid RUN format type: " + format
		replace = logOut
	}
	replace = latexifyEqn(replace)
	return head, tail, replace, logOut
}

func latexifyEqn(inString string) string {
	var outString string
	outString, _ = fixRunCmd2("PARLL", inString)
	outString, _ = fixRunCmd2("DIV", outString)
	outString, _ = fixRunCmd1("abs", outString)
	return outString
}

func valReplace(inString string, varAll map[string]varSingle, configParam map[string]string) (string, string, string, string) {
	var head, tail, errCode, expAndFormat, logOut string
	var formatStr, formatType, exp, replace, sigDigits string
	var ok bool
	var value float64
	var result []string
	var reFirstValCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)VAL(?P<res2>{.*)$`)
	var reComma = regexp.MustCompile(`(?m)^\s*(?P<res1>.*)\s*,\s*(?P<res2>.*)\s*$`)
	result = reFirstValCmd.FindStringSubmatch(inString) // found a val command
	head = result[1]
	expAndFormat, tail, logOut = matchBrackets(result[2], "{") // expression and format
	// if a comma exists in expAndFormat, then format is included after comma
	// if no comma, then only expresion is present
	if reComma.MatchString(expAndFormat) {
		result = reComma.FindStringSubmatch(expAndFormat)
		exp = result[1]
		formatStr = result[2]
		switch formatStr {
		case "=", "sym":
			formatType = formatStr
		case "U":
			formatType = formatStr
			_, sigDigits, _ = parseFormat(configParam["fmtVal"])
			formatStr = formatType + sigDigits
		default:
			formatType, sigDigits, logOut = parseFormat(formatStr)
			if logOut != "" {
				return head, tail, logOut, logOut
			}
		}
	} else { // no comma and so no format given then use default
		exp = expAndFormat
		formatType, sigDigits, _ = parseFormat(configParam["fmtVal"])
		formatStr = formatType + sigDigits
	}
	// we now know the formatType and sigDigits to use so carry on ...
	_, ok = varAll[exp] // check that exp is a variable in the varAll map
	if ok {
		// exp is a variable in varAll map
		switch formatType {
		case "E", "S", "D", "DL", "U":
			replace = value2Str(varAll[exp].value, varAll[exp].units, formatStr)
		case "=":
			replace = varAll[exp].latex + "=" + value2Str(varAll[exp].value, varAll[exp].units, configParam["fmtVal"])
		case "sym": // if sym then print out symbol instead of value
			replace = varAll[exp].latex
		default:
			logOut = "format: " + formatType + " *** NOT A VALID FORMAT"
			return head, tail, "", logOut
		}
	} else {
		// exp is an expression and not in varAll map
		_, _, value, _, errCode = runCode(exp, varAll, configParam)
		if errCode != "" {
			logOut = "expression: " + exp + " *** NOT A VALID EXPRESSION"
			return head, tail, errCode, logOut
		}
		replace = value2Str(value, varAll[exp].units, formatStr)
	}
	return head, tail, replace, logOut
}

func bracketCheck(inString string, leftBrac string) (logOut string) {
	var rightBrac string
	var count int
	switch leftBrac {
	case "{":
		rightBrac = "}"
	case "(":
		rightBrac = ")"
	case "[":
		rightBrac = "]"
	case "<":
		rightBrac = ">"
	default:
	}
	count = 0
	for i := range inString {
		if string(inString[i]) == leftBrac {
			count++
		}
		if string(inString[i]) == rightBrac {
			count--
		}
		if count < 0 {
			logOut = "-- Unmatched brackets: more " + rightBrac + " than " + leftBrac + " --"
			return
		}
	}
	if count > 0 {
		logOut = "-- Unmatched brackets: more " + leftBrac + " than " + rightBrac + " --"
	}
	return
}

// used to update the VAL{} commands in inString
// also changes dollar delimiters in case of .org output
func valUpdate(inString, fileExt string, varAll map[string]varSingle, configParam map[string]string) (outString, logOut string) {
	var inLines, dummy []string            // dummy not used but needed to make commandReplace work as it sometimes is svgList
	var inFileDummy, outFileDummy fileInfo // also not needed but required in commandReplace
	inLines = strings.Split(inString, "\n")
	for i := range inLines {
		inLines[i], _, logOut = commandReplace(inLines[i], inFileDummy, outFileDummy, varAll, configParam, true, dummy)
		if logOut != "" {
			logOut = logOut + " - error in external file: "
			return
		}
		switch fileExt {
		case ".org", ".otex":
			inLines[i] = changeDollarDelimiters(inLines[i])
		default: // do nothing
		}
	}
	outString = strings.Join(inLines, "\n")
	return
}

func runInclude(inCmd string, inFile, outFile fileInfo, varAll map[string]varSingle, configParam map[string]string, svgList []string) (string, []string, string) {
	var options = map[string]string{ // defaults shown below
		"textScale":  "1.0", // Scale size of text (in case of svg)
		"trimTop":    "0",   // trim top of svg
		"trimBottom": "0",   //  trim bottom of svg
		"trimLeft":   "0",   // trim left of svg
		"trimRight":  "0",   // trim right of svg
		"scale":      "1.0", //  Determines size of figure (in px) Only currently works for png, jpg.
	}
	var allOptions []option
	var replace, optionStr, logOut string
	var fileNameAdd, inFileName, outFileName, inFileStr string
	var fullFileName, svgFileNameNoExt, width, fullPathFileNameNoExt, tmpStr string
	var widthDefault float64
	var svgFileName string
	var result []string
	var fileName, fileExt string
	var reNameInfo = regexp.MustCompile(`(?m)^\s*(?P<res1>\w+)\.(?P<res2>\w+)`)
	if !reNameInfo.MatchString(inCmd) {
		logOut = "INCLUDE command does not have filename in correct format\n Should look like filename.ext"
		return "", svgList, logOut
	}
	result = reNameInfo.FindStringSubmatch(inCmd)
	fileName = result[1]
	fileExt = result[2]
	optionStr = getAfter(inCmd, "#") // get options after "#" character
	allOptions = getAllOptions(optionStr)
	for i := 0; i < len(allOptions); i++ { // First check if any option errors depending on file type
		switch fileExt {
		case "png", "jpg": // just checking fileExt is valid
			logOut = "No options are allowed for a .png or .jpg file in an INCLUDE command"
			return "", svgList, logOut
		case "svg", "asc":
		case "tex": // there should be no options for a tex file
			logOut = "No options are allowed for a .tex file in an INCLUDE command"
			return "", svgList, logOut
		default:
			logOut = "File extension not recognized for INCLUDE command: ." + fileExt
			return "", svgList, logOut
		}
		switch allOptions[i].name {
		case "scale", "trimTop", "trimBottom", "trimLeft", "trimRight", "textScale":
			_, err := strconv.ParseFloat(allOptions[i].value, 64) // check if option value is a decimal number
			if err != nil {
				logOut = allOptions[i].value + " is not a valid decimal number for " + allOptions[i].name
				return "", svgList, logOut
			}
			options[allOptions[i].name] = allOptions[i].value
		default:
			logOut = allOptions[i].name + " is not a valid option"
			return "", svgList, logOut
		}
	}
	fileNameAdd = "NEW"
	switch outFile.ext {
	case ".tex", ".otex":
		switch fileExt {
		case "png", "jpg", "jpeg", "pdf": // need to fix this
			fullFileName = fileName + `.` + fileExt // need .extension here as final includegraphics command needs that info
			replace = `\incPic{` + fullFileName + `}{` + width + `}{` +
				options["spaceAbove"] + `}{` + options["spaceHoriz"] + `}{` +
				options["spaceBelow"] + `}`
		case "svg", "tex":
			inFileStr, logOut = fileReadString(filepath.Join(inFile.path, fileName+"."+fileExt))
			if logOut != "" {
				return replace, svgList, logOut
			}
			inFileStr, logOut = valUpdate(inFileStr, outFile.ext, varAll, configParam) // update VAL{} elements to values
			if logOut != "" {
				return replace, svgList, logOut
			}
			fileWriteString(inFileStr, filepath.Join(outFile.path, fileName+fileNameAdd+"."+fileExt))
			switch fileExt {
			case "svg":
				svgFileNameNoExt = fileName + fileNameAdd
				fullPathFileNameNoExt = filepath.Join(outFile.path, svgFileNameNoExt)
				widthDefault = 100
				width = fmt.Sprintf("%.2f", widthDefault*str2float(options["scale"]))
				replace = `\incSvg{` + svgFileNameNoExt + `}{` + width + `}{` +
					options["trimLeft"] + `}{` + options["trimBottom"] + `}{` +
					options["trimRight"] + `}{` + options["trimTop"] + `}{` + options["textScale"] + `}`
				tmpStr = "file-open:" + fullPathFileNameNoExt + ".svg; export-filename:" + fullPathFileNameNoExt + ".pdf; export-do; "
				svgList = append(svgList, tmpStr)
			case "tex":
				fullFileName = fileName + fileNameAdd + `.tex`
				replace = `\incTex{` + fullFileName + `}`
			default: // should never be here
				fmt.Println("should not be here 09")
			}
		case "asc":
			svgFileNameNoExt = fileName + fileNameAdd + "asc"
			svgFileName = svgFileNameNoExt + ".svg"
			fullPathFileNameNoExt = filepath.Join(outFile.path, svgFileNameNoExt)
			logOut = runCommand("ltspice2svg", false, "-export="+filepath.Join(outFile.path, svgFileName), "-text=latex", filepath.Join(inFile.path, fileName+".asc"))
			if logOut != "" {
				return "ERROR: " + logOut, svgList, logOut
			}
			inFileStr, logOut = fileReadString(filepath.Join(outFile.path, svgFileName))
			if logOut != "" {
				return replace, svgList, logOut
			}
			inFileStr, logOut = valUpdate(inFileStr, outFile.ext, varAll, configParam) // update VAL{} elements to values
			if logOut != "" {
				return replace, svgList, logOut
			}
			fileWriteString(inFileStr, filepath.Join(outFile.path, svgFileName))
			widthDefault = 100
			width = fmt.Sprintf("%.2f", widthDefault*str2float(options["scale"]))
			replace = `\incSvg{` + svgFileNameNoExt + `}{` + width + `}{` +
				options["trimLeft"] + `}{` + options["trimBottom"] + `}{` +
				options["trimRight"] + `}{` + options["trimTop"] + `}{` + options["textScale"] + `}`
			tmpStr = "file-open:" + fullPathFileNameNoExt + ".svg; export-filename:" + fullPathFileNameNoExt + ".pdf; export-do; "
			svgList = append(svgList, tmpStr)
		default:
			logOut = "File extension not recognized: " + fileExt
		}
	case ".org":
		switch fileExt {
		case "png", "jpg":
			inFileName = filepath.Join(inFile.path, fileName+"."+fileExt)
			outFileName = filepath.Join(outFile.path, fileName+fileNameAdd+"."+fileExt)
			logOut = runCommand("cp", false, inFileName, outFileName)
			if logOut != "" {
				return replace, svgList, logOut
			}
			replace = "#+attr_html: :width " + options["width"] + "px\n"
			replace = replace + "[[./" + fileName + fileNameAdd + "." + fileExt + "]]"
			// original filename.svg becomes filenameNEW.svg in tmp directory
			// original filename.asc becomes filenameNEW.asc in tmp then filenameNEWasc.svg in tmp
		case "asc":
			svgFileName = fileName + fileNameAdd + "asc.svg"
			logOut = runCommand("ltspice2svg", false, "-export="+filepath.Join(outFile.path, svgFileName), "-text=latex", filepath.Join(inFile.path, fileName+".asc"))
			if logOut != "" {
				return replace, svgList, logOut
			}
			inFileStr, logOut = fileReadString(filepath.Join(outFile.path, svgFileName))
			if logOut != "" {
				return replace, svgList, logOut
			}
			inFileStr, logOut = valUpdate(inFileStr, outFile.ext, varAll, configParam) // update VAL{} elements to values
			if logOut != "" {
				return replace, svgList, logOut
			}
			// resize and trim svg
			inFileStr = svgResize(inFileStr, options["trimTop"], options["trimBottom"], options["trimLeft"], options["trimRight"], options["scale"])
			fileWriteString(inFileStr, filepath.Join(outFile.path, svgFileName))
			replace = `#+INCLUDE: "./` + svgFileName + `" export html`
		case "svg":
			inFileStr, logOut = fileReadString(filepath.Join(inFile.path, fileName+".svg"))
			if logOut != "" {
				return replace, svgList, logOut
			}
			inFileStr, logOut = valUpdate(inFileStr, outFile.ext, varAll, configParam) // update VAL{} elements to values
			if logOut != "" {
				return replace, svgList, logOut
			}
			// resize and trim svg
			inFileStr = svgResize(inFileStr, options["trimTop"], options["trimBottom"], options["trimLeft"], options["trimRight"], options["scale"])
			fileWriteString(inFileStr, filepath.Join(outFile.path, fileName+fileNameAdd+".svg"))
			replace = `#+INCLUDE: "./` + fileName + fileNameAdd + `.svg" export html`
		default:
			logOut = "File extension not recognized: " + fileExt
		}
	default:
		fmt.Println("should not be here 10")
	}
	return replace, svgList, logOut
}

// to run a command line instruction
func runCommand(program string, onlyMainError bool, args ...string) string {
	var out bytes.Buffer    // used for cmd.run for better output errors
	var stderr bytes.Buffer // used for cmd.run for better output errors
	var logOut string
	var err error
	var outStr, stderrStr string
	cmd := exec.Command(program, args...)
	//	cmd.Dir = inFile.path
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err = cmd.Run()
	outStr = out.String()
	stderrStr = stderr.String()
	if err != nil {
		logOut = fmt.Sprint(fmt.Sprint(err))
		return logOut
	}
	if !onlyMainError {
		if outStr != "" {
			logOut = fmt.Sprint(outStr)
			return logOut
		}
		if stderrStr != "" {
			logOut = fmt.Sprint(stderrStr)
			return logOut
		}
	}
	return logOut
}

func runConfigFunc(optionStr string, configParam map[string]string) (string, string) {
	// returns outString that is only used when verbose is selected as an option... otherwise it is a null string
	// also returns logOut if an error is detected
	// THIS FUNCTION MODIFIES configParam map!!!! (it is essentially a global variable)
	var outString, logOut string
	var formatType, key string
	var allOptions []option
	allOptions = getAllOptions(optionStr)
	for i := 0; i < len(allOptions); i++ {
		switch allOptions[i].name { // Below is a check to ensure the input is in the correct format
		case "random":
			_, logOut = checkRandom(allOptions[i].value)
		case "nomVar":
			_, _, logOut = convertNomVar(allOptions[i].value)
		case "fmtVal", "fmtRun()", "fmtRunEQ":
			formatType, _, logOut = parseFormat(allOptions[i].value)
			switch formatType {
			case "E", "S", "D", "DL":
			case "U":
				if allOptions[i].name == "fmtRun()" { // U is not allowed in fmtRun()
					logOut = "ERROR: " + allOptions[i].name + " = " + allOptions[i].value + " -> config setting can be either E, S, D, or DL followed by number"
				}
			default:
				logOut = "ERROR: " + allOptions[i].name + " = " + allOptions[i].value + " -> config setting can be either E, S, D, DL or U followed by number"
			}
		case "verbose":
			switch allOptions[i].value {
			case "true", "":
				configParam["verbose"] = "true"
				outString = "\\begin{verbatim}\n\nConfiguration Settings\n"
				for key = range configParam {
					if len(key) > 1 { // only print out config settings when key string length is greater than 1
						outString = outString + key + " : " + configParam[key] + "\n"
						// this is done so defaultUnits map values are not each printed
					}
				}
				outString = outString + "\\end{verbatim}"
			case "false":
			default:
				logOut = "verbose can be either true or false"
			}
		case "defaultUnits":
			configParam["defaultUnits"] = allOptions[i].value
			logOut = defaultUnitsFunc(allOptions[i].value, configParam) // THIS FUNCTION MODIFIES configParam MAP!!!
			if logOut != "" {
				logOut = "defaultUnits syntax is incorrect" + logOut
			}
		case "syntax":
			switch allOptions[i].value {
			case "orgMode", "latex":
			default:
				logOut = allOptions[i].value + " not a valid syntax option -- syntax option can only be orgMode or latex"
			}
		default:
			logOut = allOptions[i].name + " is not a valid CONFIG option"
			return "", logOut
		}
		if logOut != "" {
			return "", logOut
		}
		configParam[allOptions[i].name] = allOptions[i].value // everything looks good so update the configParam map
	}
	return outString, logOut
}

func runParamFunc(statement string, varAll map[string]varSingle, configParam map[string]string) (string, string) {
	var assignVar, rightSide, logOut string
	var units, prefix, outVerbose, optionStr string
	var value, factor, nominal float64
	var allOptions []option
	var values []float64
	var num, random int
	var result []string
	var min, max, stepSize float64
	var reEqual = regexp.MustCompile(`(?m)^\s*(?P<res1>\w+)\s*=\s*(?P<res2>.*)\s*`)
	var reArray = regexp.MustCompile(`(?m)^\s*\[(?P<res1>.*)\]\s*(?P<res2>.*)`)
	var reStep = regexp.MustCompile(`(?m)^\s*(?P<res1>\S+)\s*;\s*(?P<res2>\S+)\s*;\s*(?P<res3>\S+)[#|\s]*`)
	var reKFactor = regexp.MustCompile(`(?m)^\s*(?P<res1>[^#|\s]+)`) // match everything up to a # or space
	if reEqual.MatchString(statement) {
		result = reEqual.FindStringSubmatch(statement)
		assignVar = result[1]                    // assignVar is the left side of the "=" sign.
		rightSide = strings.TrimSpace(result[2]) // rightSide is the rightside of "=" sign.  then trim whitespace from beginning and end
		assignVar, logOut = checkReserved(assignVar, logOut)
		if logOut != "" {
			return "", logOut
		}
		tmp2, ok := varAll[assignVar]
		if !ok { // if !ok then this is the first time assigning this variable in varAll map
			// add default units and latexify version of assignVar
			varAll[assignVar] = varSingle{}
			tmp2 = varAll[assignVar]
			tmp2.units = defaultUnitsVar(assignVar, configParam) // add default units for assignVar (if defined in defaultUnits config parameter)
			tmp2.latex = latexifyVar(assignVar)                  // add latex version of assignVar
		}
		switch {
		case reArray.MatchString(rightSide): // it is an array runParam statement
			result = reArray.FindStringSubmatch(rightSide)
			values, logOut = findArrayValues(result[1])
			if logOut != "" {
				return "", logOut
			}
		case reStep.MatchString(rightSide): // it is a step runParam statement
			result = reStep.FindStringSubmatch(rightSide)
			min, logOut = str2Float64(result[1])
			if logOut != "" {
				return "", logOut
			}
			max, logOut = str2Float64(result[2])
			if logOut != "" {
				return "", logOut
			}
			stepSize, logOut = str2Float64(result[3])
			if logOut != "" {
				return "", logOut
			}
			if stepSize < 0 {
				logOut = "step size must be greater than zero"
				return "", logOut
			}
			if max < min {
				logOut = "max value must be larger than min value"
				return "", logOut
			}
			if ((max - min) / stepSize) > 100 {
				logOut = "step size is too small and results in more than 100 values"
				return "", logOut
			}
			for x := min; x <= max; x = x + stepSize {
				values = append(values, x)
			}
		case reKFactor.MatchString(rightSide): // it is a nomVar statement
			result = reKFactor.FindStringSubmatch(rightSide)
			nominal, logOut = str2Float64(result[1])
			if logOut != "" {
				return "", logOut
			}
			values = append(values, nominal) // the nominal value is value[0] so it is default
			factor, num, logOut = convertNomVar(configParam["nomVar"])
			if logOut != "" {
				return "", logOut
			}
			// Now want (1/k)^(2i/(num-1)) and (k)^(2i/(num-1)) times nominal for i=1,(num-1)/2
			// for all the other values
			for i := (num - 1) / 2; i >= 1; i = i - 1 {
				accuracy := "2"
				if math.Pow(factor, (2/(float64(num)-1))) < 1.09 { // less than 10 percent diff between values
					accuracy = "3" // increase accuracy since small variation between values
				}
				tmpNum := nominal * math.Pow(factor, (2*float64(i)/(float64(num)-1)))
				tmpNum, _ = strconv.ParseFloat(fmt.Sprintf("%."+accuracy+"g", tmpNum), 64)
				values = append(values, tmpNum)
				tmpNum = nominal * math.Pow(1/factor, (2*float64(i)/(float64(num)-1)))
				tmpNum, _ = strconv.ParseFloat(fmt.Sprintf("%."+accuracy+"g", tmpNum), 64)
				values = append(values, tmpNum)
			}
		default:
			logOut = "not a valid \\runParam statement"
			return "", logOut
		}
		// Now deal with options in the PARAM command
		optionStr = getAfter(rightSide, "#") // the options stuff after #
		allOptions = getAllOptions(optionStr)
		for i := 0; i < len(allOptions); i++ {
			switch allOptions[i].name {
			case "units":
				prefix, units = getPrefixUnits(allOptions[i].value) // separate preUnits into prefix and units
				tmp2.units = units
			case "symbol":
				tmp2.latex = allOptions[i].value
			default:
				logOut = allOptions[i].name + " is not a valid option"
				return "", logOut
			}
		}
		if configParam["verbose"] == "true" {
			outVerbose = "\\begin{verbatim}" + assignVar + " = ["
			for i := range values {
				outVerbose = outVerbose + fmt.Sprintf("%.3g", values[i])
				if i < len(values)-1 {
					outVerbose = outVerbose + ","
				}
			}
			outVerbose = outVerbose + "] units=" + prefix + units + "   symbol=" + tmp2.latex + "\\end{verbatim}"
		}
		random, logOut = checkRandom(configParam["random"])
		switch random {
		case 0: // if random == 0, then num = 0 so first element is chosen
			num = 0
		case -1: // if random == -1, then  num is a random in between 0 and values-1 (based on machine time so pretty much really random)
			num = rand.Intn(len(values))
		case -2, -3, -4: // min, max, minMax case
			sort.Float64s(values) // sort values - lowest at 0 highest at len(values)-1
			switch random {
			case -2:
				num = 0
			case -3:
				num = len(values) - 1
			case -4: // choose either min or max value
				if rand.Intn(2) == 0 {
					num = 0
				} else {
					num = len(values) - 1
				}
			default: // should never be here
				fmt.Println("should not be here 11")
			}
		default: // if here, random is a seed so use it to get the next random
			random = psuedoRand(random) // update random based on the last random value (treat last one as seed)
			num = randInt(len(values), random)
			configParam["random"] = strconv.Itoa(random)
		}
		value = values[num] * prefix2float(prefix)
		tmp2.value = value
		varAll[assignVar] = tmp2
	}
	return outVerbose, logOut
}

func getAfter(inString, charac string) string { // used to find rest of string after "charac" string in inString
	var outString string
	var reAfter = regexp.MustCompile(`(?m)` + charac + `(?P<res1>.*)$`)
	if !reAfter.MatchString(inString) {
		return ""
	}
	outString = reAfter.FindStringSubmatch(inString)[1]
	return outString
}

func convertNomVar(inString string) (float64, int, string) {
	var factor float64
	var num int
	var logOut string
	var result []string
	var err error
	var reKFactor = regexp.MustCompile(`(?m)^\s*(?P<res1>\S+)\s*:\s*(?P<res2>\S+)\s*$`)
	if reKFactor.MatchString(inString) {
		result = reKFactor.FindStringSubmatch(inString)
		factor, logOut = str2Float64(result[1])
		if logOut != "" {
			return factor, num, logOut
		}
		if factor <= 1 {
			logOut = "variation must be greater than 1"
			return factor, num, logOut
		}
		num, err = strconv.Atoi(result[2])
		if err != nil {
			logOut = "incorrect syntax for nomVar: should be variation:number"
			return factor, num, logOut
		}
		if num < 1 {
			logOut = "variation:number ... number should be > 0"
			return factor, num, logOut
		}
		if num%2 == 0 { // if num is even make it odd by adding one so that values are equally
			// spaced above and below nominal value
			num = num + 1
		}
	} else {
		logOut = "incorrect syntax for nomVar: should be variation:number"
	}
	return factor, num, logOut
}

// findArrayValues returns a slice of float64 from a comma or space delimited string of numbers
func findArrayValues(inString string) ([]float64, string) {
	var values []float64
	var logOut string
	var result []string
	var tmpNum float64
	var re0 = regexp.MustCompile(`(?m)^\s*(?P<res1>[^,\s]+)(?P<res2>.*)$`) // find first number and rest of line
	var re1 = regexp.MustCompile(`(?m)^\s*,*\s*`)                          // match any space and , at beginning of line
	for re0.MatchString(inString) {
		result = re0.FindStringSubmatch(inString)
		tmpNum, logOut = str2Float64(result[1])
		if logOut != "" {
			return values, logOut
		}
		values = append(values, tmpNum)
		inString = result[2]                          // rest of line
		inString = re1.ReplaceAllString(inString, "") // delete , and spaces at beginning of line
	}
	return values, logOut
}

func getPrefixUnits(prefixUnits string) (prefix string, units string) {
	var result []string
	var re0 = regexp.MustCompile(`(?m)^(P|T|G|M|k|m|\\mu|n|p|f|a)\s*(?P<res2>\S+)`)
	var re1 = regexp.MustCompile(`(?m)^m\s*\w`)
	if re0.MatchString(prefixUnits) {
		result = re0.FindStringSubmatch(prefixUnits)
		prefix = result[1]
		units = result[2]
		if prefix == "m" {
			if re1.MatchString(prefixUnits) {
				// "m" is a prefix so leave as is
			} else {
				// "m" is for meter so set prefix to blank and all are units
				prefix = ""
				units = prefixUnits
			}
		}
	} else {
		units = prefixUnits
	}
	return
}

// delete element of string slice while maintaing order
func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

// valSymReplace function used to replace all variables in an equation with their values in varAll
// replace is either ()val, val, or symbol (that determines how variables are replaced)
func valSymReplace(inString, replace string, varAll map[string]varSingle, configParam map[string]string) string {
	var sub string
	var result []string
	var re1 = regexp.MustCompile(`(?m)[a-zA-Z][a-zA-Z_0-9]*`) // find words ... a word starts with letter then might follow with letter/number/_
	var re2 = regexp.MustCompile(`(?m)`)                      // just a declare as it will change below
	inString = inString + " "                                 // extra space added to make the substitutions below work correctly
	result = re1.FindAllString(inString, -1)
	for i := range result {
		_, ok := varAll[result[i]]
		if ok {
			re2 = regexp.MustCompile(`(?m)` + result[i] + `([^\w])`) // need the [^\w] so it does not catch another variable that starts the same
			switch replace {
			case "()val":
				sub = "{(" + value2Str(varAll[result[i]].value, "", configParam["fmtRun()"]) + ")}"
			case "val":
				sub = "{" + value2Str(varAll[result[i]].value, "", configParam["fmtRun()"]) + "}"
			case "symbol":
				sub = "{" + varAll[result[i]].latex + "}"
			default:
			}
			if re2.MatchString(inString) {
				inString = re2.ReplaceAllString(inString, sub+`$1`)
			}
		}
	}
	return inString
}

func rightSide(inString string) string { // return rightside of a run command assignment
	var outString string
	var re0 = regexp.MustCompile(`(?m)=(?P<res1>.*)$`) // get stuff after = to end
	if re0.MatchString(inString) {
		outString = re0.FindStringSubmatch(inString)[1]
	}
	return outString
}

func value2Str(x float64, units, formatStr string) (outString string) {
	var formatType, sigDigits string
	formatType, sigDigits, _ = parseFormat(formatStr)
	switch formatType {
	case "E": // engineering notation (powers of 3 for exponent)
		significand, exponent, _ := float2Parts(x, strIncrement(sigDigits, -1))
		if exponent == "0" {
			outString = significand
		} else {
			outString = significand + "e" + exponent
		}
	case "S": // scientific notation
		outString = fmt.Sprintf("%."+strIncrement(sigDigits, -1)+"e", x)
	case "D": // decimal notation
		outString = removeTrailingZeros(fmt.Sprintf("%."+strIncrement(sigDigits, 0)+"f", x))
	case "DL": // dollar notation (2 decimal places and rounded off)
		outString = fmt.Sprintf("%.2f", math.Round(x*100)/100)
	case "U": // SI notation and includes units if available
		significand, exponent, prefix := float2Parts(x, strIncrement(sigDigits, -1))
		if units == "" {
			if exponent == "0" {
				outString = significand
			} else {
				outString = significand + "e" + exponent
			}
		} else {
			outString = significand + " \\mathrm{" + prefix + " " + units + "}"
		}
	default:
		outString = "format not recognized: " + formatType
	}
	return
}

// convert a float64 number into a string of parts in engineering form
// returns significand (ex: 2.354 or 23.54 or 235.4)
// returns exponent (ex: 0 or 3 or 9 or -3 or -6)
// returns prefix (ex: "" or k or G or m or \mu)
func float2Parts(x float64, sigDigits string) (significand string, exponent string, prefix string) {
	var xSci string
	var result []string
	var expInt int
	var signifFloat float64
	var reGetParts = regexp.MustCompile(`(?m)^\s*(?P<res1>.*)e(?P<res2>.*)$`)
	xSci = fmt.Sprintf("%."+sigDigits+"e", x)
	if reGetParts.MatchString(xSci) {
		result = reGetParts.FindStringSubmatch(xSci)
		significand = result[1]
		exponent = result[2]
		expInt, _ = strconv.Atoi(exponent)
		signifFloat, _ = strconv.ParseFloat(significand, 64)
		if expInt == -1 { // special case for as prefer to write at 0.1V instead of 100mV
			expInt = expInt + 1
			signifFloat = signifFloat / 10
		} else {
			for expInt%3 != 0 { // do until exponent is a multiple of 3 (engineering and SI notation)
				expInt = expInt - 1
				signifFloat = 10 * signifFloat
			}
		}
		significand = fmt.Sprintf("%f", signifFloat)
		significand = removeTrailingZeros(significand)
		exponent = strconv.Itoa(expInt)
		prefix = exponent2Prefix(exponent)
	}
	return
}

func removeTrailingZeros(inString string) string { // removes .00000 if at end of a string to make a number look better
	var outString string
	var reZeros = regexp.MustCompile(`(?m)\.?0*$`)
	outString = reZeros.ReplaceAllString(inString, "")
	return outString
}

func prefix2float(prefix string) (x float64) {
	switch prefix {
	case "a":
		x = 1e-18
	case "f":
		x = 1e-15
	case "p":
		x = 1e-12
	case "n":
		x = 1e-9
	case "\\mu":
		x = 1e-6
	case "m":
		x = 1e-3
	case "":
		x = 1
	case "k":
		x = 1e3
	case "M":
		x = 1e6
	case "G":
		x = 1e9
	case "T":
		x = 1e12
	case "P":
		x = 1e15
	default: // unrecognized prefix
		fmt.Println("unrecognized prefix")
	}
	return
}

func exponent2Prefix(exponent string) string {
	exp2Prefix := make(map[string]string)
	exp2Prefix["-18"] = "a"
	exp2Prefix["-15"] = "f"
	exp2Prefix["-12"] = "p"
	exp2Prefix["-9"] = "n"
	exp2Prefix["-6"] = "\\mu"
	exp2Prefix["-3"] = "m"
	exp2Prefix["0"] = ""
	exp2Prefix["3"] = "k"
	exp2Prefix["6"] = "M"
	exp2Prefix["9"] = "G"
	exp2Prefix["12"] = "T"
	exp2Prefix["15"] = "P"
	return exp2Prefix[exponent]
}

func deCommentLatex(inString string) (string, string) {
	// remove latex comments but leave \%
	// This is done for ALL lines so it takes out % in JULIA CODE AS WELL
	// might have to modify this so this func does not affect Julia code if % needed in Julia
	var comment string
	var re0 = regexp.MustCompile(`(?m)^(?P<res1>%.*)$`)     // strip off comments where % is at beginning of line
	var re1 = regexp.MustCompile(`(?m)(?P<res1>[^\\]%.*)$`) // strip off rest unless \% (since that is a real %)
	if re0.MatchString(inString) {
		comment = re0.FindStringSubmatch(inString)[1]
		inString = re0.ReplaceAllString(inString, "")
		return inString, comment
	}
	if re1.MatchString(inString) {
		comment = re1.FindStringSubmatch(inString)[1]
		inString = re1.ReplaceAllString(inString, "")
		return inString, comment
	}
	return inString, comment
}

func matchBrackets(inString, leftBrac string) (string, string, string) {
	// returns the enclosed values inside outside matching brackets - first return
	// as well as rest of string after outside matching brackets - second return
	// also return logOut - third return
	// eg: inString = "{here 1{a}}{here 2}" results in... inside = "here 1{a}" and tail="{here 2}"
	var inside, rightBrac, tail, logOut string
	var openBr int
	switch leftBrac {
	case "{":
		rightBrac = "}"
	case "(":
		rightBrac = ")"
	case "[":
		rightBrac = "]"
	case "<":
		rightBrac = ">"
	default:
	}
	openBr = 0
	for i := 0; i < len(inString); i++ {
		if string(inString[i]) == leftBrac {
			openBr++
			for j := i + 1; j < len(inString); j++ {
				switch string(inString[j]) {
				case leftBrac:
					openBr++
				case rightBrac:
					openBr--
				default:
				}
				if openBr == 0 {
					inside = inString[i+1 : j]
					if j+1 <= len(inString) {
						tail = inString[j+1:]
					}
					return inside, tail, ""
				}

			}

		}
	}
	if openBr > 0 {
		logOut = "There is no closing backet: \"" + rightBrac + "\""
	}
	return inside, tail, logOut
}

func strIncrement(inString string, k int) string {
	// take in a string representing an integer, add k to it and return incremented value as string
	i, _ := strconv.Atoi(inString)
	i = i + k
	outString := strconv.FormatInt(int64(i), 10)
	return outString
}

func function2Latex(inString string) string {
	var result []string
	var funcInput, head, tail string
	var re0 = regexp.MustCompile(`(?m)`)
	for key := range func1 {
		re0 = regexp.MustCompile(`(?mU)^(?P<res1>.*\W)(?P<res2>` + key + `\(.*)$`)
		for re0.MatchString(inString) {
			result = re0.FindStringSubmatch(inString)
			head = result[1]
			tail = result[2]
			funcInput, tail, _ = matchBrackets(tail, "(")
			if key == "log10" { // latex command cannot be \log10 since numbers not allowed
				key = "logten" // change to logten so that \logten{} is used for latex
			}
			inString = head + "\\" + key + "{" + funcInput + "}" + tail
		}
	}
	return inString
}

// recursively fixes PARLL and DIV functions for printing as latex
// stuffHere**PARLL(R1,PARLL(R2,R3))**moreStuff becomes stuffHere**R1||R2||R3**moreStuff
func fixRunCmd2(runCmd, inString string) (string, string) {
	var result []string
	var outString, head, tail, inside, var1, var2, logOut string
	var reRunCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)` + runCmd + `(?P<res2>\(.*)$`)
	var reInside = regexp.MustCompile(`(?m)^(?P<res1>.*),(?P<res2>.*)$`)
	outString = inString // default if matching below does not occur
	for reRunCmd.MatchString(outString) {
		if reRunCmd.MatchString(outString) {
			result = reRunCmd.FindStringSubmatch(outString)
			head = result[1]
			inside, tail, _ = matchBrackets(result[2], "(")
			inside, _ = fixRunCmd2(runCmd, inside)
			if reInside.MatchString(inside) {
				result = reInside.FindStringSubmatch(inside)
				var1 = result[1]
				var2 = result[2]
				switch runCmd {
				case "PARLL":
					outString = head + var1 + "||" + var2 + tail
				case "DIV":
					outString = head + `\frac{` + var1 + `}{` + var2 + `}` + tail
				default:
				}
			}
		}
	}
	return outString, logOut
}

// recursively fixes abs functions for printing as latex
// stuffHere**abs(R1+abs(R2))**moreStuff becomes stuffHere**|R1+|R2||**moreStuff
func fixRunCmd1(runCmd, inString string) (string, string) {
	var result []string
	var outString, head, tail, inside, var1, logOut string
	var reRunCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)` + runCmd + `(?P<res2>\(.*)$`)
	var reInside = regexp.MustCompile(`(?m)^(?P<res1>.*)$`)
	outString = inString // default if matching below does not occur
	for reRunCmd.MatchString(outString) {
		if reRunCmd.MatchString(outString) {
			result = reRunCmd.FindStringSubmatch(outString)
			head = result[1]
			inside, tail, _ = matchBrackets(result[2], "(")
			inside, _ = fixRunCmd1(runCmd, inside)
			if reInside.MatchString(inside) {
				result = reInside.FindStringSubmatch(inside)
				var1 = result[1]
				switch runCmd {
				case "abs":
					outString = head + "|" + var1 + "|" + tail
				default:
				}
			}
		}
	}
	return outString, logOut
}

func randInt(N, random int) int {
	// based on random (a random number), choose an
	// int from 0 to N-1
	// This has a slight bias it to it but if random is much bigger than N, the bias is quite small
	var choice = random % N
	return choice
}

func psuedoRand(x0 int) int {
	// A linear congruential generator (LCG) based on
	// https://en.wikipedia.org/wiki/Linear_congruential_generator
	// it returns an psuedorandom integer between 100000 and 999999
	var a, c, m, x1 int
	if x0 < 0 { // correct x0 if it happens to be less than 0
		x0 = -1 * x0
	}
	a = 707106 // 1e6/sqrt(2) and truncated
	c = 1
	m = 999983 // largest prime number less than 1e6
	x1 = 0
	for x1 < 100000 { // loop until find a new random number larger than 100000
		x1 = (a*x0 + c) % m
		x0 = x1
	}
	return x1
}

func checkReserved(variable, logOut string) (string, string) {
	var key string
	for key = range func1 {
		if variable == key {
			logOut = logOut + key + " is a reserved variable and cannot be assigned"
			variable = key + "-IsReservedVariable"
		}
	}
	for key = range func2 {
		if variable == key {
			logOut = logOut + key + " is a reserved variable and cannot be assigned"
			variable = key + "-IsReservedVariable"
		}
	}
	return variable, logOut
}

func syntaxError(statement, cmdType string) string {
	var errCode, shouldBeBlank string
	// valid characters in statement
	var reValidRunCode0 = regexp.MustCompile(`(?m)[\w|=|*|/|+|\-|^|(|)|\s|\.|,]+`) // valid characters in statement
	var reValidVal0 = regexp.MustCompile(`(?m)[\w|\s]+`)
	var reValidVal1 = regexp.MustCompile(`(?m)[\w|*|/|+|\-|^|(|)|\s|\.|,]+`)
	var reValidRunParam0 = regexp.MustCompile(`(?m)[\w|=|\s|\.|,|#|\\|\[|\]|:]+`)

	// first check if all characters in statement are valid characters
	switch cmdType {
	case "val":
		shouldBeBlank = reValidVal1.ReplaceAllString(statement, "")
		if shouldBeBlank != "" {
			errCode = "character(s) " + shouldBeBlank + " should not be in a \\val statement"
		}
	case "val=":
		shouldBeBlank = reValidVal0.ReplaceAllString(statement, "")
		if shouldBeBlank != "" {
			errCode = "character(s) " + shouldBeBlank + " should not be in a \\val= statement"
		}
	case "runCode":
		shouldBeBlank = reValidRunCode0.ReplaceAllString(statement, "")
		if shouldBeBlank != "" {
			errCode = "character(s) " + shouldBeBlank + " should not be in a \\run statement"
		}
	case "runParam":
		shouldBeBlank = reValidRunParam0.ReplaceAllString(statement, "")
		if shouldBeBlank != "" {
			errCode = "character(s) " + shouldBeBlank + " should not be in a \\runParam statement"
		}
	default:
		errCode = "should not be here in syntaxError"
	}
	return errCode
}

func str2Float64(numStr string) (float64, string) {
	var x float64
	var logOut string
	var err error
	x, err = strconv.ParseFloat(numStr, 64)
	if err != nil {
		logOut = numStr + " is not a valid number"
		x = 1.2345678e123
	}
	return x, logOut
}

func defaultUnitsFunc(inString string, configParam map[string]string) string {
	// This function takes in defaultUnits notation and updates configParam map to include default values
	// for variables that start with certain letters
	// For example if inString = [[iI:A][g:mA/V]] then default units for variables that start with
	// i or I are "A" while default units for variables that start with g are "mA/V"
	// This is done by adding configParam["i"]="A" and configParam["I"]="A" and configParam["g"]="mA/V" to configParam map
	var tail, logOut, letters, units string
	var result []string
	var re0 = regexp.MustCompile(`(?m)(?P<res1>[a-zA-Z]+):(?P<res2>.+)`)
	inString, tail, _ = matchBrackets(inString, "[") // removing outside brackets and just left with inside ... eg: [iI:A][g:mA/V]
	if tail != "" {
		logOut = "defaultUnits syntax is incorrect: extra characters after final \"]\""
		return logOut
	}
	for { // do this loop til no more [thing] elements left
		inString, tail, logOut = matchBrackets(inString, "[")
		if logOut != "" {
			return logOut
		}
		if inString == "" && tail == "" {
			break
		}
		if re0.MatchString(inString) {
			result = re0.FindStringSubmatch(inString)
			letters = result[1]
			units = result[2]
			for i := 0; i < len(letters); i++ {
				configParam[string(letters[i])] = units
			}
		} else {
			logOut = "defaultUnits syntax is incorrect inside \"[ ]\" element"
			return logOut
		}
		inString = tail
	}
	return logOut
}

func defaultUnitsVar(assignVar string, configParam map[string]string) string {
	var units, firstLetter string
	firstLetter = string(assignVar[0])
	_, ok := configParam[firstLetter]
	if ok { // if true then first letter of assignVar is a key in the configParam map and will have default units defined for it
		units = configParam[firstLetter]
	}
	return units
}

func getNextOption(inString string) (string, string) {
	var outString, tail string
	var result []string
	var reWithComma = regexp.MustCompile(`(?mU)^\s*(?P<res>\S.*[^\\]),\s*(?P<res2>\S.*)$`)
	var reNoComma = regexp.MustCompile(`(?mU)^\s*(?P<res1>\S.*)\s*$`)
	if reWithComma.MatchString(inString) {
		result = reWithComma.FindStringSubmatch(inString)
		outString = result[1]
		tail = result[2]
		return outString, tail
	}
	if reNoComma.MatchString(inString) {
		outString = reNoComma.FindStringSubmatch(inString)[1]
		tail = ""
	}
	return outString, tail
}

func getAllOptions(inString string) []option {
	var nextOption, tail string
	var result []string
	var newOption option
	var allOptions []option
	var reGetOptions = regexp.MustCompile(`(?m)(?P<res1>\S+)\s*=\s*(?P<res2>.*)$`)
	var reFixCommas = regexp.MustCompile(`(?m)\\,`)
	for inString != "" {
		nextOption, tail = getNextOption(inString)
		if nextOption == "" {
			return allOptions
		}
		if reGetOptions.MatchString(nextOption) {
			result = reGetOptions.FindStringSubmatch(nextOption)
			newOption = option{result[1], reFixCommas.ReplaceAllString(result[2], ",")}
		} else {
			newOption = option{nextOption, "true"} // if option has no = sign, then option is a word and set that word option to true
		}
		allOptions = append(allOptions, newOption)
		inString = tail
	}
	return allOptions
}
