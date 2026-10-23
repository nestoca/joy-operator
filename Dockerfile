FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /bin/joy-operator ./cmd/operator

FROM alpine

RUN openssh-client ca-certificates && update-ca-certificates 2>/dev/null || true

SHELL ["/bin/sh", "-o", "pipefail", "-c"]

RUN \
  apk add perl-utils && \
  wget https://get.helm.sh/helm-v3.19.5-linux-amd64.tar.gz -q && \
  echo "a0a5e8c592ed3f376ac110715eff214730c7422f9a44d96cf98117d2b8b0e6c0  helm-v3.19.5-linux-amd64.tar.gz" | shasum -a256 --check - && \
  tar -xzf helm-v3.19.5-linux-amd64.tar.gz && \
  mv linux-amd64/helm /usr/local/bin

COPY --from=builder /bin/joy-operator /usr/local/bin/joy-operator

ENTRYPOINT ["/usr/local/bin/joy-operator"]
