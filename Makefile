NAME := pg-db-admin

.PHONY: tools build

tools:
	go install github.com/aws/aws-lambda-go/cmd/build-lambda-zip@latest

build:
	mkdir -p ./aws/tf/files
	GOOS=linux GOARCH=amd64 go build -o ./aws/tf/files/pg-db-admin ./aws/

package: tools
	cd ./aws/tf && build-lambda-zip --output files/pg-db-admin.zip files/pg-db-admin

acc: acc-up acc-run acc-down

acc-up:
	cd acc && docker-compose -p pg-db-admin-acc up -d db

acc-run:
	ACC=1 gotestsum ./acc/...

acc-down:
	cd acc && docker-compose -p pg-db-admin-acc down
