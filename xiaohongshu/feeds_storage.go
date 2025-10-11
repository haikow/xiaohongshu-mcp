package xiaohongshu

import (
	"sync"
)

// FeedStorage 用于存储 Feed 列表数据，确保并发安全
type FeedStorage struct {
	feeds []FeedWithCommentStatus
	mu    sync.RWMutex
}

// NewFeedStorage 创建一个新的 FeedStorage 实例
func NewFeedStorage() *FeedStorage {
	return &FeedStorage{
		feeds: make([]FeedWithCommentStatus, 0),
	}
}

// SetFeeds 设置 Feed 列表，覆盖旧数据
func (fs *FeedStorage) SetFeeds(feeds []Feed) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	
	// 转换为带状态的 Feed 列表
	fs.feeds = make([]FeedWithCommentStatus, len(feeds))
	for i, feed := range feeds {
		fs.feeds[i] = FeedWithCommentStatus{
			Feed:   feed,
			Status: CommentStatusPending, // 默认为待评论状态
		}
	}
}

// GetFeeds 获取当前存储的 Feed 列表
func (fs *FeedStorage) GetFeeds() []FeedWithCommentStatus {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	return fs.feeds
}

// GetFeedByID 根据 ID 获取特定 Feed
func (fs *FeedStorage) GetFeedByID(id string) *FeedWithCommentStatus {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	for _, feed := range fs.feeds {
		if feed.Feed.ID == id {
			return &feed
		}
	}
	return nil
}

// MarkFeedAsCommented 标记 Feed 为已评论
func (fs *FeedStorage) MarkFeedAsCommented(id string, success bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	for i, feed := range fs.feeds {
		if feed.Feed.ID == id {
			if success {
				fs.feeds[i].Status = CommentStatusSuccess
			} else {
				fs.feeds[i].Status = CommentStatusFailed
			}
			break
		}
	}
}

// GetPendingFeeds 获取所有待评论的 Feed
func (fs *FeedStorage) GetPendingFeeds() []FeedWithCommentStatus {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	var pendingFeeds []FeedWithCommentStatus
	for _, feed := range fs.feeds {
		if feed.Status == CommentStatusPending {
			pendingFeeds = append(pendingFeeds, feed)
		}
	}
	return pendingFeeds
}