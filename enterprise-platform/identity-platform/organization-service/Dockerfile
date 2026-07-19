FROM golang:1.22-alpine AS builder

WORKDIR /workspace

COPY shared-auth-library/ /workspace/shared-auth-library/
COPY organization-service/ /workspace/organization-service/

WORKDIR /workspace/organization-service

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -ldflags="-s -w" -o /bin/organization-service ./cmd/server/main.go

FROM gcr.io/distroless/static-debian12:nonroot AS final

USER 65532:65532

COPY --from=builder /bin/organization-service /app/organization-service

EXPOSE 8085

ENTRYPOINT ["/app/organization-service"]
