package main

import (
	"math/rand"
	"os"
	"time"
)

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
