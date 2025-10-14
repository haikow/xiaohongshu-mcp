package xiaohongshu

// CommentStatus 评论状态
type CommentStatus int

const (
	CommentStatusPending CommentStatus = iota // 待评论
	CommentStatusSuccess                      // 评论成功
	CommentStatusFailed                       // 评论失败
)

// FeedWithCommentStatus 带评论状态的 Feed
type FeedWithCommentStatus struct {
	Feed   Feed          // 原始 Feed 数据
	Status CommentStatus // 评论状态
}

// GetXsecToken 获取 Feed 的 XsecToken
func (f *FeedWithCommentStatus) GetXsecToken() string {
	return f.Feed.XsecToken
}

// GetID 获取 Feed 的 ID
func (f *FeedWithCommentStatus) GetID() string {
	return f.Feed.ID
}

// String 返回状态的字符串表示
func (s CommentStatus) String() string {
	switch s {
	case CommentStatusPending:
		return "pending"
	case CommentStatusSuccess:
		return "success"
	case CommentStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}