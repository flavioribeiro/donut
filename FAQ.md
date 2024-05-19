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

## If you're seeing the error "At least one invalid signature was encountered ... GPG error: http://security." when running the app

If you see the error "At least one invalid signature was encountered." when running `make run` Or "failed to copy files: userspace copy failed: write":

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

```
 => CANCELED [test stage-1 6/6] RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2                                     4.4s
 => ERROR [app stage-1 6/8] COPY . ./donut                                                                                                                                                               4.1s
------
 > [app stage-1 6/8] COPY . ./donut:
------
failed to solve: failed to copy files: userspace copy failed: write /var/lib/docker/overlay2/30zm6uywrtfed4z4wfzbf1ema/merged/usr/src/app/donut/tmp/n5.1.2/src/tests/reference.pnm: no space left on device
make: *** [run] Error 17
```

Please try to run:

```
# PLEASE be aware that the following command will erase all your docker images, containers, volumes, etc.
make clean-docker
```