# FAQ & Dev Troubleshooting

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

## If you're seeing the error "checkptr: converted pointer straddles multiple allocations" when using -race

When the app runs using `go build -race` it stops with the error "converted pointer straddles multiple allocations". I tried to upgrade the golang image but it didn't work, so I remove the `-race` from building.

```
srt-1     |  connected.
srt-1     | Accepted SRT target connection
app-1     | fatal error: checkptr: converted pointer straddles multiple allocations
app-1     |
app-1     | goroutine 68 [running]:
app-1     | runtime.throw({0xe57eb2?, 0xc00003f17c?})
app-1     | 	/usr/local/go/src/runtime/panic.go:1047 +0x5d fp=0xc0001d7700 sp=0xc0001d76d0 pc=0x44febd
app-1     | runtime.checkptrAlignment(0xc00031840d?, 0x3?, 0x7ffffa869c74?)
app-1     | 	/usr/local/go/src/runtime/checkptr.go:26 +0x6c fp=0xc0001d7720 sp=0xc0001d7700 pc=0x41eacc
app-1     | github.com/asticode/go-astisrt/pkg.(*Socket).Connect(0xc00003e150, {0xc00031840d, 0x3}, 0xc0b0?)
app-1     | 	/go/pkg/mod/github.com/asticode/go-astisrt@v0.3.0/pkg/socket.go:85 +0x245 fp=0xc0001d77a0 sp=0xc0001d7720 pc=0xc58ce5
app-1     | github.com/asticode/go-astisrt/pkg.Dial({{0xc000232ba0, 0x4, 0x4}, {0xc00031840d, 0x3}, 0xc0001c4060, 0x9c74})
app-1     | 	/go/pkg/mod/github.com/asticode/go-astisrt@v0.3.0/pkg/client.go:53 +0x445 fp=0xc0001d78d8 sp=0xc0001d77a0 pc=0xc55925
app-1     | github.com/flavioribeiro/donut/internal/controllers/streamers.(*SRTMpegTSStreamer).connect(0xc0002aae40, 0xc0000f4550, 0xc00010c050)
app-1     | 	/usr/src/app/donut/internal/controllers/streamers/srt_mpegts.go:161 +0x819 fp=0xc0001d7b00 sp=0xc0001d78d8 pc=0xc61bf9
app-1     | github.com/flavioribeiro/donut/internal/controllers/streamers.(*SRTMpegTSStreamer).Stream(0xc0002aae40, 0xc000100540)
app-1     | 	/usr/src/app/donut/internal/controllers/streamers/srt_mpegts.go:55 +0xa9 fp=0xc0001d7fa8 sp=0xc0001d7b00 pc=0xc5fa29
```

ref https://github.com/golang/go/issues/54690

## If you're seeing the error "At least one invalid signature was encountered ... GPG error: http://security." when running the app

If you see the error "At least one invalid signature was encountered." when running `make run`, please try to run: 

```
docker compose stop
docker compose down -v --rmi all --remove-orphans
docker system prune -a -f
docker volume prune -a -f
docker image prune -a  -f

# make sure to check if it was cleaned properly
docker system df
```

Then, uncomment the `Makefile#run` commented lines, and try again.

```
3.723 W: GPG error: http://deb.debian.org/debian bookworm InRelease: At least one invalid signature was encountered.
3.723 E: The repository 'http://deb.debian.org/debian bookworm InRelease' is not signed.
3.723 W: GPG error: http://deb.debian.org/debian bookworm-updates InRelease: At least one invalid signature was encountered.
3.723 E: The repository 'http://deb.debian.org/debian bookworm-updates InRelease' is not signed.
3.723 W: GPG error: http://deb.debian.org/debian-security bookworm-security InRelease: At least one invalid signature was encountered.
3.723 E: The repository 'http://deb.debian.org/debian-security bookworm-security InRelease' is not signed.
3.723 W: An error occurred during the signature verification. The repository is not updated and the previous index files will be used. GPG error: http://archive.ubuntu.com/ubuntu focal InRelease: At least one invalid signature was encountered.
3.723 W: An error occurred during the signature verification. The repository is not updated and the previous index files will be used. GPG error: http://security.ubuntu.com/ubuntu focal-security InRelease: At least one invalid signature was encountered.
```

## If you're seeing the error "failed to copy files: userspace copy failed: write" when running the app

If you see the error "failed to copy files: userspace copy failed: write" when running `make run`, please run `docker system prune` and try again.

```
 => CANCELED [test stage-1 6/6] RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2                                     4.4s
 => ERROR [app stage-1 6/8] COPY . ./donut                                                                                                                                                               4.1s
------
 > [app stage-1 6/8] COPY . ./donut:
------
failed to solve: failed to copy files: userspace copy failed: write /var/lib/docker/overlay2/30zm6uywrtfed4z4wfzbf1ema/merged/usr/src/app/donut/tmp/n5.1.2/src/tests/reference.pnm: no space left on device
make: *** [run] Error 17
```