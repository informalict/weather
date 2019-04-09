GO = go
GO_PACKAGES = $(shell $(GO) list ./... | grep -v /vendor/)

build:
	CGO_ENABLED=0 ${GO} build -v -ldflags "-s -w" -o cmd/weather .

linter: # it may be a job in the pipeline
	gometalinter \
		--skip=vendor \
        --disable-all \
        --enable=golint \
        --enable=misspell \
        --enable=vetshadow \
        --enable=gotype \
        --enable=vet \
        --enable=goconst \
        --enable=ineffassign \
        --enable=staticcheck \
        --deadline=300s \
        ./... ;

clean:
	docker-compose down
	docker rmi weather_api

test:
	echo "mode: set" > coverage-all.out
	$(foreach pkg,$(GO_PACKAGES), \
		$(GO) test -v -race -timeout 30s -coverprofile=coverage.out $(pkg) | tee -a test-results.out || exit 1;\
		tail -n +2 coverage.out >> coverage-all.out || exit 1;)
	$(GO) tool cover -func=coverage-all.out





