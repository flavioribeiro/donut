
<img src="https://user-images.githubusercontent.com/244265/200068510-7c24d5c7-6ba0-44ee-8e60-0f157f990b90.png" width="350" />

donut is a zero setup required SRT+MPEG-TS -> WebRTC Bridge powered by [Pion](http://pion.ly/).

### Install & Run Locally

Make sure you have the `libsrt` installed in your system. If not, follow their [build instructions](https://github.com/Haivision/srt#build-instructions). 
Once you finish installing it, execute:

```
$ go install github.com/flavioribeiro/donut@latest
```
Once installed, execute `donut`. This will be in your `$GOPATH/bin`. The default will be `~/go/bin/donut`

#### Troubleshooting locally

##### Mac

If you're facing issues while trying to run it locally, such as:

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

You can try to use the [docker-compose](#run-using-docker), but if you want to run it locally you might must provide path to the linker.

```bash

# Find where the headers and libraries files are located at. If you can't find, install them with brew.
sudo find /opt/homebrew/ -name srt.h
sudo find /opt/ -name libsrt.a

# Add the required flags so the compiler/linker can find the needed files.
CGO_LDFLAGS="-L/opt/homebrew/Cellar/srt/1.5.3/lib -lsrt" CGO_CFLAGS="-I/opt/homebrew//Cellar/srt/1.5.3/include/" go run main.go helpers.go

```

You can run ffmpeg locally to simulate an SRT live transmission:

```bash
ffmpeg -hide_banner -loglevel verbose \
    -re -f lavfi -i "testsrc2=size=1280x720:rate=30,format=yuv420p" \
    -f lavfi -i "sine=frequency=1000:sample_rate=44100" \
    -c:v libx264 -preset veryfast -tune zerolatency -profile:v baseline \
    -b:v 1400k -bufsize 2800k -x264opts keyint=30:min-keyint=30:scenecut=-1 \
    -c:a aac -b:a 128k -f mpegts 'srt://0.0.0.0:40052?mode=listener&latency=400000'
```

### Run using Docker

Alternatively, you can build a docker image. Docker will take care of downloading the dependencies (including the libsrt) and compiling donut for you.

```
$ docker build -t donut .
$ docker run -it -p 8080:8080 donut
```

### Open the Web UI
Open [http://localhost:8080](http://localhost:8080). You will see three text boxes. Fill in your details for your SRT listener configuration and hit connect.

### Install & Run using Docker Compose

Docker-compose can simulate an SRT live transmission and run the donut in separate containers.

```
$ make run
```

#### Open the Web UI
Open [http://localhost:8080](http://localhost:8080). You will see three text boxes. Fill in with the SRT listener configuration and hit connect.

![donut docker-compose setup](/imgs/docker-compose-donut-setup.webp "donut docker-compose setup")

