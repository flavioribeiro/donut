# INTRODUCTION

```golang
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