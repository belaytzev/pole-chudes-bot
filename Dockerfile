FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bot . && mkdir /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bot /bot
COPY --from=build --chown=65532:65532 /data /data
ENV DB_PATH=/data/bot.db
ENTRYPOINT ["/bot"]
