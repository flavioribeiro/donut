# INTRODUCTION

```golang
// build the a donut engine for user's input (ie: srt://server)
donutEngine := h.donut.EngineFor(reqParams)
// fetches the server-side stream info (codec, ...)
serverStreamInfo := donutEngine.ServerIngredients(reqParams)
// gets the client side media support (codec, ...)
clientStreamInfo := donutEngine.ClientIngredients(reqParams)
// creates the necessary recipe (by pass, transcoding from A to B, etc)
donutRecipe := donutEngine.RecipeFor(reqParams, serverStreamInfo, clientStreamInfo)

// serve asynchronously the server stream to the client web rtc
go donutEngine.Serve(&entities.DonutParameters{
	Recipe: *donutRecipe,
	OnVideoFrame: func(data []byte, c entities.MediaFrameContext) error {
		return webRTC.SendMediaSample(VIDEO_CHANNEL, data, c)
	},
	OnAudioFrame: func(data []byte, c entities.MediaFrameContext) error {
		return webRTC.SendMediaSample(AUDIO_CHANNEL, data, c)
	},
})

```

# DATA FLOW DIAGRAM

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

# CORE COMPONENTS

```mermaid
classDiagram
    class Signaling{
        +ServeHTTP()
    }

    class WebRTC{
        +Setup()
        +CreatePeerConnection()
        +CreateTrack()
        +CreateDataChannel()
        +SendMediaSample(track)
        +SendMetadata(track)
    }

    class DonutEngine{
        +EngineFor(params)
        +ServerIngredients()
        +ClientIngredients()
        +RecipeFor(server, client)
        +Serve(donutParams)
        +Appetizer()
    }

    class Prober {
        +StreamInfo(appetizer)
	    +Match(params)
    }

    class Streamer {
        +Stream(donutParams)
	    +Match(params)
    }

    DonutEngine *-- Signaling
    WebRTC *-- Signaling
    Prober *-- DonutEngine
    Streamer *-- DonutEngine
```