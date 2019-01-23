FROM golang:1.11.4 as builder
ENV CGO_ENABLED 1
COPY . /onionbox
WORKDIR /onionbox
RUN go get -u -a -v -x github.com/ipsn/go-libtor
RUN GOOS=linux GOARCH=amd64 go build -gcflags=-m -a -tags netgo -ldflags '-w -extldflags "-static"' -o onionbox .

FROM scratch
COPY --from=builder /onionbox/onionbox .
EXPOSE 80
CMD ["./onionbox", "-debug"]