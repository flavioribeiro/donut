FROM jrottenberg/ffmpeg:5.1.4-ubuntu2204  AS base
FROM golang:1.22

# TODO: copy only required files
COPY --from=base / /

# ffmpeg/libav libraries
ENV LD_LIBRARY_PATH="/usr/local/lib:/usr/lib:/usr/lib/x86_64-linux-gnu/"
ENV CGO_CFLAGS="-I/usr/local/include/"
ENV CGO_LDFLAGS="-L/usr/local/lib"

ENV WD=/usr/src/app
WORKDIR ${WD}

RUN apt-get clean && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      tclsh pkg-config cmake libssl-dev build-essential git \
    && apt-get clean

ENV GOPROXY=direct

COPY . ./donut
COPY ./go-astiav/ /Users/leandro.moreira/src/go-astiav/
WORKDIR ${WD}/donut
RUN go build .
CMD ["/usr/src/app/donut/donut", "--enable-ice-mux=true"]