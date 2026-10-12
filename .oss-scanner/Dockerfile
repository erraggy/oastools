FROM golang:1.25-bookworm
ENV GOFLAGS=-buildvcs=false
# scanner contract: the checkout lives inside the image, at /src
COPY . /src
WORKDIR /src
# Network is available during setup only; the audit runs offline, so fetch
# modules and prebuild the code and test binaries here.
RUN go mod download \
 && go build ./... \
 && go build -o /usr/local/bin/oastools ./cmd/oastools \
 && go test -count=1 -run '^$' ./...
