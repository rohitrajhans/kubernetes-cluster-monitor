FROM golang:1.16-alpine AS builder

RUN apk --no-cache add git

# Set the Current Working Directory inside the container
WORKDIR /tmp/builder/controller

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY cmd/ cmd/
COPY pkg/ pkg/

# Build the Go app
RUN go build -o controller cmd/main.go

# Start fresh from a smaller image
FROM alpine:3.9

COPY --from=builder /tmp/builder/controller/controller /app/controller

CMD ["/app/controller"]
