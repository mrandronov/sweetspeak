

build: build-client build-server

build-client:
        go build -o sweetspeak-client client.go

build-server:
        go build -o sweetspeak-server server.go

clean:
	rm sweetspeak-client sweetspeak-server

