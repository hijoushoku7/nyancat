package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"nyancat/internal/matcher"
	"nyancat/internal/mcstatus"
	"nyancat/internal/publicip"
)

const (
	// APIのURLは通常変更しないが、障害時に別サービスへ切り替えられるよう環境変数で上書き可能にする。
	defaultPublicIPAPIURL    = "https://api.ipify.org?format=json"
	defaultMCStatusAPIURL    = "https://api.mcstatus.io/v2/status/java/{address}"
	defaultHTTPTimeoutSecond = 5
)

// configは起動に必要な環境変数を、検証済みの値としてまとめる。
// 各処理が直接os.Getenvを呼ばないため、設定不足を起動時にまとめて検出できる。
type config struct {
	discordToken   string
	discordGuildID string
	publicIPAPIURL string
	mcStatusAPIURL string
	httpTimeout    time.Duration
}

// botはメッセージ処理に必要な依存関係をまとめる。
// グローバル変数にしないことで、担当範囲とデータの受け渡しを明確にしている。
type bot struct {
	logger         *slog.Logger
	guildID        string
	messageMatcher matcher.Matcher
	publicIP       *publicip.Client
	mcStatus       *mcstatus.Client
}

func main() {
	// 標準出力へ出すとDocker Composeのlogsから確認できるため、ログファイルは別途作成しない。
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 必須設定が足りない状態でDiscordへ接続せず、起動時点で明確に失敗させる。
	cfg, err := loadConfig()
	if err != nil {
		logger.Error("設定の読み込みに失敗しました", "error", err)
		os.Exit(1)
	}

	// 2つのAPIでHTTPクライアントを共有し、接続の再利用とタイムアウト設定を一元化する。
	httpClient := &http.Client{Timeout: cfg.httpTimeout}
	handler := &bot{
		logger:         logger,
		guildID:        cfg.discordGuildID,
		messageMatcher: matcher.New(),
		publicIP:       publicip.NewClient(httpClient, cfg.publicIPAPIURL),
		mcStatus:       mcstatus.NewClient(httpClient, cfg.mcStatusAPIURL),
	}

	// DiscordのGateway接続やレート制限処理を自作せず、DiscordGoに任せる。
	session, err := discordgo.New("Bot " + cfg.discordToken)
	if err != nil {
		logger.Error("Discordセッションを作成できませんでした", "error", err)
		os.Exit(1)
	}
	// 投稿本文だけを扱うBotなので、メンバー一覧など不要なIntentは要求しない。
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsMessageContent
	session.AddHandler(handler.onMessageCreate)

	if err := session.Open(); err != nil {
		logger.Error("Discordへ接続できませんでした", "error", err)
		os.Exit(1)
	}
	// SIGTERMで終了するDocker環境でも、Gateway接続を閉じてからプロセスを終える。
	defer func() {
		if err := session.Close(); err != nil {
			logger.Error("Discordセッションを正常に終了できませんでした", "error", err)
		}
	}()

	logger.Info("nyancatを起動しました")

	// 空ループで待機せず、OSまたはDockerからの終了シグナルを待つ。
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("nyancatを終了します")
}

// loadConfigはDocker Composeから渡された環境変数を読み、必須値と型を検証する。
// MVPでは設定ファイルの解析処理を増やさず、秘密情報とも相性のよい環境変数に統一する。
func loadConfig() (config, error) {
	cfg := config{
		discordToken:   strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		discordGuildID: strings.TrimSpace(os.Getenv("DISCORD_GUILD_ID")),
		publicIPAPIURL: envOrDefault("PUBLIC_IP_API_URL", defaultPublicIPAPIURL),
		mcStatusAPIURL: envOrDefault("MC_STATUS_API_URL", defaultMCStatusAPIURL),
		httpTimeout:    defaultHTTPTimeoutSecond * time.Second,
	}

	// 1項目ずつ終了せず、不足している必須項目を一度に表示する。
	var missing []string
	if cfg.discordToken == "" {
		missing = append(missing, "DISCORD_TOKEN")
	}
	if cfg.discordGuildID == "" {
		missing = append(missing, "DISCORD_GUILD_ID")
	}
	if len(missing) > 0 {
		return config{}, fmt.Errorf("必須環境変数が未設定です: %s", strings.Join(missing, ", "))
	}

	// タイムアウトは0以下だと即失敗や無制限の原因になるため、正の整数だけを許可する。
	if raw := strings.TrimSpace(os.Getenv("HTTP_TIMEOUT_SECONDS")); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return config{}, errors.New("HTTP_TIMEOUT_SECONDSには1以上の整数を指定してください")
		}
		cfg.httpTimeout = time.Duration(seconds) * time.Second
	}

	return cfg, nil
}

// envOrDefaultは任意設定が空なら安全な既定値を返す。
// 同じ分岐を設定項目ごとに重複させないための小さな補助関数である。
func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// onMessageCreateはDiscordの新着投稿を絞り込み、条件に合う場合だけAPI取得と返信を行う。
func (b *bot) onMessageCreate(session *discordgo.Session, event *discordgo.MessageCreate) {
	// 対象外サーバーとBot投稿を無視し、意図しない公開やBot同士の返信ループを防ぐ。
	if event.GuildID != b.guildID || event.Author == nil || event.Author.Bot {
		return
	}
	// 全投稿でAPIを呼ばず、Minecraftの接続先や状態を尋ねる言い回しだけを対象にする。
	if !b.messageMatcher.Matches(event.Content) {
		return
	}

	// IP取得と状態取得を合わせて10秒で打ち切り、遅いAPIがイベント処理を占有し続けないようにする。
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message := b.fetchMessage(ctx)
	// スラッシュコマンドやDMではなく、検知したチャンネルへ通常メッセージとして返す。
	if _, err := session.ChannelMessageSend(event.ChannelID, message); err != nil {
		b.logger.Error("Discordへメッセージを送信できませんでした", "error", err)
	}
}

// fetchMessageは2つのAPI結果を、Discordへそのまま送れる文章に組み立てる。
func (b *bot) fetchMessage(ctx context.Context) string {
	// 動的な自宅IPをMinecraft状態APIの問い合わせ先にも使うため、IPを先に取得する。
	ip, err := b.publicIP.Fetch(ctx)
	if err != nil {
		b.logger.Error("グローバルIPを取得できませんでした", "error", err)
		return "グローバルIPとMinecraftサーバーの状態を取得できませんでした。"
	}

	status, err := b.mcStatus.Fetch(ctx, ip)
	if err != nil {
		b.logger.Error("Minecraftサーバーの状態を取得できませんでした", "error", err)
		// 状態APIだけが失敗した場合、取得済みのIPまで捨てず利用者へ返す。
		return fmt.Sprintf("グローバルIP: %s\nMinecraftサーバー: 状態を取得できませんでした", ip)
	}

	// オフライン時は人数やバージョンが存在しないため、誤解を招く0値を表示しない。
	if !status.Online {
		return fmt.Sprintf("グローバルIP: %s\nMinecraftサーバー: オフライン", ip)
	}

	message := fmt.Sprintf(
		"グローバルIP: %s\nMinecraftサーバー: オンライン\nプレイヤー: %d/%d",
		ip,
		status.PlayersOnline,
		status.PlayersMax,
	)
	// APIがバージョンを返さないサーバーもあるため、空文字は表示しない。
	if status.Version != "" {
		message += "\nバージョン: " + status.Version
	}
	return message
}
