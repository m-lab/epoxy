FROM golang:1.11 as build

# Add the local files to be sure we are building the local source code instead
# of downloading from GitHub. All other package dependencies will be downloaded
# from HEAD.
ADD . /go/src/github.com/m-lab/epoxy
ENV CGO_ENABLED 0
RUN go get -t -v github.com/m-lab/epoxy/...
RUN go test -v github.com/m-lab/epoxy/...
RUN go get github.com/m-lab/epoxy/cmd/epoxy_boot_server

# Now copy the built binary into a minimal base image.
FROM alpine
COPY --from=build /go/bin/epoxy_boot_server /

# We must install the ca-certificates package so the ePoxy server can securely
# connect to the LetsEncrypt servers to register & create our certificates.
# As well, valid ca-certificates are needed for the storage proxy connections.
RUN apk update && apk add ca-certificates

WORKDIR /
ENTRYPOINT ["/epoxy_boot_server"]
