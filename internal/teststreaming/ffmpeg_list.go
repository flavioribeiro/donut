package teststreaming

import "github.com/flavioribeiro/donut/internal/entities"

// For debugging:
// use <-loglevel verbose>
// remove <-nostats>

// DO NOT REMOVE THE EXTRA SPACES ON THE END OF THESE LINES
var ffmpeg_input = ` 
	-hide_banner -loglevel error -nostats  
	-re -f lavfi -i testsrc2=size=512x288:rate=30,format=yuv420p 
	-f lavfi -i sine=frequency=1000:sample_rate=44100 
`

var FFMPEG_LIVE_SRT_MPEG_TS_H264_AAC = testFFmpeg{
	arguments: ffmpeg_input + ` 
    	-c:v libx264 -preset veryfast -tune zerolatency -profile:v baseline
    	-b:v 500k -bufsize 1000k -x264opts keyint=30:min-keyint=30:scenecut=-1		
    	-c:a aac -b:a 96k -f mpegts srt://0.0.0.0:45678?mode=listener&smoother=live&transtype=live
	`,
	expectedStreams: []entities.Stream{
		entities.Stream{Codec: entities.H264, Type: entities.VideoType},
		entities.Stream{Codec: entities.AAC, Type: entities.AudioType},
	},
	output: FFmpegOutput{Host: "127.0.0.1", Port: 45678},
}

// ref https://x265.readthedocs.io/en/stable/cli.html#executable-options
var FFMPEG_LIVE_SRT_MPEG_TS_H265_AAC = testFFmpeg{
	arguments: ffmpeg_input + `
    	-c:v libx265 -preset veryfast -profile:v main
    	-b:v 500k -bufsize 1000k -x265-params keyint=30:min-keyint=30:scenecut=0		
    	-c:a aac -b:a 96k -f mpegts srt://0.0.0.0:45678?mode=listener&smoother=live&transtype=live
	`,
	expectedStreams: []entities.Stream{
		entities.Stream{Codec: entities.H265, Type: entities.VideoType},
		entities.Stream{Codec: entities.AAC, Type: entities.AudioType},
	},
	output: FFmpegOutput{Host: "127.0.0.1", Port: 45678},
}
