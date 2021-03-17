package main

import (
	"fmt"
	"math"
	"math/rand"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func makeTex(problemInput, sigDigits, randomStr string, inFile, outFile fileInfo) (string, string) {
	var inLines []string
	var logOut, comment, errorHeader, spiceFile, spiceFilename string
	var texOut string
	var linesToRemove []int
	var reDeletethis = regexp.MustCompile(`(?m)\*\*deletethis\*\*`)
	var reNotBlankLine = regexp.MustCompile(`(?m)\S`)

	// using map here as I want to be able to iterate over key names as well
	// as looking at value for each key
	// these are configuration parameters
	var configParam = map[string]string{ // defaults shown below
		"paramRandom":    randomStr, // can be false, true, any positive integer, min, max, minMax
		"paramSigDigits": sigDigits, // number of significant digits to print
		"paramKFactor":   "1.3:5",   // variation from x/k to kx : number of choices
		"paramFormat":    "eng",     // can be eng, sci or decimal
		"paramVerbose":   "false",   // can be true or false
	}

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

	inLines = strings.Split(problemInput, "\n")
	for i := range inLines {
		inLines[i], comment = deCommentLatex(inLines[i])
		logOut = syntaxWarning(inLines[i])
		if logOut != "" {
			errorHeader = errorHeader + logOutError(logOut, i, "WARNING")
		}
		inLines[i], logOut = valRunReplace(inLines[i], varAll, configParam, false)
		if logOut != "" {
			errorHeader = errorHeader + logOutError(logOut, i, "ERROR")
		}
		inLines[i] = fixParll(inLines[i])
		inLines[i] = inLines[i] + comment // add back comment that was removed above
		inLines[i] = function2Latex(inLines[i])
		if reDeletethis.MatchString(inLines[i]) {
			inLines[i] = reDeletethis.ReplaceAllString(inLines[i], "")
			if !reNotBlankLine.MatchString(inLines[i]) { // if a blank line then add line number to linesToRemove list
				linesToRemove = append(linesToRemove, i) // do it later so that line numbers still correct if an error is reported after a line removal
			}
		}
		spiceFile, spiceFilename, logOut = checkLTSpice(inLines[i], inFile, outFile, sigDigits, varAll, configParam)
		if logOut != "" {
			errorHeader = errorHeader + logOutError(logOut, i, "ERROR")
		} else {
			fileWriteString(spiceFile, filepath.Join(outFile.path, spiceFilename+"_update.asc"))
		}
	}
	// remove lines that are slated for removal
	k := 0
	for i := range linesToRemove {
		inLines = remove(inLines, linesToRemove[i]-k) // need to subtract k as that those number of lines have already been removed
		k++
	}
	texOut = strings.Join(inLines, "\n")
	return texOut, errorHeader
}

// add comment notation and line number to logOut info and add carriage return
// Also print out the logOut
func logOutError(logOut string, lineNum int, typeErr string) string {
	var outString string
	if lineNum != -1 { // dont include line number if lineNum = -1
		logOut = logOut + " - Line number: " + strconv.Itoa(lineNum+1)
	}
	if typeErr == "" {
		outString = "% " + logOut + "\n" // use tex comment notation and add CR
	} else {
		outString = "% " + typeErr + ": " + logOut + "\n" // use tex comment notation and add CR
	}
	fmt.Print(outString)
	return outString
}

func valRunReplace(inString string, varAll map[string]varSingle, configParam map[string]string, ltSpice bool) (string, string) {
	// ltSpice is a bool that if true implies we are replacing things in a .asc file (instead of a .prb file)
	var head, tail, replace, logOut, newLog string
	var reFirstvalRunCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)\\(?P<res2>run.*|val.*)(?P<res3>{.*)$`)
	var reFirstvalCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)\\(?P<res2>val.*)(?P<res3>{.*)$`)
	var reFirstRunCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)\\(?P<res2>run.*)(?P<res3>{.*)$`)
	var reFixSpice = regexp.MustCompile(`(?mU)\\(?P<res1>\\val.*{)`)
	if ltSpice { // fix ltSpice file so all \\val.*{ are changed to \val.*{
		if reFixSpice.MatchString(inString) {
			inString = reFixSpice.ReplaceAllString(inString, "$res1")
		}
	}
	for reFirstvalRunCmd.MatchString(inString) { // chec for val or run command
		if reFirstvalCmd.MatchString(inString) { // check for a val command
			head, tail, replace, newLog = valReplace(inString, varAll, configParam) // found a val command
			if newLog != "" {
				logOut = logOut + " " + newLog
			}
		}
		if !ltSpice { // also run these commands below if ltSpice is false
			if reFirstRunCmd.MatchString(inString) { // check for a run command
				head, tail, replace, newLog = runReplace(inString, varAll, configParam)
				if newLog != "" {
					logOut = logOut + " " + newLog
				}
			}
		}
		inString = head + replace + tail
	}
	return inString, logOut
}

func runReplace(inString string, varAll map[string]varSingle, configParam map[string]string) (string, string, string, string) {
	var head, tail, logOut, runCmdType, runCmd, replace, assignVar string
	var answer float64
	var result []string
	var reFirstRunCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)\\(?P<res2>run.*)(?P<res3>{.*)$`)
	result = reFirstRunCmd.FindStringSubmatch(inString)
	head = result[1]
	runCmdType = result[2]
	runCmd, tail = matchBrackets(result[3], "{")
	replace = "" // so the old replace is not used
	switch runCmdType {
	case "runParam": // Used for setting parameters and config parameters
		replace, logOut = runParamFunc(runCmd, varAll, configParam)
		if replace == "" {
			replace = "**deletethis**"
		}
	case "runSilent": // run statement but do not print anything
		replace = "**deletethis**"
		_, _, _, logOut = runCode(runCmd, varAll)
	case "run": // run statement and print statement (ex: v_2 = 3*V_t)
		assignVar, runCmd, answer, logOut = runCode(runCmd, varAll)
		if assignVar == "" {
			replace = float2Str(answer, configParam) // not an assignment statment so just return  answer
		} else {
			replace = "\\mbox{$" + latexStatement(runCmd, varAll) + "$}"
		}
	case "run=": // run statement and print out statement = result (with units) (ex: v_2 = 3*V_t = 75mV)
		assignVar, runCmd, _, logOut = runCode(runCmd, varAll)
		if assignVar == "" {
			replace = "error: not an assignment statement"
		} else {
			replace = "\\mbox{$" + latexStatement(runCmd, varAll) + " = " + valueInSI(assignVar, varAll, configParam) + "$}"
		}
	case "run()": // same as run but include = bracket values in statement (ex" v_2 = 3*V_t = 3*(25e-3))
		_, runCmd, _, logOut = runCode(runCmd, varAll)
		replace = "\\mbox{$" + latexStatement(runCmd, varAll) + bracketed(runCmd, varAll, configParam) + "$}"
	case "run()=": // same as run() but include result (ex: v_2 = 3*V_t = 3*(25e-3)=75mV)
		assignVar, runCmd, _, logOut = runCode(runCmd, varAll)
		replace = "\\mbox{$" + latexStatement(runCmd, varAll) + bracketed(runCmd, varAll, configParam) + " = " + valueInSI(assignVar, varAll, configParam) + "$}"
	default:
		// if here, then error as \run**something else** is here
		logOut = "\\" + runCmdType + " *** NOT A VALID COMMAND\n"
	}
	return head, tail, replace, logOut
}

func valReplace(inString string, varAll map[string]varSingle, configParam map[string]string) (string, string, string, string) {
	var head, valCmdType, tail, replace, valCmd, key, logOut, tmp, newLog string
	var ok bool
	var answer float64
	var result []string
	var reFirstvalCmd = regexp.MustCompile(`(?mU)^(?P<res1>.*)\\(?P<res2>val.*)(?P<res3>{.*)$`)
	result = reFirstvalCmd.FindStringSubmatch(inString) // found a val command
	head = result[1]
	valCmdType = result[2]
	valCmd, tail = matchBrackets(result[3], "{")
	replace = "" // so the old replace is not used

	// first check if configParam going to be printed out and if so, then print it
	for key = range configParam {
		if valCmd == key {
			replace = configParam[key] // printing out a configParam
		}
	}
	if replace == "" { // no configParam found then do below
		switch valCmdType {
		case "val", "valNDec", "valNEng", "valNSci":
			_, _, answer, newLog = runCode(valCmd, varAll)
			if newLog == "" {
				if valCmdType == "val" {
					replace = float2Str(answer, configParam)
				} else {
					tmp = configParam["paramFormat"]
					switch valCmdType {
					case "valNDec":
						configParam["paramFormat"] = "decimal"
					case "valNEng":
						configParam["paramFormat"] = "eng"
					case "valNSci":
						configParam["paramFormat"] = "sci"
					default: // never here
					}
					replace = float2Str(answer, configParam)
					configParam["paramFormat"] = tmp
				}
			} else {
				replace = newLog
			}
		case "val=", "valU", "valLtx":
			_, ok = varAll[valCmd] // check that valCmd is a variable in the varAll map
			if ok {
				switch valCmdType {
				case "val=": // print out var = result (with SI units).  (ex: V_1 = 3V or v_{tx} = 23mV or D = 10km)
					replace = "\\mbox{$" + varAll[valCmd].latex + " = " + valueInSI(valCmd, varAll, configParam) + "$}"
				case "valU": // print out result (with SI units). (ex: 3V or 23mV or 10km)
					replace = "\\mbox{$" + valueInSI(valCmd, varAll, configParam) + "$}"
				case "valLtx":
					replace = "\\mbox{$" + varAll[valCmd].latex + "$}"
				default:
					// should never be here
					logOut = "can not be here 06"
				}
			} else {
				logOut = "variable " + valCmd + " is NOT DEFINED"
				replace = "\\mbox{$" + valCmd + " \\text{ NOT DEFINED}$}"
			}
		default:
			// if here, then \val**something else** found so an error message
			logOut = "\\" + valCmdType + " *** NOT A VALID COMMAND"
		}
	}
	return head, tail, replace, logOut
}

func checkLTSpice(inString string, inFile, outFile fileInfo, sigDigits string, varAll map[string]varSingle, configParam map[string]string) (string, string, string) {
	var spiceFilename, spiceFile, logOut string
	var inLines []string
	var reLTSpice = regexp.MustCompile(`(?mU)\\incProbLTspice.*{\s*(?P<res1>\S*)\s*}`)
	if reLTSpice.MatchString(inString) {
		spiceFilename = reLTSpice.FindStringSubmatch(inString)[1]
		spiceFile, logOut = fileReadString(filepath.Join(inFile.path, spiceFilename+".asc"))
		if logOut != "" {
			return "", "", logOut
		}
		spiceFile, _ = convertIfUtf16(spiceFile)
		inLines = strings.Split(spiceFile, "\n")
		for i := range inLines {
			inLines[i], logOut = valRunReplace(inLines[i], varAll, configParam, true)
			if logOut != "" {
				logOut = logOut + " - error in " + spiceFilename + ".asc"
				return "", "", logOut
			}
		}
		spiceFile = strings.Join(inLines, "\n")
		fileWriteString(spiceFile, filepath.Join(outFile.path, spiceFilename+"_update.asc"))
	}
	return spiceFile, spiceFilename, logOut
}

func syntaxWarning(statement string) (logOut string) { // check that syntax seems okay and give warning if not okay
	var reDollar = regexp.MustCompile(`(?m)\$`)
	logOut = bracketCheck(statement, "{")
	logOut = logOut + bracketCheck(statement, "(")
	logOut = logOut + bracketCheck(statement, "[")
	matches := reDollar.FindAllStringIndex(statement, -1)
	// need to count $ and see that they are even (backetCheck will not work here)
	if len(matches)%2 != 0 {
		logOut = logOut + "-- Uneven number of $ so likely unmatched --"
	}
	return
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

func runParamFunc(statement string, varAll map[string]varSingle, configParam map[string]string) (string, string) {
	var assignVar, rightSide, key, logOut string
	var units, latex, prefix, outVerbose string
	var value, factor, nominal float64
	var values []float64
	var num, random int
	var result []string
	var min, max, stepSize float64
	var reEqual = regexp.MustCompile(`(?m)^\s*(?P<res1>\w+)\s*=\s*(?P<res2>.*)\s*`)
	var reArray = regexp.MustCompile(`(?m)^\s*\[(?P<res1>.*)\]\s*(?P<res2>.*)`)
	var reOptions = regexp.MustCompile(`(?m)#(?P<res1>.*)$`)
	var reUnits = regexp.MustCompile(`(?m)\\paramUnits(?P<res1>{.*)$`)
	var reLatex = regexp.MustCompile(`(?m)\\paramLatex(?P<res1>{.*)$`)
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
		for key = range configParam {
			if assignVar == key { // it is a configParam runParam statement
				switch assignVar {
				case "paramRandom":
					_, logOut = checkRandom(rightSide)
					if logOut == "" {
						configParam["paramRandom"] = rightSide
					}
				case "paramSigDigits":
					// put check for paramSigDigits here
					configParam["paramSigDigits"] = rightSide
				case "paramFormat":
					switch rightSide {
					case "eng", "sci", "decimal":
						configParam["paramFormat"] = rightSide
					default:
						logOut = "paramFormat can be either eng, sci or decimal"
						return "", logOut
					}
				case "paramKFactor":
					_, _, logOut = convertKFactor(rightSide)
					if logOut != "" {
						return "", logOut
					}
					configParam["paramKFactor"] = rightSide
				case "paramVerbose":
					switch rightSide {
					case "true", "false":
						configParam["paramVerbose"] = rightSide
					default:
						logOut = "paramVerbose can be either true or false"
						return "", logOut
					}
				default:
					logOut = "should never be here 05"
				}
				return "", logOut
			}
		}
		tmp2, ok := varAll[assignVar]
		if !ok { // if !ok then this is the first time assigning this variable in varAll map
			varAll[assignVar] = varSingle{}
			tmp2 = varAll[assignVar]
			tmp2.latex = latexifyVar(assignVar) // add latex version of assignVar
			tmp2.units = defaultUnits(assignVar)
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
		case reKFactor.MatchString(rightSide): // it is a kFactor statement
			result = reKFactor.FindStringSubmatch(rightSide)
			nominal, logOut = str2Float64(result[1])
			if logOut != "" {
				return "", logOut
			}
			values = append(values, nominal) // the nominal value is value[0] so it is default
			factor, num, logOut = convertKFactor(configParam["paramKFactor"])
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
		if reOptions.MatchString(rightSide) {
			options := reOptions.FindStringSubmatch(rightSide)[1] // the options stuff after #
			if reUnits.MatchString(options) {
				tmp := reUnits.FindStringSubmatch(options)[1] // just the stuff {.*$
				preUnits, _ := matchBrackets(tmp, "{")        // a string that has prefix and units together
				prefix, units = getPrefixUnits(preUnits)      // separate preUnits into prefix and units
				tmp2.units = units
			}
			if reLatex.MatchString(options) {
				tmp := reLatex.FindStringSubmatch(options)[1]
				latex, _ = matchBrackets(tmp, "{")
				tmp2.latex = latex
			}
		}
		if configParam["paramVerbose"] == "true" {
			outVerbose = "% " + assignVar + " = ["
			for i := range values {
				outVerbose = outVerbose + fmt.Sprintf("%g", values[i]*prefix2float(prefix))
				if i < len(values)-1 {
					outVerbose = outVerbose + ","
				}
			}
			outVerbose = outVerbose + "]"
		}
		random, logOut = checkRandom(configParam["paramRandom"])
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
			}
		default: // if here, random is a seed so use it to get the next random
			random = psuedoRand(random) // update random based on the last random value (treat last one as seed)
			num = randInt(len(values), random)
			configParam["paramRandom"] = strconv.Itoa(random)
		}
		value = values[num] * prefix2float(prefix)
		tmp2.value = value
		varAll[assignVar] = tmp2
	}
	return outVerbose, logOut
}

func convertKFactor(inString string) (float64, int, string) {
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
			logOut = "kFactor must be greater than 1"
			return factor, num, logOut
		}
		num, err = strconv.Atoi(result[2])
		if err != nil {
			logOut = "incorrect syntax for \\runKFactor: should be kFactor:number"
			return factor, num, logOut
		}
		if num < 1 {
			logOut = "kFactor:number ... number should be > 0"
			return factor, num, logOut
		}
		if num%2 == 0 { // if num is even make it odd by adding one so that values are equally
			// spaced above and below nominal value
			num = num + 1
		}
	} else {
		logOut = "incorrect syntax for \\runKFactor: should be kFactor:number"
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

// bracketed function used to create () part of \run() where bracketed part are
// all numbers for the variables
func bracketed(statement string, varAll map[string]varSingle, configParam map[string]string) (outString string) {
	var backPart, sub string
	var result []string
	var re0 = regexp.MustCompile(`(?m)=(?P<res1>.*)$`) // get stuff after = to end
	var re1 = regexp.MustCompile(`(?m)(?P<res1>\w+)`)  // find all words
	var re2 = regexp.MustCompile(`(?m)`)               // just a declare as it will change below
	if re0.MatchString(statement) {
		backPart = re0.FindStringSubmatch(statement)[1]
		result = re1.FindAllString(backPart, -1)
		for i := range result {
			_, ok := varAll[result[i]]
			if ok {
				re2 = regexp.MustCompile(`(?m)` + result[i])
				sub = "(" + float2Str(varAll[result[i]].value, configParam) + ")"
				backPart = re2.ReplaceAllString(backPart, sub)
			}
		}
		outString = " = " + backPart
	}
	return
}

func valueInSI(variable string, varAll map[string]varSingle, configParam map[string]string) (outSI string) {
	significand, exponent, prefix := float2Parts(varAll[variable].value, strIncrement(configParam["paramSigDigits"], -1))
	if varAll[variable].units == "" {
		if exponent == "0" {
			outSI = significand
		} else {
			outSI = significand + "e" + exponent
		}
	} else {
		outSI = "\\mbox{$" + significand + " \\units{" + prefix + " " + varAll[variable].units + "}$}"
	}
	return
}

func float2Str(x float64, configParam map[string]string) (outString string) {
	outString = "should not occur 01"
	switch configParam["paramFormat"] {
	case "eng":
		significand, exponent, _ := float2Parts(x, strIncrement(configParam["paramSigDigits"], -1))
		if exponent == "0" {
			outString = significand
		} else {
			outString = significand + "e" + exponent
		}
	case "sci":
		outString = fmt.Sprintf("%."+strIncrement(configParam["paramSigDigits"], -1)+"e", x)
	case "decimal": // decimal unless values are very large or very small in which case -- sci
		outString = fmt.Sprintf("%."+strIncrement(configParam["paramSigDigits"], -1)+"g", x)
	default:
		outString = "should not occur 02"
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
	var reZeros = regexp.MustCompile(`(?m)\.?0*$`)
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
		significand = reZeros.ReplaceAllString(significand, "")
		exponent = strconv.Itoa(expInt)
		prefix = exponent2Prefix(exponent)
	}
	return
}

func latexStatement(statement string, varAll map[string]varSingle) string {
	var result, result2 []string
	var head, tail string
	var reWord = regexp.MustCompile(`(?m)[a-zA-Z][a-zA-Z_0-9]*`) // used to find all words in statement
	var re1 = regexp.MustCompile(`(?m)`)                         // just a way to declare re1 (it changes below)
	statement = statement + " "                                  // need extra space at end so search below works correctly if word is at end of statement
	result = reWord.FindAllString(statement, -1)
	for i := range result {
		_, ok := varAll[result[i]]
		if ok {
			re1 = regexp.MustCompile(`(?m)(?P<res1>.*\W|^)` + result[i] + `(?P<res2>\W.*)$`)
			tail = statement
			statement = ""
			for re1.MatchString(tail) {
				result2 = re1.FindStringSubmatch(tail)
				head = result2[1]
				tail = result2[2]
				statement = statement + head + varAll[result[i]].latex
			}
			statement = statement + tail
		}
	}
	return statement
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

func matchBrackets(inString, leftBrac string) (string, string) {
	// returns the first enclosed values inside outside matching brackets
	// as well as rest of string after outside matching brackets
	var inside, rightBrac, tail string
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
	openBr := 0
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
					return inside, tail
				}

			}

		}
	}
	return inside, tail
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
			funcInput, tail = matchBrackets(tail, "(")
			if key == "log10" { // latex command cannot be \log10 since numbers not allowed
				key = "logten" // change to logten so that \logten{} is used for latex
			}
			inString = head + "\\" + key + "{" + funcInput + "}" + tail
		}
	}
	return inString
}

func fixParll(inString string) string {
	var result []string
	var outString, head, tail, inside, var1, var2 string
	var reParll = regexp.MustCompile(`(?mU)^(?P<res1>.*)parll(?P<res2>\(.*)$`)
	var reInside = regexp.MustCompile(`(?m)^(?P<res1>.*),(?P<res2>.*)$`)
	outString = inString // default if matching below does not occur
	for reParll.MatchString(outString) {
		if reParll.MatchString(outString) {
			result = reParll.FindStringSubmatch(outString)
			head = result[1]
			inside, tail = matchBrackets(result[2], "(")
			inside = fixParll(inside)
			if reInside.MatchString(inside) {
				result = reInside.FindStringSubmatch(inside)
				var1 = result[1]
				var2 = result[2]
				outString = head + var1 + "||" + var2 + tail
			}
		}
	}
	return outString
}

func randInt(N, random int) int {
	// based on random (a random number), choose an
	// int from 0 to N-1
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
	m = 999999
	x1 = 0
	for x1 < 100000 {
		x1 = (a*x0 + c) % m
		x0 = x1
	}
	return x1
}

func checkReserved(variable, logOut string) (string, string) {
	var key string
	switch variable {
	case "parll": // can add more reserved variables here
		logOut = logOut + variable + " is a reserved variable and cannot be assigned"
		variable = variable + "IsReservedVariable"
	default:
	}
	for key = range func1 {
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
