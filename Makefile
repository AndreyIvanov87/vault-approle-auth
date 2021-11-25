SERVICE_NAME := vaultauth
CURRENT_DIR = $(shell pwd)

GOOS?=darwin
GOARCH?=amd64


build:
	cd ${CURRENT_DIR}/app && GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go build -mod=readonly\
		-o ${CURRENT_DIR}/.bin/${SERVICE_NAME} ./cmd/${SERVICE_NAME}/*.go

build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make build

run:
	cd ${CURRENT_DIR}/app && GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=0 go run -mod=readonly\
        ./cmd/${SERVICE_NAME}/*.go

run-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 make run