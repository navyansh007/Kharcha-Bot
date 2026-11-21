FROM golang:1.27.4-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bot ./cmd/bot

FROM gcr.io/distroless/static:nonroot
COPY --from=build /bot /bot
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/bot"]
