#!/bin/bash
while true
do
    ffmpeg -hide_banner -loglevel debug \
        -re -f lavfi -i testsrc2=size=768x432:rate=30,format=yuv420p \
        -f lavfi -i sine=frequency=1000:sample_rate=44100 \
        -c:v libx264 -preset veryfast -tune zerolatency -profile:v baseline \
        -vf "drawtext=text='RTMP streaming':box=1:boxborderw=10:x=(w-text_w)/2:y=(h-text_h)/2:fontsize=64:fontcolor=black" \
        -b:v 1000k -bufsize 2000k -x264opts keyint=30:min-keyint=30:scenecut=-1 \
        -c:a aac -b:a 128k \
        -f flv -listen 1 -rtmp_live live "rtmp://${RTMP_HOST}:${RTMP_PORT}/live/app"
done