# Build the moco-init binary
FROM quay.io/cybozu/golang:1.17-focal as builder
ENV GOPROXY=https://goproxy.cn
# Copy the go source
COPY ./ .

# Build
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o moco-init .

# the moco init image
FROM scratch as controller
LABEL org.opencontainers.image.source https://github.com/cybozu-go/moco

COPY --from=builder /work/moco-init ./
ENV PATH=/:"$PATH"
USER 27:27

ENTRYPOINT ["/moco-init"]

