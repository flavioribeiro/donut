# Running on MacOS

To run Donut locally using MacOS, make sure you have `ffmpeg@5` installed:

```bash
brew install ffmpeg@5
```

You can have multiple versions of ffmpeg installed in your mac. To find where the specific `ffmpeg@5`` was installed, run:

```bash
sudo find /opt/homebrew -name avcodec.h
```

Let's assume the prior command showed two entries:

```bash
sudo find /opt/homebrew -name avcodec.h
/opt/homebrew/Cellar/ffmpeg/7.0_1/include/libavcodec/avcodec.h
/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/include/libavcodec/avcodec.h
```

You must configure the CGO library path pointing it to ffmpeg 5 (`5.1.4_6`) folder not the newest (`7.0_1`).

```bash
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/"
export CGO_CFLAGS="-I/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/include/"
export PKG_CONFIG_PATH="/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/pkgconfig"
```

After you set the proper cgo paths, you can run it locally:

```bash
go run main.go -- --enable-ice-mux=true
go test -v ./...
```

# Simulating SRT and RTMP live streaming

You can use docker to simulate `SRT` and `RTMP` streaming:

```bash
# docker compose stop && docker compose down && docker compose up nginx_rtmp haivision_srt
make run-srt-rtmp-streaming-alone
```

They're both now exposed `RTMP/1935` and `SRT/40052` in your `localhost`. You can use VLC to test both streams:

* vlc rtmp://localhost/live/app
* vlc srt://localhost:40052