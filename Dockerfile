FROM golang:1.20 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY lib ./lib

RUN CGO_ENABLED=0 GOOS=linux go build -o /mnk-rss/mnk-rss

FROM gcr.io/distroless/base-debian11 AS build-release-stage

WORKDIR /

COPY --from=build-stage /mnk-rss/mnk-rss /mnk-rss/mnk-rss

EXPOSE 8010

USER nonroot:nonroot

ENTRYPOINT ["/mnk-rss/mnk-rss"]