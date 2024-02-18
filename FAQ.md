# FAQ

## I can't connect two tabs or browser at the same for the SRT

It doesn't work When I try to connect in another browser or tab, or even when I try to refresh the current page. It raises an seemingly timeout error.

```
astisrt: connecting failed: astisrt: connecting failed: astisrt: Connection setup failure: connection timed out
```

Apparently both `ffmpeg` and `srt-live-transmit` won't allow multiple persistent connections.

ref1 https://github.com/Haivision/srt/blob/master/docs/apps/srt-live-transmit.md#medium-srt
ref2 https://github.com/asticode/go-astisrt/issues/6#issuecomment-1917076767

## It's not working on Firefox/Chrome/Edge.

[WebRTC establishes a baseline set of codecs which all compliant browsers are required to support. Some browsers may choose to allow other codecs as well.](https://developer.mozilla.org/en-US/docs/Web/Media/Formats/WebRTC_codecs#supported_video_codecs)

You might also want to check the general [support for codecs by containers](https://en.wikipedia.org/wiki/Comparison_of_video_container_formats).

## If you're facing issues while trying to run or compile it locally, such as:

```
mod/github.com/asticode/go-astisrt@v0.3.0/pkg/callbacks.go:4:11: fatal error: 'srt/srt.h' file not found
 #include <srt/srt.h>
          ^~~~~~~~~~~
1 error generated.
```

```
./main.go:117:2: undefined: setCors
./main.go:135:3: undefined: errorToHTTP
./main.go:147:3: undefined: errorToHTTP
./main.go:154:3: undefined: errorToHTTP
./main.go:158:3: undefined: errorToHTTP
./main.go:165:3: undefined: errorToHTTP
./main.go:174:18: undefined: assertSignalingCorrect
```

```
/opt/homebrew/Cellar/go/1.21.6/libexec/pkg/tool/darwin_arm64/link: running cc failed: exit status 1
ld: warning: ignoring duplicate libraries: '-lsrt'
ld: library 'srt' not found
clang: error: linker command failed with exit code 1 (use -v to see invocation)
```

You can try to use the [docker-compose](/README.md#run-using-docker-compose), but if you want to run it locally you must provide path to the linker.

```bash
#  For MacOS
CGO_LDFLAGS="-L$(brew --prefix srt)/lib -lsrt" CGO_CFLAGS="-I$(brew --prefix srt)/include/" go run main.go
```

## If you're seeing the error "could not determine kind of name for C.AV_CODEC"

Make sure you're using ffmpeg `"n5.1.2"` (via `make install-ffmpeg`), go-astiav@v0.12.0 only supports ffmpeg 5.0.

```
../../go/pkg/mod/github.com/asticode/go-astiav@v0.12.0/codec_context_flag.go:38:50: could not determine kind of name for C.AV_CODEC_FLAG2_DROP_FRAME_TIMECODE
../../go/pkg/mod/github.com/asticode/go-astiav@v0.12.0/codec_context_flag.go:21:51: could not determine kind of name for C.AV_CODEC_FLAG_TRUNCATED
```

## If you're seeing the error "issue /usr/bin/ld: skipping incompatible lib.so when searching for -lavdevice"

Fixing the docker platform fixed the problem. Even though the configured platform is amd64, the final objects are x64, don't know why yet.

```
# The tools to check the compiled objects format:
find / -name libsrt.so # to find the objects
objdump -a /opt/srt_lib/lib/libsrt.so
objdump -a /usr/local/lib/libavformat.so
```

Fixing the platform.

Dockerfile
```Dockerfile
FROM --platform=linux/amd64 jrottenberg/ffmpeg:5.1.2-ubuntu2004  AS base
```

docker-compose.yml
```yaml
platform: "linux/amd64"
```
