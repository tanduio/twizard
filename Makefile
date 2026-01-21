build-server:
	go build -o ./bin/twizsrv ./cmd/server

build-client:
	go build -o ./bin/twizcli ./cmd/client

build:
	go build -o ./bin/twizsrv ./cmd/server
	go build -o ./bin/twizcli ./cmd/client