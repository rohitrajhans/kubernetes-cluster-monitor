build-collector: collector/collector.go
	docker build -f collector.Dockerfile . -t logger:1

build-endpoint: endpoint/endpoint.go
	docker build -f endpoint.Dockerfile . -t log-api:1

build-sender: sender/sender.go
	./build_sender.sh
