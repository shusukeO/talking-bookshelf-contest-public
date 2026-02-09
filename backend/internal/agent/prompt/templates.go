package prompt

// ============================================================================
// Gemini Flash 最速版プロンプト
// - 情報量を1/3〜1/5に削減
// - 例文は構文のみ
// - 「禁止」より「優先」表現
// ============================================================================

// SystemPromptFlash is the minimal system prompt for Gemini Flash
// Keep this as short as possible - every token costs latency
//
// プロンプト内容は公開リポジトリから省略しています。
// 以下の要素を含むシステムプロンプト:
// - キャラクター設定（本棚としての人格・口調）
// - 応答ルール（文数制限、言語対応、メモ準拠）
// - 書籍リンク形式 [book::タイトル::id] の指定
// - Function Calling ツールの使用指示
// - 出力フォーマット（EMOTION, SUGGESTIONS タグ）
// - 応答例
const SystemPromptFlash = `TODO: System prompt omitted from public repository.`

// Validation templates
//
// プロンプト内容は公開リポジトリから省略しています。
// 以下のテンプレートを含む:
// - ValidationPromptJa: メモ準拠検証プロンプト（メモ・回答・リンク形式を受け取り OK/NG 判定）
// - CorrectionPromptJaWithBook: 特定書籍についての再生成プロンプト（日本語）
// - CorrectionPromptJaGeneral: 一般質問への再生成プロンプト（日本語）
// - CorrectionPromptEnWithBook: 特定書籍についての再生成プロンプト（英語）
// - CorrectionPromptEnGeneral: 一般質問への再生成プロンプト（英語）
// いずれも fmt.Sprintf の %s プレースホルダーを使用
const (
	ValidationPromptJa         = `TODO: Validation prompt omitted. Args: %s %s %s`
	CorrectionPromptJaWithBook = `TODO: Correction prompt (ja, with book) omitted. Args: %s %s`
	CorrectionPromptJaGeneral  = `TODO: Correction prompt (ja, general) omitted. Args: %s`
	CorrectionPromptEnWithBook = `TODO: Correction prompt (en, with book) omitted. Args: %s %s`
	CorrectionPromptEnGeneral  = `TODO: Correction prompt (en, general) omitted. Args: %s`
)

// Fallback messages
const (
	FallbackMessageJa = "うーん、ちょっと混乱しちゃった。もう一度聞いてもらえる？"
	FallbackMessageEn = "Hmm, I got a bit confused. Could you ask me again?"
)
