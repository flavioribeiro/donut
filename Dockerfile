FROM golang:1.19 AS builder

ENV SRT_VERSION="v1.5.0"

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      tclsh pkg-config cmake libssl-dev build-essential git \
    && apt-get clean

RUN \
  cd /opt && \
  git clone https://github.com/asticode/go-astisrt.git && \
  cd go-astisrt && \
  make install-srt version="${SRT_VERSION}" && \
  mv tmp/${SRT_VERSION} /opt/srt && \
  cd .. && \
  rm -rf go-astisrt

FROM golang:1.19
ENV WD=/usr/src/app
WORKDIR ${WD}

RUN mkdir srt-lib
COPY --from=builder /opt/srt /opt/srt

ENV GOPROXY=direct
ENV LD_LIBRARY_PATH="$LD_LIBRARY_PATH:/opt/srt/lib/"
ENV CGO_CFLAGS="-I/opt/srt/include/"
ENV CGO_LDFLAGS="-L/opt/srt/lib/"
ENV PKG_CONFIG_PATH="/opt/srt/lib/pkgconfig"

COPY . ./donut
WORKDIR ${WD}/donut
RUN go build .
CMD ["/usr/src/app/donut/donut", "--enable-ice-mux=true"]
