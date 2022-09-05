const replaceElement = (oldElem, newElem) => {
  oldElem.parentNode.replaceChild(newElem, oldElem)
}

const fixElem = elem => {
  const textElem = elem.closest('text')
  const x = textElem.x.baseVal[0].value
  const y = textElem.y.baseVal[0].value
  const fontSize = textElem.style['font-size']
  const fontSizeNum = parseFloat(textElem.style['font-size'])
  const anchor = textElem.style['text-anchor']
  
  const svgElem = elem.querySelector('svg')
  svgElem.style['font-size'] = fontSize
  svgElem.setAttribute('x', x)
  svgElem.setAttribute('y', y)

  // Adjust position based on anchor
  const widthPx = svgElem.width.baseVal.value
  const heightPx = svgElem.height.baseVal.value
  const translateY = `translateY(-${heightPx / 2}px)`
  const translateX = (
    anchor === 'middle' ? `translateX(-${widthPx / 2}px)`
      : anchor === 'end' ? `translateX(-${widthPx}px)`
      : ''
  )
  svgElem.style['transform'] = `${translateX} ${translateY}`

  replaceElement(textElem, svgElem)
}

const fixSvg = () => {
  const elemsToFix = [...document.querySelectorAll('svg .MathJax_SVG')]
  elemsToFix.forEach(fixElem)
}

const runAfterMathJax = f => {
  const intervalId = setInterval(() => {
    if (typeof MathJax !== 'undefined') {
      MathJax.Hub.Register.StartupHook("End Typeset", f)
      clearInterval(intervalId)
    }
  }, 100)
}

runAfterMathJax(fixSvg)
