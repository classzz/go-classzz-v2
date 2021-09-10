# Support setting various labels on the final image
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

# Build Gczz in a stock Go builder container
FROM golang:1.16-alpine as builder

RUN apk add --no-cache gcc musl-dev linux-headers git

ADD . /go-classzz-v2
RUN cd /go-classzz-v2 && go run build/ci.go install ./cmd/gczz

# Pull Gczz into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
COPY --from=builder /go-classzz-v2/build/bin/gczz /usr/local/bin/

EXPOSE 8545 8546 32668 32668/udp
ENTRYPOINT ["gczz"]

# Add some metadata labels to help programatic image consumption
ARG COMMIT=""
ARG VERSION=""
ARG BUILDNUM=""

LABEL commit="$COMMIT" version="$VERSION" buildnum="$BUILDNUM"
