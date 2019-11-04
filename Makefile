.PHONY: libdmd clean remoted thingspro-agent-docker push

ARCH               ?= amd64
GO                  = GOPATH= CGO_ENABLED=0 go
GOCGO               = GOPATH= CGO_ENABLED=1 go
CC                  = gcc
ifeq ($(ARCH),armhf)
CC                  = arm-linux-gnueabihf-gcc
GO                 := GOPATH= GOOS=linux GOARCH=arm GOARM=7 $(GO)
GOCGO              := GOPATH= CC=$(CC) GOOS=linux GOARCH=arm GOARM=7 $(GOCGO)
endif
LDFLAGS            += -s -w
GOFLAGS             = -mod=vendor -ldflags "$(LDFLAGS)"

DRONE_BUILD_NUMBER ?= unknown
BUILDNUM           ?= $(DRONE_BUILD_NUMBER)

VERSION = 0.0.1

.PHONY: client
client:
	$(GOCGO) build $(GOFLAGS) -ldflags "-X main.VERSION=$(VERSION)" -o ./build/$(ARCH)/ha-slave ./cmd/client

.PHONY: mqtt-exporter cloud-build cloud-run cloud-stop
mqtt-exporter:
	$(GOCGO) build $(GOFLAGS) -ldflags "-X main.VERSION=$(VERSION)" -o ./build/$(ARCH)/mqtt-exporter ./cmd/cloud
cloud-build:
	docker build \
		-t moxaisd/mqtt-exporter \
		--build-arg ARCH=$(ARCH) \
		--add-host repo.isd.moxa.com:10.144.29.201 \
		-f cmd/cloud/Dockerfile \
		.
cloud-run:
	docker-compose -f cmd/cloud/docker-compose.yml up -d
cloud-stop:
	docker-compose -f cmd/cloud/docker-compose.yml down

.PHONY: test/dev test
test/dev:
	docker build \
		-t moxaisd/ha-dev \
		--build-arg ARCH=$(ARCH) \
		--add-host repo.isd.moxa.com:10.144.29.201 \
		-f Dockerfile \
		.
	docker create -it --rm \
		--name ha \
		-w /data \
		-v ${PWD}:/data \
		moxaisd/ha-dev \
		bash
	docker start ha
	docker attach ha