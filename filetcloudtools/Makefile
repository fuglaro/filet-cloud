# filetwm - dynamic window manager (filet-lignux fork of dwm)
# See LICENSE file for copyright and license details.

VERSION = filetcloudtools-1.0

all: filetcloudtools.deb

filetcloudtools.deb: $(shell find filetcloudtools/ -type f)
	dpkg-deb --build filetcloudtools

clean:
	rm -f filetcloudtools.deb

.PHONY: all clean
