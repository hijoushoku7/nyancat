package publicip

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// ClientはグローバルIP取得APIとの通信だけを担当する。
// Discord処理からHTTPやJSONの詳細を分離し、API単体でテストできるようにする。
type Client struct {
	httpClient *http.Client
	apiURL     string
}

// responseはAPIレスポンスのうち、Botが実際に使うIPだけを受け取る。
// 不要な項目まで汎用mapで扱わず、型の違いをデコード時に検出する。
type response struct {
	IP string `json:"ip"`
}

// NewClientは共有HTTPクライアントを受け取る。
// 関数内でhttp.DefaultClientを固定使用せず、タイムアウト設定とテスト用通信を差し替え可能にする。
func NewClient(httpClient *http.Client, apiURL string) *Client {
	return &Client{httpClient: httpClient, apiURL: apiURL}
}

// FetchはAPIから現在のグローバルIPv4アドレスを1件取得する。
func (c *Client) Fetch(ctx context.Context) (string, error) {
	// 呼び出し元の期限やキャンセルをHTTPリクエストにも伝播させる。
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("IP取得リクエストの作成: %w", err)
	}

	result, err := c.httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("IP取得APIへの接続: %w", err)
	}
	defer result.Body.Close()

	// エラーHTMLなどをJSONとして誤解析しないよう、成功ステータスだけを受け付ける。
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("IP取得APIがHTTP %dを返しました", result.StatusCode)
	}

	var body response
	if err := json.NewDecoder(result.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("IP取得APIレスポンスの解析: %w", err)
	}

	// 外部APIの値をそのままDiscordへ公開せず、実在するIPv4形式か検証する。
	// Minecraftの接続先は今回IPv4前提なので、IPv6は曖昧に表示せず明示的に失敗させる。
	ip := net.ParseIP(body.IP)
	if ip == nil {
		return "", errors.New("IP取得APIが不正なIPアドレスを返しました")
	}
	if ip.To4() == nil {
		return "", errors.New("IP取得APIがIPv4以外のアドレスを返しました")
	}

	return ip.String(), nil
}
