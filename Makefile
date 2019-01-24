UNIX_BINARY=onionbox

run: # Rebuild the docker container
	docker build -t onionbox . && \
	docker run -p 80 onionbox
stop:
	docker stop onionbox && \
	docker container rm $(docker container ls -aq)
exec:
	docker exec -it onionbox bash
linux: # Builds a binary for linux
	GOOS=linux GOARCH=amd64 go build -gcflags=-m -a -tags netgo -ldflags '-w -extldflags "-static"' -o $(UNIX_BINARY) . && \
	mv $(UNIX_BINARY) ../$(UNIX_BINARY) && \
	cd - > /dev/null
arm: # Builds a binary for ARM
	GOOS=linux GOARCH=arm64 go build -gcflags=-m -a -tags netgo -ldflags '-w -extldflags "-static"' -o $(UNIX_BINARY) . && \
	mv $(UNIX_BINARY) ../$(UNIX_BINARY) && \
	cd - > /dev/null
lint: # Will lint the project
	golint
	go vet ./...
	go fmt ./...
test: lint # Will run tests on the project as well as lint
	go test -v ./...

.PHONY: run stop exec lint test linux arm