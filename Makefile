run: # Builds and runs the project
	docker-compose up -d
stop: # Stops the project
	docker-compose down -v --remove-orphans
restart: stop run
reset: # Refreshes base onionbox docker image and restarts the project
	docker rmi -f onionbox_onionbox:latest
	docker-compose up -d
build: # Builds the onionbox artifact
	go get -u -a -v -x github.com/ipsn/go-libtor
	go mod download
	cd cmd/onionbox && CGO_ENABLED=1 go build -a -installsuffix cgo -ldflags '-s' -o onionbox .
logs: # Prints docker-compose logs
	docker-compose logs -f --tail 100 onionbox
exec: # Open a bash shell into the onionbox docker container
	docker exec -it onionbox bash
lint: # Will lint the project
	golint ./...
	go fmt ./...
test: # Will run tests on the project
	go test -v -race -bench=. -cpu=1,2,4 ./...
	go vet ./...

.PHONY: run stop restart reset build logs exec lint test
