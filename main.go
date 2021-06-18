package main

import (
	"math"
	"math/rand"
	"os"
	"time"
)

type tokenAndType struct {
	token     string
	tokenType string
}

var func2 = map[string]struct {
	name func(float64, float64) float64
	prec int // precedence value (higher is more priority)
}{
	"+":     {add, 2},
	"-":     {sub, 2},
	"*":     {mult, 3},
	"/":     {div, 3},
	"^":     {pow, 5},
	"parll": {parll, 5},
}

var func1 = map[string]func(float64) float64{
	// if adding a new function... may need to change preamble.tex
	"abs":   math.Abs,
	"asin":  math.Asin,
	"asind": asind, // arc sin(x) where x is in degrees
	"asinh": math.Asinh,
	"acos":  math.Acos,
	"acosd": acosd, // arc cos(x) where x is in degrees
	"acosh": math.Acosh,
	"atan":  math.Atan,
	"atand": atand, // arc tan(x) where x is in degrees (not radians)
	"atanh": math.Atanh,
	"ceil":  math.Ceil,
	"cos":   math.Cos,
	"cosd":  cosd, // cos(x) where x is in degrees
	"cosh":  math.Cosh,
	"exp":   math.Exp,
	"floor": math.Floor,
	"ln":    math.Log,   // also natural log
	"log":   math.Log,   // natural log
	"log10": math.Log10, // log base 10
	"round": math.Round,
	"sin":   math.Sin,
	"sind":  sind, // sin(x) where x is in degrees
	"sinh":  math.Sinh,
	"sqrt":  math.Sqrt,
	"tan":   math.Tan,
	"tand":  tand, // tan(x) where x is in degrees
	"tanh":  math.Tanh,
	"neg":   neg,
	"pos":   pos,
	"dB":    dB,
	"dBV":   dBV,
}

type varSingle struct { // a structure for each variable hold info below
	latex string  // the latex equivalent of the variable
	units string  // Just the units part (without the prefix)
	value float64 //The value in float64 format (should equal value)
}

// fileInfo is structure that contains file info (example: path1/path2/filename.ext)
type fileInfo struct {
	path string // the relative Directory location (path/path2)
	name string // the filename without extension (filename)
	ext  string // the filename extension (ext)
	full string // full path/path2/filename.ext
}

// input file name is a command line arg (no flag) (path/infile.prb or path/infile.asc)
// output file name is -export=path/outfile.tex or -export=path/outfile.svg
// infile.prb should have outfile.tex while infile.asc should have outfile.svg
// (output file extension needs to be appropriate for input file extension)
// in both cases, error log is at beginning of output file as commented lines
// also software version is output at beginning of output file

func main() {
	var inFileStr, logOut, header, outStr string
	var sigDigits, randomStr, errorHeader, errorHeader2 string
	var inFile, outFile fileInfo
	var version string

	rand.Seed(time.Now().UnixNano()) // needed so a new seed occurs every time the program is run
	//currentTime := time.Now()
	//	todayDate = currentTime.Format("2006-01-02")
	version = "0.8.5" + " (" + "2021-06-15" + ")"

	inFile, outFile, randomStr, sigDigits, logOut = commandFlags(version) // outFile depends on inFile file extension
	fileWriteString("", outFile.full)
	if logOut != "" {
		errorHeader = logOutError(logOut, -1, "ERROR") // first time assigning errorHeader so no need to concatenate
		fileAppendString(errorHeader, outFile.full)
		os.Exit(1)
	}
	header = "Created with problem2tex: version = " + version

	errorHeader = logOutError(header, -1, "") // first time assigning errorHeader so no need to concatenate
	inFileStr, logOut = fileReadString(inFile.full)
	if logOut != "" {
		errorHeader = errorHeader + logOutError(logOut, -1, "ERROR")
	}
	outStr, errorHeader2 = makeTex(inFileStr, sigDigits, randomStr, inFile, outFile)
	outStr = errorHeader + errorHeader2 + outStr
	fileAppendString(outStr, outFile.full)

}

// *******************************************************************************************
// *******************************************************************************************
