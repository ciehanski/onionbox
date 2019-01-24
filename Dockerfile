FROM golang:1.11.4 as builder
COPY . /onionbox
WORKDIR /onionbox
RUN go get -u -a -v -x github.com/ipsn/go-libtor && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -gcflags=-m -a -tags netgo -ldflags '-w -extldflags "-static"' -o onionbox . && \
    go test -v ./...
FROM scratch
COPY --from=builder /onionbox/onionbox .
EXPOSE 80
CMD ["./onionbox", "-debug", "-mem 1024"]