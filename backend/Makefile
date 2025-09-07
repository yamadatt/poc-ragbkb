build:
	GOOS=linux GOARCH=amd64 go build -o main ./src/

# SAM specific build target
build-RAGApiFunction:
	GOOS=linux GOARCH=amd64 go build -o $(ARTIFACTS_DIR)/bootstrap ./src/

clean:
	rm -f main

.PHONY: build build-RAGApiFunction clean