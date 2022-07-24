package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func svgResize(svgIn string, trimTopStr, trimBottomStr, trimLeftStr, trimRightStr, scaleStr string) (svgOut string) {
	var widthViewPort, heightViewPort, xmin, ymin, widthViewBox, heightViewBox float64
	var sizeOfLetter float64
	sizeOfLetter = 10.0 // used so that trimTop = 1 is more than just one pixel (in this case 10)
	scale := str2float(scaleStr)
	trimTop := sizeOfLetter * str2float(trimTopStr)
	trimBottom := sizeOfLetter * str2float(trimBottomStr)
	trimLeft := sizeOfLetter * str2float(trimLeftStr)
	trimRight := sizeOfLetter * str2float(trimRightStr)
	widthViewPort, heightViewPort, xmin, ymin, widthViewBox, heightViewBox = getSvgInfo(svgIn)
	widthViewPort = widthViewPort * scale
	heightViewPort = heightViewPort * scale // these 2 statements scale size of draw image
	ymin = ymin + trimTop
	heightViewPort = heightViewPort - (trimTop+trimBottom)*(heightViewPort/heightViewBox)
	heightViewBox = heightViewBox - trimTop - trimBottom
	xmin = xmin + trimLeft
	widthViewPort = widthViewPort - (trimLeft+trimRight)*(widthViewPort/widthViewBox)
	widthViewBox = widthViewBox - trimLeft - trimRight
	svgOut = putSvgInfo(svgIn, widthViewPort, heightViewPort, xmin, ymin, widthViewBox, heightViewBox)
	return
}

func str2float(inString string) float64 {
	var result float64
	result, _ = strconv.ParseFloat(inString, 64)
	return result
}

func getSvgInfo(svgIn string) (widthViewPort, heightViewPort, xmin, ymin, widthViewBox, heightViewBox float64) {
	var result []string
	var svgInfo string
	var reSvgInfo = regexp.MustCompile(`(?msU)(?P<res1><svg.*viewBox.*>)`)
	svgInfo = reSvgInfo.FindString(svgIn)
	if svgInfo == "" {
		return
	}
	var reWidth = regexp.MustCompile(`(?mU)width\s*=\s*\"(?P<res1>.*)\"`)
	if reWidth.MatchString(svgInfo) {
		result = reWidth.FindStringSubmatch(svgInfo)
		widthViewPort, _ = strconv.ParseFloat(result[1], 64)
	}
	var reHeight = regexp.MustCompile(`(?mU)height\s*=\s*\"(?P<res1>.*)\"`)
	if reHeight.MatchString(svgInfo) {
		result = reHeight.FindStringSubmatch(svgInfo)
		heightViewPort, _ = strconv.ParseFloat(result[1], 64)
	}
	var reViewBox = regexp.MustCompile(`(?mU)viewBox\s*=\s*\"(?P<res1>.*)\s+(?P<res2>.*)\s+(?P<res3>.*)\s+(?P<res4>.*)\"`)
	if reViewBox.MatchString(svgInfo) {
		result = reViewBox.FindStringSubmatch(svgInfo)
		xmin, _ = strconv.ParseFloat(result[1], 64)
		ymin, _ = strconv.ParseFloat(result[2], 64)
		widthViewBox, _ = strconv.ParseFloat(result[3], 64)
		heightViewBox, _ = strconv.ParseFloat(result[4], 64)
	}
	return
}

func putSvgInfo(svgIn string, widthViewPort, heightViewPort, xmin, ymin, widthViewBox, heightViewBox float64) (svgOut string) {
	var viewBoxStr string
	var svgInfo string
	var reSvgInfo = regexp.MustCompile(`(?msU)(?P<res1><svg.*viewBox.*>)`)
	found := reSvgInfo.FindString(svgIn)
	if found == "" {
		svgOut = svgIn
		return
	}
	svgInfo = found
	// found a viewBox match in <svg > definition
	var reWidth = regexp.MustCompile(`(?mU)width\s*=\s*\".*\"`)
	svgInfo = reWidth.ReplaceAllString(svgInfo, "width=\""+fmt.Sprintf("%.2f", widthViewPort)+"\"")
	var reHeight = regexp.MustCompile(`(?mU)height\s*=\s*\".*\"`)
	svgInfo = reHeight.ReplaceAllString(svgInfo, "height=\""+fmt.Sprintf("%.2f", heightViewPort)+"\"")
	var reViewBox = regexp.MustCompile(`(?mU)viewBox\s*=\s*\".*\"`)
	viewBoxStr = fmt.Sprintf("%.2f", xmin) + " " + fmt.Sprintf("%.2f", ymin) + " "
	viewBoxStr = viewBoxStr + fmt.Sprintf("%.2f", widthViewBox) + " " + fmt.Sprintf("%.2f", heightViewBox)
	svgInfo = reViewBox.ReplaceAllString(svgInfo, "viewBox=\""+viewBoxStr+"\"")
	svgOut = strings.Replace(svgIn, found, svgInfo, 1)
	return
}
