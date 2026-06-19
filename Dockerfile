FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags netgo -ldflags '-s -w' -o app ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /src/app ./app
COPY --from=build /src/content ./content
COPY --from=build /src/static ./static
COPY --from=build /src/templates ./templates

ENV PORT=8080
EXPOSE 8080

CMD ["./app"]
