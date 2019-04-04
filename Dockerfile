FROM golang:1.12 AS builder
ARG DIR="/usr/local/go/src/github.com/mieczyslaw1980/weather"
RUN mkdir -p ${DIR}
COPY . ${DIR}
WORKDIR ${DIR}
RUN CGO_ENABLED=0 go build -v -ldflags "-s -w" -o /weather

FROM alpine:3.7
COPY --from=builder /weather /weather
ENTRYPOINT ["/weather"]
