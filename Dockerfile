FROM golang:1.11.5 as builder
COPY . /onionbox
WORKDIR /onionbox
RUN go get -u -a -v -x github.com/ipsn/go-libtor
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -gcflags=-m -a -tags netgo -ldflags '-w -extldflags "-static"' -o onionbox .
RUN go test -v -race -bench -cpu=1,2,4 ./...
FROM scratch
COPY --from=builder /onionbox/onionbox .
EXPOSE 80
CMD ["./onionbox", "-debug"]