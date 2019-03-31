FROM alpine:3.7

ENTRYPOINT ["./weather"]
CMD []

# Copy the binary
COPY cmd/weather .
