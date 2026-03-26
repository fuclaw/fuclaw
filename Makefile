.PHONY: build-agent build-host run-host run-host-docker build-all

build-agent:
	docker build -t fuclaw-agent:latest -f build/package/Dockerfile .

build-host:
	docker build -t fuclaw-host:latest -f build/package/host.Dockerfile .

run-host:
	go run cmd/fuclaw/main.go

run-host-docker:
	docker run --rm -it \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v "$(CURDIR)":"$(CURDIR)" \
		-w "$(CURDIR)" \
		--env-file .env \
		fuclaw-host:latest

build-all: build-agent build-host run-host
