.PHONY: install

make:
	mkdir -p ./build
	go build -o ./build/logger ./cmd/logger.go

clean:
	rm -rf ./build
