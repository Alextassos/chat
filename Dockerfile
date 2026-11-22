FROM golang:1.25.5-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -trimpath -ldflags="-s -w" -o /forum .

FROM debian:bookworm-slim

WORKDIR /app
COPY --from=build /forum ./forum
COPY static ./static
COPY templates ./templates

ENV PORT=10000
ENV DATABASE_PATH=/tmp/forum.db
EXPOSE 10000

CMD ["/app/forum"]
