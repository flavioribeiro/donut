FROM golang:1.19

ENV WD=/usr/src/app
ENV SRT_VERSION="v1.5.3"
ENV SRT_FOLDER="/opt/srt_lib"
WORKDIR ${WD}

RUN apt-get clean && apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
      tclsh pkg-config cmake libssl-dev build-essential git \
    && apt-get clean

RUN \
  mkdir -p "${SRT_FOLDER}" && \
  git clone --depth 1 --branch "${SRT_VERSION}" https://github.com/Haivision/srt && \
  cd srt && \
  ./configure --prefix=${SRT_FOLDER} $(configure) && \
  make && \
  make install

# To find where the srt.h and libsrt.so were you can
# find / -name srt.h
# find / -name libsrt.so
# inside the container docker run -it --rm -t <TAG_YOU_BUILT> bash
ENV GOPROXY=direct
ENV LD_LIBRARY_PATH="$LD_LIBRARY_PATH:${SRT_FOLDER}/lib/"
ENV CGO_CFLAGS="-I${SRT_FOLDER}/include/"
ENV CGO_LDFLAGS="-L${SRT_FOLDER}/lib/"

COPY . ./donut
WORKDIR ${WD}/donut
RUN go build -race .
CMD ["/usr/src/app/donut/donut", "--enable-ice-mux=true"]