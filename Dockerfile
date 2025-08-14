# ---
FROM golang:1.24 AS build

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

# Download dependencies first to leverage Docker cache when changing main.go
WORKDIR /work
COPY go.mod go.sum /work/
RUN go mod download

# Build main
COPY main.go /work
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=private \
  go build -o bin/webhook .

# ---
FROM scratch AS run

LABEL org.opencontainers.image.source=https://github.com/vitrvvivs/reduce-cpu-requests-webhook

COPY --from=build /work/bin/webhook /bin/webhook

CMD ["/bin/webhook"]
