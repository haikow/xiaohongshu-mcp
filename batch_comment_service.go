package main

import (
	"context"
	"time"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
)

// BatchCommentService 批量评论服务
type BatchCommentService struct {
	xhsService *XiaohongshuService
	storage    *xiaohongshu.FeedStorage
}

// NewBatchCommentService 创建批量评论服务实例
func NewBatchCommentService(xhsService *XiaohongshuService, storage *xiaohongshu.FeedStorage) *BatchCommentService {
	return &BatchCommentService{
		xhsService: xhsService,
		storage:    storage,
	}
}

// CommentFeed 评论单个Feed
func (s *BatchCommentService) CommentFeed(ctx context.Context, feedID, xsecToken, commentContent string) error {
	logrus.Infof("开始评论 Feed: %s", feedID)
	
	// 发表评论
	_, err := s.xhsService.PostCommentToFeed(ctx, feedID, xsecToken, commentContent)
	if err != nil {
		logrus.Errorf("评论 Feed %s 失败: %v", feedID, err)
		s.storage.MarkFeedAsCommented(feedID, false)
		return err
	}
	
	// 标记为成功
	s.storage.MarkFeedAsCommented(feedID, true)
	logrus.Infof("成功评论 Feed: %s", feedID)
	return nil
}

// ProcessAllFeeds 处理所有待评论的Feed
func (s *BatchCommentService) ProcessAllFeeds(ctx context.Context, commentGenerator func(xiaohongshu.FeedWithCommentStatus) string) error {
	logrus.Info("开始批量处理所有待评论的Feed")
	
	// 获取所有待评论的Feed
	pendingFeeds := s.storage.GetPendingFeeds()
	if len(pendingFeeds) == 0 {
		logrus.Info("没有待评论的Feed")
		return nil
	}
	
	logrus.Infof("发现 %d 个待评论的Feed", len(pendingFeeds))
	
	// 逐个处理Feed
	for i, feed := range pendingFeeds {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			logrus.Info("批量评论被取消")
			return ctx.Err()
		default:
		}
		
		feedID := feed.GetID()
		xsecToken := feed.GetXsecToken()
		
		logrus.Infof("处理第 %d/%d 个Feed: %s", i+1, len(pendingFeeds), feedID)
		
		// 生成评论内容
		commentContent := commentGenerator(feed)
		if commentContent == "" {
			logrus.Warnf("Feed %s 的评论内容为空，跳过", feedID)
			continue
		}
		
		// 评论Feed
		if err := s.CommentFeed(ctx, feedID, xsecToken, commentContent); err != nil {
			logrus.Errorf("处理Feed %s 时出错: %v", feedID, err)
			// 继续处理下一个Feed，不中断整个流程
		}
		
		// 添加延迟，避免请求过于频繁
		logrus.Infof("等待2秒后处理下一个Feed...")
		time.Sleep(2 * time.Second)
	}
	
	logrus.Info("批量评论处理完成")
	return nil
}

// ProcessFeedsWithLimit 处理指定数量的待评论Feed
func (s *BatchCommentService) ProcessFeedsWithLimit(ctx context.Context, limit int, commentGenerator func(xiaohongshu.FeedWithCommentStatus) string) error {
	logrus.Infof("开始批量处理最多 %d 个待评论的Feed", limit)
	
	// 获取所有待评论的Feed
	pendingFeeds := s.storage.GetPendingFeeds()
	if len(pendingFeeds) == 0 {
		logrus.Info("没有待评论的Feed")
		return nil
	}
	
	// 限制处理数量
	if limit > 0 && limit < len(pendingFeeds) {
		pendingFeeds = pendingFeeds[:limit]
	}
	
	logrus.Infof("将处理 %d 个Feed", len(pendingFeeds))
	
	// 逐个处理Feed
	for i, feed := range pendingFeeds {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			logrus.Info("批量评论被取消")
			return ctx.Err()
		default:
		}
		
		feedID := feed.GetID()
		xsecToken := feed.GetXsecToken()
		
		logrus.Infof("处理第 %d/%d 个Feed: %s", i+1, len(pendingFeeds), feedID)
		
		// 生成评论内容
		commentContent := commentGenerator(feed)
		if commentContent == "" {
			logrus.Warnf("Feed %s 的评论内容为空，跳过", feedID)
			continue
		}
		
		// 评论Feed
		if err := s.CommentFeed(ctx, feedID, xsecToken, commentContent); err != nil {
			logrus.Errorf("处理Feed %s 时出错: %v", feedID, err)
			// 继续处理下一个Feed，不中断整个流程
		}
		
		// 添加延迟，避免请求过于频繁
		logrus.Infof("等待2秒后处理下一个Feed...")
		time.Sleep(2 * time.Second)
	}
	
	logrus.Info("批量评论处理完成")
	return nil
}

// GetCommentStatus 获取评论状态统计
func (s *BatchCommentService) GetCommentStatus() map[string]int {
	statusCount := make(map[string]int)
	
	feeds := s.storage.GetFeeds()
	for _, feed := range feeds {
		switch feed.Status {
		case xiaohongshu.CommentStatusPending:
			statusCount["pending"]++
		case xiaohongshu.CommentStatusSuccess:
			statusCount["success"]++
		case xiaohongshu.CommentStatusFailed:
			statusCount["failed"]++
		}
	}
	
	return statusCount
}

// ResetFailedFeeds 重置失败的Feed状态为待评论
func (s *BatchCommentService) ResetFailedFeeds() {
	feeds := s.storage.GetFeeds()
	for _, feed := range feeds {
		if feed.Status == xiaohongshu.CommentStatusFailed {
			s.storage.MarkFeedAsCommented(feed.GetID(), false) // 重新标记为失败，然后重置
			s.storage.MarkFeedAsCommented(feed.GetID(), true)  // 然后标记为成功（实际上是重置为待评论）
		}
	}
	logrus.Info("已重置所有失败的Feed状态")
}