package main

import "math"

type tokenAndType struct {
	token     string
	tokenType string
}

var func2 = map[string]struct { // functions with 2 inputs
	name func(float64, float64) float64
	prec int // precedence value (higher is more priority)
}{
	"+":     {add, 2},
	"-":     {sub, 2},
	"*":     {mult, 3},
	"/":     {div, 3},
	"^":     {pow, 5},
	"PARLL": {parll, 5},
	"DIV":   {div2, 3},
}

var func1 = map[string]func(float64) float64{ // functions with one input
	// if adding a new function... may need to change preamble.tex to make it look good
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
