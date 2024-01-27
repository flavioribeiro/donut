# FAQ

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
# Find where the headers and libraries files are located at. If you can't find, install them with brew or apt-get.
# Feel free to replace the path after the find to roo[/] or any other suspicious place.
sudo find /opt/homebrew/ -name srt.h
sudo find /opt/ -name libsrt.a # libsrt.so for linux

# Add the required flags so the compiler/linker can find the needed files.
CGO_LDFLAGS="-L/opt/homebrew/Cellar/srt/1.5.3/lib -lsrt" CGO_CFLAGS="-I/opt/homebrew//Cellar/srt/1.5.3/include/" go run main.go helpers.go
```
