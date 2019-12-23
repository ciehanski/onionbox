run:
	docker-compose up -d
stop: # Stops the project
	docker-compose down -v --remove-orphans && \
	docker rmi -f onionbox_onionbox:latest
restart: stop run
logs:
	docker-compose logs -f --tail 100 onionbox
exec: # Open a bash shell into the docker container
	docker exec -it onionbox bash
lint: # Will lint the project
	golint ./...
	go fmt ./...
test: # Will run tests on the project
	go test -v -race -bench=. -cpu=1,2,4 ./... && \
	go vet ./...

.PHONY: run stop restart logs exec lint test