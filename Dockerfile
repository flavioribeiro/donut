# Platform is being enforced here to solve issues like:
# issue /usr/bin/ld: skipping incompatible /usr/local/lib/libavdevice.so when searching for -lavdevice
#
# The tools to check the compiled objects format showed:
# objdump -a /opt/srt_lib/lib/libsrt.so (file format elf64-littleaarch64)
# objdump -a /usr/local/lib/libavformat.so (file format elf64-little)
#
# Once the platform was fixed, the problem disappeared. Even though the configured platform
# is amd64, the final objects are x64, don't know why yet.
#
# FFmpeg/libAV is fixed on version 5.1.2 because go-astiav binding supports it.
# see https://github.com/asticode/go-astiav/issues/27
FROM --platform=linux/amd64 jrottenberg/ffmpeg:5.1.2-ubuntu2004  AS base
FROM golang:1.19

# TODO: copy only required files
COPY --from=base / /

# ffmpeg/libav libraries
ENV LD_LIBRARY_PATH="/usr/local/lib:/usr/lib:/usr/lib/x86_64-linux-gnu/"
ENV CGO_CFLAGS="-I/usr/local/include/"
ENV CGO_LDFLAGS="-L/usr/local/lib"

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

ENV GOPROXY=direct
# DO NOT ALTER THE ORDER OF THE FLAGS HERE, ffmpeg installs srt as well,
# but we want to use the SRT we just built.
ENV LD_LIBRARY_PATH="${SRT_FOLDER}/lib:$LD_LIBRARY_PATH"
ENV CGO_CFLAGS="-I${SRT_FOLDER}/include/ ${CGO_CFLAGS}"
ENV CGO_LDFLAGS="-L${SRT_FOLDER}/lib ${CGO_LDFLAGS}"

COPY . ./donut
WORKDIR ${WD}/donut
RUN go build -race .
CMD ["/usr/src/app/donut/donut", "--enable-ice-mux=true"]