#!/bin/bash
# SRT deps
export CGO_LDFLAGS="-L$(brew --prefix srt)/lib" 
export CGO_CFLAGS="-I$(brew --prefix srt)/include/"
export PKG_CONFIG_PATH="$(brew --prefix srt)/lib/pkgconfig"

# ffmpeg/libav deps
CGO_LDFLAGS="$CGO_LDFLAGS -L$(pwd)/tmp/n5.1.2/lib/"
CGO_CFLAGS="$CGO_CFLAGS -I$(pwd)/tmp/n5.1.2/include/"
PKG_CONFIG_PATH="$PKG_CONFIG_PATH:$(pwd)/tmp/n5.1.2/lib/pkgconfig"