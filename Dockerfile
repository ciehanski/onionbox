FROM golang:1.11.4
ENV CGO_ENABLED 1
COPY . /onionbox
WORKDIR /onionbox
RUN go get -u -a -v -x github.com/ipsn/go-libtor
RUN GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -o onionbox .
EXPOSE 80
CMD ["./onionbox -debug"]