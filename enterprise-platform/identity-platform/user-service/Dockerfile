FROM golang:1.22-alpine AS builder

WORKDIR /workspace

COPY shared-security-library/ /workspace/shared-security-library/
COPY shared-auth-library/ /workspace/shared-auth-library/
COPY user-service/ /workspace/user-service/

WORKDIR /workspace/user-service

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /bin/user-service ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

USER 65532:65532

COPY --from=builder /bin/user-service /app/user-service

EXPOSE 8083

ENTRYPOINT ["/app/user-service"]
