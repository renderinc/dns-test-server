FROM golang:1.13 as builder
WORKDIR /opt/dns-test-server
ADD ./cmd/ ./cmd/
ADD ./pkg/ ./pkg/
ADD ./go.mod ./go.sum ./
RUN GOOS=linux GOARCH=amd64 go build -tags netgo -ldflags '-s -w' -o dns-test-server ./cmd/main.go

FROM scratch
WORKDIR /opt/dns-test-server
COPY --from=builder /opt/dns-test-server/dns-test-server ./
CMD ["./dns-test-server"]
