NAME := pg-db-admin
REPO := "public.ecr.aws/k6p1g1l8/pg-db-admin"
VERSION := $(shell git rev-parse --short HEAD)

.PHONY: build publish

build:
	docker build -t pg-db-admin:aws -f aws.Dockerfile .

publish:
	aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
	docker tag pg-db-admin:aws $(REPO):$(VERSION)
	docker push $(REPO):$(VERSION)
	docker tag pg-db-admin:aws $(REPO):latest
	docker push $(REPO):latest
