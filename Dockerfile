# ──────────────────────────────
# 1) Build stage
# ──────────────────────────────
FROM golang:1.21.6-alpine AS builder
RUN apk add --no-cache git build-base
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o doctoriumd ./cmd/doctoriumd


# ──────────────────────────────
# 2) Runtime stage
# ──────────────────────────────
FROM alpine:3.18
RUN apk add --no-cache bash ca-certificates wget tar jq expect
SHELL ["/bin/bash", "-c"]

# CometBFT 설치
RUN wget https://github.com/cometbft/cometbft/releases/download/v0.37.15/cometbft_0.37.15_linux_amd64.tar.gz \
  && tar -xzf cometbft_0.37.15_linux_amd64.tar.gz \
  && mv cometbft /usr/local/bin/ \
  && rm cometbft_0.37.15_linux_amd64.tar.gz

# 바이너리 및 엔트리포인트 복사
COPY --from=builder /app/doctoriumd /usr/local/bin/doctoriumd
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

# 데이터 볼륨
VOLUME ["/root/.doctoriumd"]
WORKDIR /root

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD []
