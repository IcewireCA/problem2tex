package main

import (
	"testing"
)

// Not set up to work correctly!!  Needs to be fixed
// func TestMakeTex(t *testing.T) {
// 	var inFile, outFile fileInfo
// 	var errorHeader, texOut string
// 	var testNames = []string{"test01"}

// 	for i := range testNames {
// 		problemInput, _ := fileReadString("./bigTestInputs/" + testNames[i] + ".prb")
// 		expectedTexOut, _ := fileReadString("./bigTestInputs/" + testNames[i] + ".tex")
// 		expectedOut := strings.Split(expectedTexOut, "\n")
// 		texOut, errorHeader = makeTex(problemInput, "false", inFile, outFile)
// 		texOut = errorHeader + texOut
// 		actualOut := strings.Split(texOut, "\n")
// 		for j := range actualOut {
// 			if actualOut[j] != expectedOut[j] {
// 				jStr := strconv.Itoa(j)
// 				t.Error(testNames[i]+" line "+jStr+" Failed: {} expected {} received {} ... {}", expectedOut[j], "{}", actualOut[j], "{}")
// 			}
// 		}
// 	}
// }

func TestGetNextOption(t *testing.T) {
	var option, tail string
	type addTest struct {
		inString, optionWant, tailWant string
	}
	var addTests = []addTest{ // set of tests
		addTest{" scale=1.0 ,  units=\\muA/V, symbol=V_{h\\,o}", "scale=1.0 ", "units=\\muA/V, symbol=V_{h\\,o}"},
		addTest{"scale= 1.0,  symbol=V_{h\\,o}", "scale= 1.0", "symbol=V_{h\\,o}"},
		addTest{"symbol=V_{h\\,o}, other stuff", "symbol=V_{h\\,o}", "other stuff"},
		addTest{" symbol=V_{h\\,o}  ", "symbol=V_{h\\,o}", ""},
		addTest{"", "", ""},
	}
	for _, test := range addTests {
		option, tail = getNextOption(test.inString)
		if option != test.optionWant {
			t.Errorf("FAIL: got: %v ---> want: %v", option, test.optionWant)
		}
		if tail != test.tailWant {
			t.Errorf("FAIL: got: %v ---> want: %v", tail, test.tailWant)
		}
	}
}

func TestGetAllOptions(t *testing.T) {
	var allOptions []option
	type addTest struct {
		inString       string
		allOptionsWant []option
	}
	var addTests = []addTest{ // set of tests
		addTest{"scale=1.0, units=\\muA/V", []option{option{name: "scale", value: "1.0"}, option{name: "units", value: "\\muA/V"}}},
		addTest{"scale= 1.0,  symbol=V_{h\\,o}", []option{option{name: "scale", value: "1.0"}, option{name: "symbol", value: "V_{h,o}"}}},
		addTest{" ", []option{}},
		addTest{"fred,  h=3, harry", []option{option{name: "fred", value: "true"}, option{name: "h", value: "3"}, option{name: "harry", value: "true"}}},
	}
	for _, test := range addTests {
		allOptions = getAllOptions(test.inString)
		for i := 0; i < len(allOptions); i++ {
			if allOptions[i].name != test.allOptionsWant[i].name {
				t.Errorf("FAIL: input: %v", test.inString)
				t.Errorf("FAIL: got: %v ---> want: %v", allOptions[i].name, test.allOptionsWant[i].name)
			}
			if allOptions[i].value != test.allOptionsWant[i].value {
				t.Errorf("FAIL: got: %v ---> want: %v", allOptions[i].value, test.allOptionsWant[i].value)
			}
		}
	}
}
