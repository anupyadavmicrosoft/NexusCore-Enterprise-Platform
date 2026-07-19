FROM golang:1.22-alpine AS builder

WORKDIR /workspace

COPY shared-security-library/ /workspace/shared-security-library/
COPY shared-auth-library/ /workspace/shared-auth-library/
COPY tenant-service/ /workspace/tenant-service/

WORKDIR /workspace/tenant-service

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /bin/tenant-service ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

USER 65532:65532

COPY --from=builder /bin/tenant-service /app/tenant-service

EXPOSE 8084

ENTRYPOINT ["/app/tenant-service"]
