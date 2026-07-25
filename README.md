# nyancat

Discord の会話から Minecraft サーバーの接続先や状態を尋ねる言い回しを検知し、自宅回線のグローバル IPv4 アドレスと Minecraft Java Edition サーバーのオンライン状態を返す Bot です。

Go、DiscordGo、Docker Compose を使用します。Minecraft サーバー自体はこの Compose の管理対象に含みません。

## 応答例

次のようなメッセージに反応します。

```text
マイクラって今できる？
鯖開いてる？
サーバーIP教えて
Minecraft server status?
```

オンラインの場合:

```text
グローバルIP: 203.0.113.10
Minecraftサーバー: オンライン
プレイヤー: 2/20
バージョン: 1.21.8
```

オフラインの場合:

```text
グローバルIP: 203.0.113.10
Minecraftサーバー: オフライン
```

Bot は指定した Discord サーバー内の全テキストチャンネルを対象にし、検知したチャンネルへ通常メッセージを送信します。メッセージや取得結果は保存しません。

## Discord Bot の準備

1. [Discord Developer Portal](https://discord.com/developers/applications) で Application と Bot を作成する
2. Bot の Token を発行する
3. Bot 設定の **Privileged Gateway Intents** で **Message Content Intent** を有効にする
4. OAuth2 から Bot を Discord サーバーへ追加する
5. Bot に対象チャンネルの閲覧とメッセージ送信を許可する

管理者権限や過去メッセージの閲覧権限は不要です。

## 設定

`.env.example` をコピーして `.env` を作成します。

```bash
cp .env.example .env
```

最低限、次の2項目を書き換えます。

```dotenv
DISCORD_TOKEN=Discordで発行したBotトークン
DISCORD_GUILD_ID=Botを動作させるDiscordサーバーID
```

Discord サーバー ID は、Discord の開発者モードを有効にしてサーバーを右クリックするとコピーできます。

`.env` は Git の管理対象外です。Bot トークンを `.env.example` やソースコードへ記載しないでください。

## 起動

```bash
docker compose up -d --build
```

ログ確認:

```bash
docker compose logs -f bot
```

停止:

```bash
docker compose down
```

## 使用する外部 API

- グローバル IPv4 アドレス: [ipify](https://www.ipify.org/)
- Minecraft Java Edition の状態: [mcstatus.io](https://mcstatus.io/docs)

言い回しに一致するメッセージを受信するたびに、両 API へ問い合わせます。Minecraft は標準ポート `25565` を使用する前提です。返信にはポート番号を表示しません。

## 開発

Go 1.26 以上を使用します。

```bash
go test ./...
go vet ./...
go build ./cmd/nyancat
```

現在の仕様は [MVP 要件定義](docs/mvp-requirements.md) を参照してください。将来機能を含む初期案は [docs/requirements.md](docs/requirements.md) にあります。
