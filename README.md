
<img src="https://user-images.githubusercontent.com/244265/200068510-7c24d5c7-6ba0-44ee-8e60-0f157f990b90.png" width="350" />

**donut** is a zero setup required [SRT](https://en.wikipedia.org/wiki/Secure_Reliable_Transport) (_MPEG-TS_) and [RTMP](https://en.wikipedia.org/wiki/Real-Time_Messaging_Protocol) to [WebRTC](https://webrtc.org/) bridge powered by [Pion](http://pion.ly/).

# HOW IT WORKS


```go
// It builds an engine based on user inputs
// {url: url, id: id, sdp: webRTCOffer}
donutEngine := donut.EngineFor(reqParams)
// It fetches the server-side (streaming server) stream info (codec, ...)
serverStreamInfo := donutEngine.ServerIngredients(reqParams)
// It gets the client side (browser) media support (codec, ...)
clientStreamInfo := donutEngine.ClientIngredients(reqParams)
// Given the client's restrictions and the server's availability, it builds the right recipe.
donutRecipe := donutEngine.RecipeFor(reqParams, serverStreamInfo, clientStreamInfo)

// It streams the media from the backend server to the client while there's data.
go donutEngine.Serve(DonutParameters{
	Recipe: donutRecipe,
	OnVideoFrame: func(data []byte, c MediaFrameContext) error {
		return SendMediaSample(VIDEO_TYPE, data, c)
	},
	OnAudioFrame: func(data []byte, c MediaFrameContext) error {
		return SendMediaSample(AUDIO_TYPE, data, c)
	},
})
```

```mermaid
sequenceDiagram
    actor User

    box Navy
        participant browser
    end

    User->>+browser: input protocol, host, port, id, and opts
    User->>+browser: click on [Connect]
    
    Note over donut,browser: WebRTC connection setup
    
    browser->>+browser: create WebRTC browserOffer
    browser->>+donut: POST /doSignaling {browserOffer}

    donut->>+browser: reply WebRTC {serverOffer}

    Note over donut,browser: WebRTC connection setup

    loop Async streaming
        donut--)streaming server: fetchMedia
        donut--)donut: ffmpeg::libav demux/transcode
        donut--)browser: sendWebRTCMedia
        browser--)browser: render audio/video frames
        User--)browser: watch media
    end
```

![donut docker-compose setup](/.github/docker-compose-donut-setup.webp "donut docker-compose setup")

ref: [how donut works](/HOW_IT_WORKS.md)

# QUICK START

Make sure you have the `ffmpeg 5.x.x`. You must configure the CGO library path pointing it to **ffmpeg 5**.

```bash
export CGO_LDFLAGS="-L/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/"
export CGO_CFLAGS="-I/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/include/"
export PKG_CONFIG_PATH="/opt/homebrew/Cellar/ffmpeg@5/5.1.4_6/lib/pkgconfig"
```

Now you can install and run it:

```bash
go install github.com/flavioribeiro/donut@latest
donut
```

Here are specific instructions [to run on MacOS](/MAC_DEVELOPMENT.md).

# RUN USING DOCKER-COMPOSE

Alternatively, you can use `docker-compose` to simulate an [SRT live transmission and run the donut effortless](/DOCKER_DEVELOPMENT.md).

```bash
make run
```

## OPEN THE WEB UI
Open [http://localhost:8080/demo](http://localhost:8080/demo). You will see two text fields. Fill them with the your streaming info and hit connect.

![donut docker-compose setup](/.github/docker-compose-donut-setup.webp "donut docker-compose setup")

### How it works

Please check the [How it works](/HOW_IT_WORKS.md) section.

### FAQ

Please check the [FAQ](/FAQ.md) if you're facing any trouble.