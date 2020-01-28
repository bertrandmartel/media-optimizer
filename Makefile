clean:
		@rm -rf dist
		@mkdir -p dist

build: clean
		go build

run:
		go run ./main.go

install:
		go get -u ./...

test:
		go test ./... --cover