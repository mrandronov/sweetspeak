

build: build-client build-server

build-client:
        go build -o sweetspeak-client main.go

build-server:
        go build -o sweetspeak-server server.go


