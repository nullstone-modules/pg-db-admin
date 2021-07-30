NAME := pg-db-admin

.PHONY: build publish

build:
	GOOS=linux go build -o ./aws/tf/files/pg-db-admin ./aws/

package:
	cd ./aws/tf && tar -cvzf aws-module.tgz *.tf files/* && mv aws-module.tgz ../../
