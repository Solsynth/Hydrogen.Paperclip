# Building Backend
FROM golang:alpine as paperclip-server

WORKDIR /source
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs -o /dist ./pkg/main.go

# Runtime
FROM golang:alpine

COPY --from=paperclip-server /dist /paperclip/server

RUN apk add --no-cache ffmpeg exiftool

EXPOSE 8445

CMD ["/paperclip/server"]
