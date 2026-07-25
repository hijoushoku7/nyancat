package mcstatus

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ClientはMinecraft状態APIとの通信だけを担当する。
// MinecraftプロトコルをBot内で再実装せず、MVPでは外部REST APIへ責務を委譲する。
type Client struct {
	httpClient *http.Client
	apiURL     string
}

// StatusはDiscordへの表示に必要な値だけを公開する。
// 外部API固有のJSON構造をmainパッケージへ漏らさないための内部モデルである。
type Status struct {
	Online        bool
	PlayersOnline int
	PlayersMax    int
	Version       string
}

// responseはmcstatus.ioのJSON形式に対応する通信用モデルである。
type response struct {
	Online  bool `json:"online"`
	Players struct {
		Online int `json:"online"`
		Max    int `json:"max"`
	} `json:"players"`
	Version struct {
		NameClean string `json:"name_clean"`
		NameRaw   string `json:"name_raw"`
	} `json:"version"`
}

// NewClientは共有HTTPクライアントと、{address}を含むAPI URLを受け取る。
// API URLを固定せず、将来のサービス切り替えとテストを容易にする。
func NewClient(httpClient *http.Client, apiURL string) *Client {
	return &Client{httpClient: httpClient, apiURL: apiURL}
}

// Fetchは指定アドレスのMinecraft Java Editionサーバー状態を取得する。
func (c *Client) Fetch(ctx context.Context, address string) (Status, error) {
	// アドレスをURL用にエスケープし、文字列連結による不正なURL生成を避ける。
	endpoint := strings.ReplaceAll(c.apiURL, "{address}", url.PathEscape(address))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Status{}, fmt.Errorf("状態取得リクエストの作成: %w", err)
	}

	// Query機能はプラグイン等の追加情報を取得するが、今回は不要なので無効化して応答を速くする。
	// 利用者がURL側で明示した場合は、その設定を上書きしない。
	query := request.URL.Query()
	if !query.Has("query") {
		query.Set("query", "false")
		request.URL.RawQuery = query.Encode()
	}

	result, err := c.httpClient.Do(request)
	if err != nil {
		return Status{}, fmt.Errorf("状態取得APIへの接続: %w", err)
	}
	defer result.Body.Close()

	// 非成功レスポンスをオフライン扱いにせず、API障害として呼び出し元へ返す。
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return Status{}, fmt.Errorf("状態取得APIがHTTP %dを返しました", result.StatusCode)
	}

	var body response
	if err := json.NewDecoder(result.Body).Decode(&body); err != nil {
		return Status{}, fmt.Errorf("状態取得APIレスポンスの解析: %w", err)
	}

	// 装飾を除いた名前を優先し、APIが返さない場合だけ元のバージョン表記を使う。
	version := body.Version.NameClean
	if version == "" {
		version = body.Version.NameRaw
	}

	return Status{
		Online:        body.Online,
		PlayersOnline: body.Players.Online,
		PlayersMax:    body.Players.Max,
		Version:       version,
	}, nil
}
