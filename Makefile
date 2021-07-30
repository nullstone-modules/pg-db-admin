NAME := pg-db-admin

.PHONY: tools build

tools:
	cd ~
	go get -u github.com/aws/aws-lambda-go/cmd/build-lambda-zip
	go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip
	cd -

build:
	mkdir -p ./aws/tf/files
	GOOS=linux GOARCH=amd64 go build -o ./aws/tf/files/pg-db-admin ./aws/

package: tools
	build-lambda-zip --output ./aws/tf/files/pg-db-admin.zip ./aws/tf/files/pg-db-admin
