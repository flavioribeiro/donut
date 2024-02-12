#!/bin/bash
if ! brew list srt &>/dev/null; then
    echo "ERROR you must install srt"
    echo "brew install srt"
    exit 1
fi

if ! brew list ffmpeg &>/dev/null; then
    echo "ERROR you must install ffmpeg"
    echo "brew install ffmpeg"
    exit 1
fi
