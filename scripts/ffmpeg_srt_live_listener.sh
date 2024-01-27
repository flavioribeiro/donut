ffmpeg -hide_banner -loglevel verbose \
    -re -f lavfi -i "testsrc2=size=1280x720:rate=30,format=yuv420p" \
    -f lavfi -i "sine=frequency=1000:sample_rate=44100" \
    -c:v libx264 -preset veryfast -tune zerolatency -profile:v baseline \
    -b:v 1000k -bufsize 2000k -x264opts keyint=30:min-keyint=30:scenecut=-1 \
    -f mpegts "srt://${SRT_LISTENING_HOST}:${SRT_LISTENING_PORT}?mode=listener&latency=${SRT_LISTENING_LATENCY_US}"

