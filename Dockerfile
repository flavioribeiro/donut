FROM golang:1.18 AS builder

ENV WD=/usr/src/app

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      tclsh pkg-config cmake libssl-dev build-essential git clang \
    && apt-get clean

WORKDIR ${WD}
RUN git clone https://github.com/asticode/go-astisrt.git
WORKDIR ${WD}/go-astisrt
RUN make install-srt

FROM golang:1.18
ENV WD=/usr/src/app
WORKDIR ${WD}

RUN mkdir srt-lib
COPY --from=builder ${WD}/go-astisrt/tmp/ ./srt-lib

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y clang
ENV CGO_LDFLAGS="-L${WD}/srt-lib/v1.5.0/lib/"
ENV CGO_CPPFLAGS="-I${WD}/srt-lib/v1.5.0/include/"
ENV PKG_CONFIG_PATH="${WD}/srt-lib/v1.5.0/lib/pkgconfig"
ENV LD_LIBRARY_PATH="${WD}/srt-lib/v1.5.0/lib/"

COPY . ./donut
WORKDIR ${WD}/donut
RUN CC=clang go build .
ENTRYPOINT ["/usr/src/app/donut/donut"]