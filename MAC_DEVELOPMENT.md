# Running on MacOS

To develop using your mac, make sure you have `ffmpeg@5` installed:

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

You must configure the CGO library path pointing it to the ffmpeg 5 folder.

```bash
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/"
export CGO_CFLAGS="-I/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/include/"
export PKG_CONFIG_PATH="/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/pkgconfig"
```

Now, you can run it locally:

```bash
go run main.go -- --enable-ice-mux=true
go test -v ./...
```
