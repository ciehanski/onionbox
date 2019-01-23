FROM golang:1.11.4
ENV CGO_ENABLED 1
COPY . /onionbox
WORKDIR /onionbox
RUN go get -u -a -v -x github.com/ipsn/go-libtor
EXPOSE 80
CMD ["go", "run", "onionbox.go", "-debug"]