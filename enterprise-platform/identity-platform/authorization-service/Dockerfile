FROM golang:1.22-alpine AS builder

WORKDIR /workspace

COPY shared-jwt-library/ /workspace/shared-jwt-library/
COPY shared-auth-library/ /workspace/shared-auth-library/
COPY authorization-service/ /workspace/authorization-service/

WORKDIR /workspace/authorization-service

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /bin/authorization-service ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

USER 65532:65532

COPY --from=builder /bin/authorization-service /app/authorization-service

EXPOSE 8082

ENTRYPOINT ["/app/authorization-service"]
