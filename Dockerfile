FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY core ./core
COPY pipeline ./pipeline
COPY main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/serve .

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/serve /app/serve
COPY web/dist /app/web/dist
ENV WEB_ROOT=/app/web/dist
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/serve"]
