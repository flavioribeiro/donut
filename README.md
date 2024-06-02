
<img src="https://user-images.githubusercontent.com/244265/200068510-7c24d5c7-6ba0-44ee-8e60-0f157f990b90.png" width="350" />

donut is a zero setup required SRT+MPEG-TS and RTMP -> WebRTC Bridge powered by [Pion](http://pion.ly/).

### Install & Run Locally

Make sure you have the `ffmpeg 5.x.x` (with SRT) installed in your system. 

You can have multiple versions of ffmpeg installed in your system. To find where the specific `ffmpeg 5.x.x` was installed, run:

```bash
sudo find /opt/homebrew -name avcodec.h
```

Let's assume the prior command showed two entries:

```bash
sudo find /opt/homebrew -name avcodec.h
/opt/homebrew/Cellar/ffmpeg/7.0_1/include/libavcodec/avcodec.h
/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/include/libavcodec/avcodec.h
```

You must configure the CGO library path pointing it to **ffmpeg 5** (`5.1.4_6`) folder instead of the newest (`7.0_1`).

```bash
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/"
export CGO_CFLAGS="-I/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/include/"
export PKG_CONFIG_PATH="/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/pkgconfig"
```

Once you finish installing, and setting it up, execute:

```bash

go install github.com/flavioribeiro/donut@latest

```

Once installed, execute `donut`. This will be in your `$GOPATH/bin`. The default will be `~/go/bin/donut`

Here are specific instructions [to run on MacOS](/MAC_DEVELOPMENT.md).

### Run using docker-compose

Alternatively, you can use `docker-compose` to simulate an [SRT live transmission and run the donut effortless](/DOCKER_DEVELOPMENT.md).


#### Open the Web UI
Open [http://localhost:8080/demo](http://localhost:8080/demo). You will see two text fields. Fill them with the your streaming info and hit connect.

![donut docker-compose setup](/.github/docker-compose-donut-setup.webp "donut docker-compose setup")

### How it works

Please check the [How it works](/HOW_IT_WORKS.md) section.

### FAQ

Please check the [FAQ](/FAQ.md) if you're facing any trouble.