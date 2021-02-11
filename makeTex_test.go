package main

import (
	"strconv"
	"strings"
	"testing"
)

// Not set up to work correctly!!  Needs to be fixed
func TestMakeTex(t *testing.T) {
	var inFile, outFile fileInfo
	var errorHeader, texOut string
	var testNames = []string{"test01"}

	for i := range testNames {
		problemInput, _ := fileReadString("./bigTestInputs/" + testNames[i] + ".prb")
		expectedTexOut, _ := fileReadString("./bigTestInputs/" + testNames[i] + ".tex")
		expectedOut := strings.Split(expectedTexOut, "\n")
		texOut, errorHeader = makeTex(problemInput, "4", "false", inFile, outFile)
		texOut = errorHeader + texOut
		actualOut := strings.Split(texOut, "\n")
		for j := range actualOut {
			if actualOut[j] != expectedOut[j] {
				jStr := strconv.Itoa(j)
				t.Error(testNames[i]+" line "+jStr+" Failed: {} expected {} received {} ... {}", expectedOut[j], "{}", actualOut[j], "{}")
			}
		}
	}
}
