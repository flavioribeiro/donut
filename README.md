
<img src="https://user-images.githubusercontent.com/244265/200068510-7c24d5c7-6ba0-44ee-8e60-0f157f990b90.png" width="350" />

donut is a zero setup required SRT+MPEG-TS -> WebRTC Bridge powered by [Pion](http://pion.ly/).

### Install & Run Locally

Make sure you have the `libsrt` installed in your system. If not, follow their [build instructions](https://github.com/Haivision/srt#build-instructions). 
Once you finish installing it, execute:

```
$ go install github.com/flavioribeiro/donut@latest
```
Once installed, execute `donut`. This will be in your `$GOPATH/bin`. The default will be `~/go/bin/donut`

### Run using docker-compose

Alternatively, you can use `docker-compose` to simulate an SRT live transmission and run the donut effortless.

```
$ make run
```

#### Open the Web UI
Open [http://localhost:8080](http://localhost:8080). You will see three text boxes. Fill in with the SRT listener configuration and hit connect.

![donut docker-compose setup](/.github/docker-compose-donut-setup.webp "donut docker-compose setup")

### How it works

Please check the [How it works](/HOW_IT_WORKS.md) section.

### FAQ

Please check the [FAQ](/FAQ.md) if you're facing any trouble.