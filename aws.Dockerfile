FROM golang:1.16-alpine AS builder
RUN apk --no-cache add ca-certificates
RUN apk add git
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* ./
RUN go mod download
COPY . ./
RUN --mount=type=cache,target=/root/.cache/go-build go build -o app ./aws/

FROM public.ecr.aws/lambda/go:1
COPY --from=builder /src/app ${LAMBDA_TASK_ROOT}
CMD [ "app" ]
