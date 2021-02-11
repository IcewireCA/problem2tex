package main

import (
	"math"
)

// add new funtions in this file
// also need to update map tables in main.go

func funcTwo(a, b float64, f func(float64, float64) float64) float64 {
	return f(a, b)
}

func funcOne(a float64, f func(float64) float64) float64 {
	return f(a)
}

func add(a, b float64) float64 {
	return a + b
}

func sub(a, b float64) float64 {
	return a - b
}

func mult(a, b float64) float64 {
	return a * b
}

func div(a, b float64) float64 {
	return a / b
}

func pow(a, b float64) float64 {
	return math.Pow(a, b)
}

func neg(a float64) float64 {
	return -1.0 * a
}

func pos(a float64) float64 {
	return a
}

func dBV(a float64) float64 {
	return 20 * math.Log10(a)
}

func dB(a float64) float64 {
	return 10 * math.Log10(a)
}

func cosd(a float64) float64 { // returns cos(a) where a is in degrees
	return math.Cos((a / 180) * math.Pi)
}

func sind(a float64) float64 { // returns sin(a) where a is in degrees
	return math.Sin((a / 180) * math.Pi)
}

func tand(a float64) float64 { // returns tan(a) where a is in degrees
	return math.Tan((a / 180) * math.Pi)
}

func acosd(a float64) float64 { // returns arctan(a) in degrees
	return (180 / math.Pi) * math.Acos(a)
}

func asind(a float64) float64 { // returns arctan(a) in degrees
	return (180 / math.Pi) * math.Asin(a)
}

func atand(a float64) float64 { // returns arctan(a) in degrees
	return (180 / math.Pi) * math.Atan(a)
}

func parll(a, b float64) float64 { // parallel function for calculation of parallel resistors or series capacitors
	return 1 / (1/a + 1/b)
}
