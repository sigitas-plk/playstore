BUILD_DIR=build

test: 
	go test./...

dep: 
	go mod download

vet: 
	go vet

clean: 
	go clean
	rm -rf ${BUILD_DIR}