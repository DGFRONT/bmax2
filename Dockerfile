# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bmax ./cmd/bmax \
 && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/bmax /app/bmax
COPY --from=build --chown=nonroot:nonroot /out/data /data
COPY seed /app/seed
ENV ADDR=:8080 \
    DATABASE_PATH=/data/bmax.db \
    SEED_DIR=/app/seed
VOLUME /data
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/app/bmax", "healthcheck"]
ENTRYPOINT ["/app/bmax"]
CMD ["serve"]
