# Build container with go SDK
FROM docker.io/library/golang:alpine3.16 as build
WORKDIR /usr/src/app
ADD . .
RUN CGO_ENABLED=0  go build -ldflags '-extldflags "-static"' -o /usr/src/app/unity-broker

# Minimal resulting container
FROM scratch
COPY --from=build /usr/src/app/unity-broker /unity-broker
COPY .env /.env
EXPOSE 8397
ENTRYPOINT ["/unity-broker"]