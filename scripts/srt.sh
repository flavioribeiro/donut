# ref https://github.com/Haivision/srt/blob/master/docs/apps/srt-live-transmit.md
srt-live-transmit \
    udp://${SRT_UDP_TS_INPUT_HOST}:${SRT_UDP_TS_INPUT_PORT} \
    srt://:${SRT_LISTENING_PORT}?congestion=live -v