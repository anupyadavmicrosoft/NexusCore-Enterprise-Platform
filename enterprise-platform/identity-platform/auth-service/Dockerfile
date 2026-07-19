FROM golang:1.22-alpine AS builder

WORKDIR /workspace

COPY shared-jwt-library/ /workspace/shared-jwt-library/
COPY shared-oauth-library/ /workspace/shared-oauth-library/
COPY shared-security-library/ /workspace/shared-security-library/
COPY shared-auth-library/ /workspace/shared-auth-library/
COPY auth-service/ /workspace/auth-service/

WORKDIR /workspace/auth-service

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /bin/auth-service ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

USER 65532:65532

COPY --from=builder /bin/auth-service /app/auth-service

EXPOSE 8081

ENTRYPOINT ["/app/auth-service"]
