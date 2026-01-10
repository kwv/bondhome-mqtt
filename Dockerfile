# Optimized for GoReleaser and fast local builds.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
ARG TARGETPLATFORM
COPY ${TARGETPLATFORM:-.}/bondhome-mqtt /app/bondhome-mqtt

USER nonroot:nonroot
ENTRYPOINT ["/app/bondhome-mqtt"]
