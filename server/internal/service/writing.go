package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"gorm.io/gorm"

	"zzdzz-blog/server/config"
	"zzdzz-blog/server/internal/model"
)

// 写作工作流(语料 → 成稿), 设计见 doc/v0.4-tech-design.md.
// compose 是无状态单步 LLM 调用, 循环由前端编排;
// 风格卡与版本快照是它仅有的两个持久化触点(§5 / §6).

// ComposeAction 三种单步动作.
type ComposeAction string

const (
	ComposeOutline ComposeAction = "outline"
	ComposeDraft   ComposeAction = "draft"
	ComposeRefine  ComposeAction = "refine"
)

// 输入长度上限(按 rune 计, 设计 §4.4).
const (
	MaxMaterialLen = 20000
	MaxDraftLen    = 50000
	MaxProfileLen  = 2000
	MaxNoteLen     = 500
)

// 版本保留上限(设计 §3): 每次 insert 后 prune, 每篇只留最近 50 版.
const MaxVersionsPerArticle = 50

var (
	// ErrNoStyleSamples 用户没有可学习的已发布文章, 无法自动提炼风格卡
	ErrNoStyleSamples = errors.New("没有可学习的已发布文章(需先发布或私有保存过文章)")
	// ErrDeriveTooFrequent 提炼限频: 每 user 每分钟 1 次(设计 §10)
	ErrDeriveTooFrequent = errors.New("提炼太频繁,请一分钟后再试")
	// ErrVersionNotFound 版本不存在或不属于该文章
	ErrVersionNotFound = errors.New("article version not found")
)

// temperature 按动作写死(设计 §4.3), 首版不进配置.
func (a ComposeAction) temperature() float32 {
	switch a {
	case ComposeOutline:
		return 0.4
	case ComposeRefine:
		return 0.5
	default:
		return 0.7
	}
}

// Temperature 暴露给 handler 构造 LLM 请求用.
func Temperature(a ComposeAction) float32 { return a.temperature() }

type ComposeInput struct {
	Action      ComposeAction
	Material    string
	Outline     string
	Draft       string
	Instruction string
	// StyleCard 非空时注入 system(设计 §4.3/§5.3), 由 handler 查 profile 后传入
	StyleCard string
}

// BuildComposeMessages 组装 system + user 两条消息.
// system = 角色 + 文风(风格卡或通用) + 按动作的输出契约;
// user = 有则拼的四个标签段.
func BuildComposeMessages(in ComposeInput) []openai.ChatCompletionMessage {
	var user strings.Builder
	addSection := func(tag, content string) {
		content = strings.TrimSpace(content)
		if content == "" {
			return
		}
		fmt.Fprintf(&user, "<%s>\n%s\n</%s>\n\n", tag, content, tag)
	}
	addSection("语料", in.Material)
	addSection("当前大纲", in.Outline)
	addSection("当前稿件", in.Draft)
	addSection("本次要求", in.Instruction)

	return []openai.ChatCompletionMessage{
		{Role: "system", Content: composeSystemPrompt(in.Action, in.StyleCard)},
		{Role: "user", Content: user.String()},
	}
}

func composeSystemPrompt(a ComposeAction, styleCard string) string {
	base := "你是博客写作助手,把作者提供的原始语料整理成博客文章。语料是作者自己的想法与素材,你的职责是整理与成文,不得替作者虚构新事实。"
	style := "文风:自然、具体、克制;多用短句与主动语态;像作者本人在跟读者讲话,不用翻译腔与AI套话(如「综上所述」「总而言之」「在当今时代」)。"
	if styleCard != "" {
		style = "作者的文风卡如下,写作时严格遵守:\n" + styleCard
	}
	var contract string
	switch a {
	case ComposeOutline:
		contract = "本次任务:从语料提炼文章大纲。只输出一个 markdown 无序列表,每行一个要点,最多 12 行;不输出任何解释、前言或结语。"
	case ComposeDraft:
		contract = "本次任务:按大纲(若有)把语料写成完整的博客文章正文。只输出 markdown 正文本身,直接开始,不要寒暄;禁止用代码围栏(```)包裹全文;除非语料明确要求,不要输出一级标题。"
	case ComposeRefine:
		contract = "本次任务:按<本次要求>调整<当前稿件>。返回调整后的全文(markdown),不是 diff、不是片段;要求未提及的部分保持原样;禁止用代码围栏(```)包裹全文。"
	}
	return base + "\n\n" + style + "\n\n" + contract
}

// -------------------- WritingService --------------------

type WritingService struct {
	db *gorm.DB
	ai *config.AIConfig

	deriveMu   sync.Mutex
	lastDerive map[uint64]time.Time // 提炼限频: 每 user 每分钟 1 次(§10)
}

func NewWritingService(db *gorm.DB, ai *config.AIConfig) *WritingService {
	return &WritingService{db: db, ai: ai, lastDerive: map[uint64]time.Time{}}
}

func (s *WritingService) aiReady() bool {
	return s.ai != nil && s.ai.Enabled && s.ai.APIKey != "" && s.ai.BaseURL != "" && s.ai.Model != ""
}

func (s *WritingService) llmClient() *openai.Client {
	cfg := openai.DefaultConfig(s.ai.APIKey)
	cfg.BaseURL = s.ai.BaseURL
	return openai.NewClientWithConfig(cfg)
}

// -------------------- 风格卡(设计 §5) --------------------

// StyleSampleArticles 风格抽样:该用户 public/private 文章(排除 draft,防未完成语料污染提炼)
// 按创建时间取最近 5 篇(设计 §5.1: 风格是"怎么写",不需要语义检索,时间序抽样即可).
func (s *WritingService) StyleSampleArticles(userID uint64) ([]model.Article, error) {
	var arts []model.Article
	err := s.db.Where("author_id = ? AND visibility IN ('public','private')", userID).
		Order("created_at DESC").Limit(5).Find(&arts).Error
	return arts, err
}

func (s *WritingService) GetStyleProfile(userID uint64) (*model.StyleProfile, error) {
	var p model.StyleProfile
	err := s.db.Where("user_id = ?", userID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// SaveManualStyleProfile 用户手改风格卡, source 置 manual(设计 §5.2).
func (s *WritingService) SaveManualStyleProfile(userID uint64, profile string) (*model.StyleProfile, error) {
	return s.saveStyleProfile(userID, profile, "manual")
}

func (s *WritingService) saveStyleProfile(userID uint64, profile, source string) (*model.StyleProfile, error) {
	p := model.StyleProfile{
		UserID:  userID,
		Profile: truncateRunes(strings.TrimSpace(profile), MaxProfileLen),
		Source:  source,
	}
	// Save 按主键 upsert(user_id 冲突则更新), UpdatedAt 由 gorm 维护
	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

// DeriveStyleProfile 抽样 → 单次 LLM 提炼 → upsert(source=auto).
// 返回提炼结果与本次用到的样例文章, 供前端展示"参考了哪几篇".
// 失败也占限频额度(先记时间后调用), 防止刷接口烧 token.
func (s *WritingService) DeriveStyleProfile(ctx context.Context, userID uint64) (*model.StyleProfile, []model.Article, error) {
	if !s.aiReady() {
		return nil, nil, errors.New("AI not configured (set ai.api_key / ai.base_url / ai.model in config.yaml)")
	}
	if err := s.allowDerive(userID); err != nil {
		return nil, nil, err
	}
	arts, err := s.StyleSampleArticles(userID)
	if err != nil {
		return nil, nil, err
	}
	if len(arts) == 0 {
		return nil, nil, ErrNoStyleSamples
	}
	profile, err := s.callDeriveLLM(ctx, arts)
	if err != nil {
		return nil, nil, err
	}
	p, err := s.saveStyleProfile(userID, profile, "auto")
	if err != nil {
		return nil, nil, err
	}
	return p, arts, nil
}

func (s *WritingService) allowDerive(userID uint64) error {
	s.deriveMu.Lock()
	defer s.deriveMu.Unlock()
	if t, ok := s.lastDerive[userID]; ok && time.Since(t) < time.Minute {
		return ErrDeriveTooFrequent
	}
	s.lastDerive[userID] = time.Now()
	return nil
}

func (s *WritingService) callDeriveLLM(ctx context.Context, arts []model.Article) (string, error) {
	var samples strings.Builder
	for i, a := range arts {
		fmt.Fprintf(&samples, "## 样本%d:《%s》\n%s\n\n", i+1, a.Title, truncateRunes(a.Content, 1500))
	}
	msgs := []openai.ChatCompletionMessage{
		{Role: "system", Content: "你是文风分析师。下面是作者最近的几篇文章节选,请提炼这位作者的写作风格卡,供写作助手模仿其文笔。只输出 markdown,固定包含这些小节:## 语气与视角 / ## 句式与长短 / ## 习惯用语与口头禅(尽量引用原文的具体用词) / ## 文章结构套路 / ## 标点与格式偏好 / ## 禁用与雷区。总长不超过 800 字;只描述可模仿的写法特征,不要评价文章内容的好坏。"},
		{Role: "user", Content: samples.String()},
	}
	resp, err := s.llmClient().CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       s.ai.Model,
		Messages:    msgs,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("upstream error: %w", err)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no choices returned")
	}
	// 提炼输出同样可能被围栏包裹,复用剥壳器(非流式:一次 Write + Finish)
	fs := NewFenceStripper()
	out := fs.Write(resp.Choices[0].Message.Content) + fs.Finish()
	return strings.TrimSpace(out), nil
}

// -------------------- 版本快照(设计 §6) --------------------

var validOrigins = map[string]bool{
	"manual": true, "ai_outline": true, "ai_draft": true, "ai_refine": true,
}

// CreateSnapshot 把文章当前内容存为快照; 采用 AI 结果前调用, 快照失败调用方必须阻断(§6.1).
func (s *WritingService) CreateSnapshot(articleID uint64, origin, note string, actor Actor) (*model.ArticleVersion, error) {
	var a model.Article
	if err := s.db.First(&a, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	if !canEdit(a, actor) {
		return nil, ErrArticleNotOwned
	}
	if !validOrigins[origin] {
		origin = "manual" // pre_restore 由服务端保留,不接受外部传入
	}
	v := model.ArticleVersion{
		ArticleID: a.ID,
		Title:     a.Title,
		Summary:   a.Summary,
		Content:   a.Content,
		Origin:    origin,
		Note:      truncateRunes(note, MaxNoteLen),
		CreatedBy: actor.UserID,
	}
	if err := s.db.Create(&v).Error; err != nil {
		return nil, err
	}
	s.pruneVersions(a.ID)
	return &v, nil
}

// ListVersions 版本列表(不带 content, 列表轻量; 判权与文章写接口一致).
func (s *WritingService) ListVersions(articleID uint64, actor Actor) ([]model.ArticleVersion, error) {
	if err := s.articleEditable(articleID, actor); err != nil {
		return nil, err
	}
	var vs []model.ArticleVersion
	err := s.db.Select("id", "article_id", "title", "origin", "note", "created_by", "created_at").
		Where("article_id = ?", articleID).
		Order("created_at DESC, id DESC").Limit(MaxVersionsPerArticle).
		Find(&vs).Error
	return vs, err
}

// GetVersion 单个版本(含 content, 供回填预览).
func (s *WritingService) GetVersion(articleID, versionID uint64, actor Actor) (*model.ArticleVersion, error) {
	if err := s.articleEditable(articleID, actor); err != nil {
		return nil, err
	}
	var v model.ArticleVersion
	err := s.db.Where("id = ? AND article_id = ?", versionID, articleID).First(&v).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrVersionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// RestoreVersion 回滚(§6.3): 先把当前内容存 pre_restore 快照, 再把目标版本的
// title/summary/content 写回文章. 版本链只增不改, 任何操作都可再退回.
func (s *WritingService) RestoreVersion(articleID, versionID uint64, actor Actor) (*model.Article, error) {
	v, err := s.GetVersion(articleID, versionID, actor)
	if err != nil {
		return nil, err
	}
	var a model.Article
	if err := s.db.First(&a, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrArticleNotFound
		}
		return nil, err
	}
	pre := model.ArticleVersion{
		ArticleID: a.ID,
		Title:     a.Title,
		Summary:   a.Summary,
		Content:   a.Content,
		Origin:    "pre_restore",
		Note:      fmt.Sprintf("回滚到版本 #%d 前的自动备份", versionID),
		CreatedBy: actor.UserID,
	}
	if err := s.db.Create(&pre).Error; err != nil {
		return nil, err
	}
	a.Title = v.Title
	a.Summary = v.Summary
	a.Content = v.Content
	if err := s.db.Save(&a).Error; err != nil {
		return nil, err
	}
	s.pruneVersions(a.ID)
	return &a, nil
}

func (s *WritingService) articleEditable(articleID uint64, actor Actor) error {
	var a model.Article
	if err := s.db.First(&a, articleID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrArticleNotFound
		}
		return err
	}
	if !canEdit(a, actor) {
		return ErrArticleNotOwned
	}
	return nil
}

func (s *WritingService) pruneVersions(articleID uint64) {
	s.db.Exec(`DELETE FROM article_versions WHERE article_id = ? AND id NOT IN (
		SELECT id FROM article_versions WHERE article_id = ? ORDER BY created_at DESC, id DESC LIMIT ?
	)`, articleID, articleID, MaxVersionsPerArticle)
}

// -------------------- 流式剥壳 --------------------

// fenceStripper 剥掉模型输出首尾的 ``` 围栏(模型偶尔无视输出契约, 设计 §4.3).
// 头部:缓冲到能判定是否以围栏开头(分片可能把 ``` 切开,数够 3 个才下结论),
//      是则连同语言标注行一起吞掉;
// 尾部:反引号及其后的尾随空白挂起不下发,Finish 时若挂起段恰为围栏则丢弃;
//      正文中间的代码围栏会被后续内容"顶出",原样保留.
type fenceStripper struct {
	primed  bool
	headBuf strings.Builder
	hold    []byte
}

func NewFenceStripper() *fenceStripper { return &fenceStripper{} }

// Write 返回本段 delta 中应当下发的部分.
func (f *fenceStripper) Write(s string) string {
	if !f.primed {
		f.headBuf.WriteString(s)
		head := f.headBuf.String()
		trimmed := strings.TrimLeft(head, " \t\r\n")
		if trimmed == "" {
			return "" // 到目前为止全是空白,继续等
		}
		bt := 0
		for bt < len(trimmed) && trimmed[bt] == '`' {
			bt++
		}
		if bt == len(trimmed) {
			// 到现在只有反引号:第 3 个可能还在路上,不能下结论
			if len(head) <= 4096 {
				return ""
			}
			f.primed = true
			f.headBuf.Reset()
			return f.holdBack(head) // 超长仍无结论,放弃剥壳
		}
		f.primed = true
		if bt >= 3 {
			rest := head[len(head)-len(trimmed)+bt:]
			if nl := strings.Index(rest, "\n"); nl >= 0 {
				f.headBuf.Reset()
				return f.holdBack(rest[nl+1:]) // 吞掉 ``` 语言标注行
			}
			if len(head) <= 4096 {
				f.primed = false // 语言行还没收全,继续缓冲
				return ""
			}
			f.headBuf.Reset()
			return f.holdBack(rest)
		}
		f.headBuf.Reset()
		return f.holdBack(head)
	}
	return f.holdBack(s)
}

// Finish 流结束:挂起段去空白后恰为 ``` 则判定为尾部围栏丢弃,否则补发.
func (f *fenceStripper) Finish() string {
	if !f.primed {
		// 流在头部判定前就结束了
		head := f.headBuf.String()
		f.headBuf.Reset()
		f.primed = true
		if strings.HasPrefix(strings.TrimLeft(head, " \t\r\n"), "```") {
			return "" // 只有围栏头没有正文,整段丢弃
		}
		f.hold = append(f.hold, head...)
	}
	if strings.TrimSpace(string(f.hold)) == "```" {
		f.hold = nil
		return ""
	}
	out := string(f.hold)
	f.hold = nil
	return out
}

// holdBack 逐字符下发;反引号与空白先挂起(可能是尾部围栏及其换行),
// 一旦出现其他字符说明不是结尾,把挂起段原样吐出再继续.
// 挂起中的反引号已有 3 个时,第 4 个不再挂起(如 ```` 开头的代码块).
func (f *fenceStripper) holdBack(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if holdable(c, f.hold) {
			f.hold = append(f.hold, c)
			if len(f.hold) >= 64 {
				out.Write(f.hold)
				f.hold = nil
			}
			continue
		}
		if len(f.hold) > 0 {
			out.Write(f.hold)
			f.hold = nil
		}
		out.WriteByte(c)
	}
	return out.String()
}

func holdable(c byte, hold []byte) bool {
	isWS := c == ' ' || c == '\t' || c == '\r' || c == '\n'
	if c != '`' && !isWS {
		return false
	}
	n := strings.Count(string(hold), "`")
	if c == '`' {
		return n < 3
	}
	return n <= 3 // 空白可跟在围栏后,一并挂起等 EOF 判定
}
