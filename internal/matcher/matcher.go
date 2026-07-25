package matcher

import (
	"strings"
	"unicode"
)

// Matcherは「Minecraftを表す語」と「問い合わせを表す語」を別々に管理する。
// 2種類の語を両方要求することで、「今日マイクラやろう」や単なる「IP」の誤反応を減らす。
type Matcher struct {
	subjectTerms []string
	queryTerms   []string
}

// NewはMVPで扱う言い回しを持つMatcherを返す。
// 現段階では編集頻度が低いため、設定ファイルや形態素解析を導入せずコード内の小さな辞書にする。
func New() Matcher {
	return Matcher{
		subjectTerms: []string{
			"マイクラ",
			"minecraft",
			"mc",
			"鯖",
			"サーバー",
		},
		queryTerms: []string{
			"ip",
			"ｉｐ",
			"address",
			"where",
			"online",
			"status",
			"open",
			"アドレス",
			"接続先",
			"どこ",
			"オンライン",
			"状態",
			"状況",
			"開いて",
			"あいて",
			"起動して",
			"立って",
			"たって",
			"入れる",
			"はいれる",
			"動いて",
			"うごいて",
			"できる",
		},
	}
}

// Matchesは正規化した文章に、対象語と問い合わせ語が1つずつ含まれるか判定する。
// 完全一致では自然な言い回しを逃し、単一キーワードだけでは誤反応が増えるためAND条件にする。
func (m Matcher) Matches(message string) bool {
	normalized := normalize(message)
	return containsAny(normalized, m.subjectTerms) && containsAny(normalized, m.queryTerms)
}

// normalizeは判定に影響しない空白・句読点・英字の大小差を取り除く。
// MVPでは高度な自然言語処理を使わず、予測しやすい軽量な前処理に限定する。
func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsSpace(r):
			return -1
		case strings.ContainsRune("?!？！。、,.！", r):
			return -1
		default:
			return r
		}
	}, value)
}

// containsAnyは候補のどれか1つが本文に含まれるかを調べる。
// 語数が少ないため、検索用インデックスや正規表現より単純な走査を選ぶ。
func containsAny(value string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}
