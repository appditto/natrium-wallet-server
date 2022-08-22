FROM golang:1.19.0 AS builder

# Set environment
ENV CGO_ENABLED=1

# Copy files
WORKDIR /build
ADD . .

# Download & Verify dependencies
RUN go mod download
RUN go mod verify

# Build binary
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o natrium_go

# Copy files to /dist
WORKDIR /dist
RUN cp /build/natrium_go ./natrium_go

# Link any dependent libaries
RUN ldd natrium_go | tr -s '[:blank:]' '\n' | grep '^/' | \
    xargs -I % sh -c 'mkdir -p $(dirname ./%); cp % ./%;'
RUN mkdir -p lib64 && cp /lib64/ld-linux-x86-64.so.2 lib64/

# Create data dir
RUN mkdir /data


### Copy binary to scratch image
FROM scratch

COPY --chown=0:0 --from=builder /dist /
COPY --chown=65534:0 --from=builder /data /data
USER 65534
WORKDIR /data

EXPOSE 3000

ENTRYPOINT ["/natrium_go"]