BINARY_NAME=vanilla-synchronizer
SRC_DIR=.

build:
	go build -o ${BINARY_NAME} vanilla_synchronizer.go

clean:
	rm -rf ${BINARY_NAME}

run: build
	./${BINARY_NAME} sync-config.yaml
