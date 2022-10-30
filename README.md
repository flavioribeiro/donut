# donut
donut is a zero setup required SRT+MPEG-TS -> WebRTC Bridge

## Instructions

### Install `donut`

```
go install github.com/flavioribeiro/donut@latest
```

### Run ice-tcp
Execute `donut`. This will be in your `$GOPATH/bin`. The default will be `~/go/bin/donut`

### Open the Web UI
Open [http://localhost:8080](http://localhost:8080). You will see three text boxes. Fill in your details for your SRT listener configuration and hit connect.
