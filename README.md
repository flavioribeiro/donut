
<img src="https://user-images.githubusercontent.com/244265/200067221-6b5b4341-7007-41d0-80a5-776d235cf3ff.png" width="350" />

donut is a zero setup required SRT+MPEG-TS -> WebRTC Bridge powered by [Pion](http://pion.ly/). 

## Instructions

### Install `donut`

```
go install github.com/flavioribeiro/donut@latest
```
### Run ice-tcp
Execute `donut`. This will be in your `$GOPATH/bin`. The default will be `~/go/bin/donut`

### Open the Web UI
Open [http://localhost:8080](http://localhost:8080). You will see three text boxes. Fill in your details for your SRT listener configuration and hit connect.

