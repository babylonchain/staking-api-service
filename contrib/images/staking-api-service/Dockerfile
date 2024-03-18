FROM golang:1.21-alpine AS builder

ARG VERSION="HEAD"

RUN apk add --no-cache  \
    make \
    git \
    build-base \
    linux-headers \
    libc-dev \
    pkgconfig \
    alpine-sdk \
    libsodium-dev \
    libsodium-static \
    openssh

# Load private repos SSH deploy key and configure ssh
RUN mkdir -p /root/.ssh && ssh-keyscan github.com >> /root/.ssh/known_hosts
RUN git config --global url."git@github.com:".insteadOf "https://github.com/"
ENV GOPRIVATE=github.com/babylonchain/*
# Build
WORKDIR /go/src/github.com/babylonchain/staking-api-service
# Cache dependencies
COPY go.mod go.sum /go/src/github.com/babylonchain/staking-api-service/
RUN --mount=type=secret,id=sshKey,target=/root/.ssh/id_rsa  go mod download
# Copy the rest of the files
COPY ./ /go/src/github.com/babylonchain/staking-api-service/

# Cosmwasm - Download correct libwasmvm version
RUN WASMVM_VERSION=$(go list -m github.com/CosmWasm/wasmvm | cut -d ' ' -f 2) && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/libwasmvm_muslc.$(uname -m).a \
    -O /lib/libwasmvm_muslc.a && \
    # verify checksum
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.a | grep $(cat /tmp/checksums.txt | grep libwasmvm_muslc.$(uname -m) | cut -d ' ' -f 1)


RUN LDFLAGS='-extldflags "-static" -v' \
    CGO_ENABLED=1 \
    CGO_LDFLAGS="$CGO_LDFLAGS -lstdc++ -lm -lsodium" \
    LINK_STATICALLY=true \
    BUILD_TAGS="muslc" \
    make build

# Final minimal image with binary only
FROM alpine:3.16 as run

RUN addgroup --gid 1138 -S staking-service && adduser --uid 1138 -S staking-service -G staking-service
RUN apk add bash curl jq

# Label should match your github repo
LABEL org.opencontainers.image.source="https://github.com/babylonchain/staking-api-service:${VERSION}"


# Copy over binaries from the build-env
COPY --from=builder /go/src/github.com/babylonchain/staking-api-service/build/staking-api-service /bin/staking-api-service
COPY --from=builder /go/src/github.com/babylonchain/staking-api-service/contrib/images/staking-api-service/entrypoint.sh /bin/entrypoint.sh

WORKDIR /home/staking-service
RUN chown -R staking-service /home/staking-service
USER staking-service

ENTRYPOINT ["/bin/entrypoint.sh"]
CMD []
STOPSIGNAL SIGTERM
