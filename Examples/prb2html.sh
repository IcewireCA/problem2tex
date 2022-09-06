#!/bin/bash
filename=$(basename -- "$1")
extension="${filename##*.}"
filename="${filename%.*}"
cp prb_svgFix.js tmp/prb_svgFix.js
cat prbHeader.txt $filename.$extension > temp235.prb
problem2tex -random=0 -outFlag=flagSolAns -export=tmp/$filename.org temp235.prb
emacs tmp/$filename.org --batch -f org-html-export-to-html --kill
rm temp235.prb

