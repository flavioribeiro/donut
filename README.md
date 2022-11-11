
<img src="https://user-images.githubusercontent.com/244265/200068510-7c24d5c7-6ba0-44ee-8e60-0f157f990b90.png" width="350" />

donut is a zero setup required SRT+MPEG-TS -> WebRTC Bridge powered by [Pion](http://pion.ly/).

## Instructions

### Install & Run `donut`

```
$ docker build -t donut .
$ docker run -it -p 8080:8080 donut
```

Docker will take care of downloading the dependencies (including the libsrt) and compiling donut for you.

### Open the Web UI
Open [http://localhost:8080](http://localhost:8080). You will see three text boxes. Fill in your details for your SRT listener configuration and hit connect.
