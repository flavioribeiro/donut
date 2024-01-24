FROM ubuntu:20.04 AS builder

ENV SRT_VERSION="v1.5.3"
ENV SRT_FOLDER="/opt/srt_lib"

RUN apt-get clean && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      tclsh pkg-config cmake libssl-dev build-essential git \
    && apt-get clean

RUN \
  mkdir -p "${SRT_FOLDER}" && \
  git clone --depth 1 --branch "${SRT_VERSION}" https://github.com/Haivision/srt && \
  cd srt && \
  ./configure --prefix=. $(configure) && \
  make && \
  make install

FROM golang:1.19
ENV WD=/usr/src/app
WORKDIR ${WD}

RUN mkdir srt-lib
COPY --from=builder /srt /opt/srt

# To find where the srt.h and libsrt.so were you can
# find / -name srt.h
# find / -name libsrt.so
# inside the container docker run -it --rm -t <TAG_YOU_BUILT> bash
ENV GOPROXY=direct
ENV LD_LIBRARY_PATH="$LD_LIBRARY_PATH:/opt/srt/lib/"
ENV CGO_CFLAGS="-I/opt/srt/include/"
ENV CGO_LDFLAGS="-L/opt/srt/lib/"

COPY . ./donut
WORKDIR ${WD}/donut
RUN go build .
CMD ["/usr/src/app/donut/donut", "--enable-ice-mux=true"]
