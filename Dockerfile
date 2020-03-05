# We need a go compiler that's based on an image with libsystemd-dev installed,
# segment/golang give us just that.
FROM goodeggs/platform-base:2.1.0 as base

FROM base as build

ENV GOPATH=/go

RUN install_packages build-essential golang govendor libsystemd-dev

# Copy the ecs-logs sources so they can be built within the container.
COPY . /go/src/github.com/segmentio/ecs-logs

# Build ecs-logs, then cleanup all unneeded packages.
RUN cd /go/src/github.com/segmentio/ecs-logs && \
    govendor sync && \
    go build -o /usr/local/bin/ecs-logs

FROM base

COPY --from=build /usr/local/bin/ecs-logs /usr/local/bin/

# Sets the container's entry point.
ENTRYPOINT ["ecs-logs"]
