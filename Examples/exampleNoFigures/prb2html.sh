#!/bin/bash
filename=$(basename -- "$1")
extension="${filename##*.}"
filename="${filename%.*}"
cp prb_svgFix.js tmp/fix_svg_mathjax.js
problem2tex -random=0 -outFlag=flagSolAns -export=tmp/$filename.org $filename.$extension
emacs tmp/$filename.org --batch -f org-html-export-to-html --kill

