CGO_ENABLED=0 GOOS=linux go build && \
docker build . -t sidecar-injection-operator:1 \
