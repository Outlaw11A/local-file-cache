GO = go

app: fetch
	CGO_ENABLED=0 go build -v -o app -a -installsuffix cgo -ldflags="-s -w" src/*.go

fetch:
	cd src/ && $(GO) get -v || true && cd ..
