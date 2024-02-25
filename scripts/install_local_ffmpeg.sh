#!/bin/bash
set -e
PREFIX="/opt/ffmpeg"


# from https://github.com/asticode/go-astiav/blob/master/Makefile
version="n5.1.2"
srcPath="tmp/$(version)/src"
postCheckout=""

rm -rf $(srcPath)
mkdir -p $(srcPath)	
git clone --depth 1 --branch $(version) https://github.com/FFmpeg/FFmpeg $(srcPath)
# TODO: install all required libraries (srt, rtmp, aac, x264...) and enable them.
cd $(srcPath) && ./configure --prefix=.. $(configure) \
    --disable-htmlpages  --disable-doc --disable-txtpages --disable-podpages  --disable-manpages \
        # --enable-gpl \
        # --disable-ffmpeg --disable-ffplay --disable-ffprobe --enable-libopus  \
        # --enable-libsvtav1 --enable-libfdk-aac --enable-libopus \
        # --enable-libfreetype --enable-libsrt --enable-librtmp \
        # --enable-libvorbis --enable-libx265  --enable-libx264 --enable-libvpx
cd $(srcPath) && make
cd $(srcPath) && make install