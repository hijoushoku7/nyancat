# ビルド環境と実行環境を分け、最終イメージにGoコンパイラやソースを含めない。
FROM golang:1.26.5-alpine3.24 AS build

WORKDIR /src

# 依存定義を先にコピーし、ソース変更時もDockerの依存取得キャッシュを再利用する。
COPY go.mod go.sum* ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nyancat ./cmd/nyancat

# 実行時には小さなAlpineイメージと、HTTPS通信に必要な証明書だけを使う。
FROM alpine:3.24

# rootでBotを動かす必要がないため、専用の非特権ユーザーを作成する。
RUN apk add --no-cache ca-certificates \
    && addgroup -S nyancat \
    && adduser -S -G nyancat nyancat

COPY --from=build /out/nyancat /usr/local/bin/nyancat

USER nyancat

ENTRYPOINT ["/usr/local/bin/nyancat"]
