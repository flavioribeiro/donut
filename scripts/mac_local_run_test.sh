if ! brew list srt &>/dev/null; then
    echo "ERROR you must install srt"
    echo "brew install srt"
    exit 1
fi

if ! brew list ffmpeg &>/dev/null; then
    echo "ERROR you must install ffmpeg"
    echo "brew install ffmpeg"
    exit 1
fi

export CGO_LDFLAGS="-L$(brew --prefix srt)/lib -lsrt" 
export CGO_CFLAGS="-I$(brew --prefix srt)/include/"

# testing with logging 
# ref https://github.com/golang/go/issues/46959#issuecomment-1407594935
# go test -v -p 1 ./...
go test -v ./...