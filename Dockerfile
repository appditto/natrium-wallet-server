FROM --platform=$BUILDPLATFORM golang:1.19-alpine AS build

WORKDIR /src
ARG TARGETOS TARGETARCH
RUN --mount=target=. \
    --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/natrium-server

FROM alpine

RUN apk add --no-cache ca-certificates

# Copy binary
COPY --from=build /out/natrium-server /bin

EXPOSE 8080

ADD alerts.json .

CMD ["natrium-server"]