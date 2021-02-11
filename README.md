problem2tex app
======================

This app can convert a .prb file (a file containing a question) to .tex 

Why should I use this app?
-----

- Quickly generate questions with solutions that have random parameters. 


Versions
--------
- Available for MacOS, Linux and Windows
- Binaries located at [https://www.icewire.ca/downloads.html](https://www.icewire.ca/downloads.html)
- A docker container with icemaker, inkscape and latex is at [https://hub.docker.com/r/icewire314/latexinkice](https://hub.docker.com/r/icewire314/latexinkice)

Quick Setup
-----------

- MacOS/Linux

```bash
# unzip zip file
unzip problem2tex<version>.zip
# Ensure icemaker binary is executable
chmod 755 problem2tex
# Test icemaker app is working
./problem2tex --help
```

- Windows

Similar to MacOS/Linux above but with Windows commands

Website/Documentation
-------------
- [https://www.icewire.ca](https://www.icewire.ca)
- [https://www.icewire.ca/icemaker.pdf](https://www.icewire.ca/icemaker.pdf)
- [https://github.com/icewire314/latexinkice-docker](https://github.com/icewire314/latexinkice-docker) A docker container with icemaker, Inkscape and Latex

License
-------

See [LICENSE](LICENSE) file.
