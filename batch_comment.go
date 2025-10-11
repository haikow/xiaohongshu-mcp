package main

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
	"time"
)

// BatchCommentService 批量评论服务
type BatchCommentService struct {
	xhsService *XiaohongshuService
}

// NewBatchCommentService 创建批量评论服务实例
func NewBatchCommentService(xhsService *XiaohongshuService) *BatchCommentService {
	return &BatchCommentService{
		xhsService: xhsService,
	}
}

// generateCommentContent 生成评论内容（示例：根据笔记标题生成评论）
func (s *BatchCommentService) generateCommentContent(feed xiaohongshu.FeedWithCommentStatus) string {
	// 这里可以调用 AI 服务或使用预设模板生成评论内容
	// 示例：简单地使用标题生成评论
	title := feed.Feed.NoteCard.DisplayTitle
	return fmt.Sprintf("很棒的分享！关于'%s'的内容让我受益匪浅。期待更多更新！", title)
}

// processFeed 处理单个 Feed 的评论流程
func (s *BatchCommentService) processFeed(ctx context.Context, feed xiaohongshu.FeedWithCommentStatus) error {
	feedID := feed.GetID()
	xsecToken := feed.GetXsecToken()

	// 1. 获取 Feed 详情
	logrus.Infof("获取 Feed 详情: %s", feedID)
	detail, err := s.xhsService.GetFeedDetail(ctx, feedID, xsecToken)
	if err != nil {
		logrus.Errorf("获取 Feed 详情失败: %v", err)
		xiaohongshu.GlobalFeedStorage.MarkFeedAsCommented(feedID, false)
		return err
	}

	// 2. 生成评论内容（这里可以加入 AI 总结逻辑）
	commentContent := s.generateCommentContent(feed)
	logrus.Infof("生成评论内容: %s", commentContent)

	// 3. 发表评论
	logrus.Infof("发表评论到 Feed: %s", feedID)
	_, err = s.xhsService.PostCommentToFeed(ctx, feedID, xsecToken, commentContent)
	if err != nil {
		logrus.Errorf("发表评论失败: %v", err)
		xiaohongshu.GlobalFeedStorage.MarkFeedAsCommented(feedID, false)
		return err
	}

	// 4. 标记为已评论
	xiaohongshu.GlobalFeedStorage.MarkFeedAsCommented(feedID, true)
	logrus.Infof("评论成功发表到 Feed: %s", feedID)
	return nil
}

// StartBatchComment 启动批量评论流程
func (s *BatchCommentService) StartBatchComment(ctx context.Context) error {
	logrus.Info("开始批量评论流程")

	// 1. 获取待评论的 Feed 列表
	pendingFeeds := xiaohongshu.GlobalFeedStorage.GetPendingFeeds()
	logrus.Infof("发现 %d 个待评论的 Feed", len(pendingFeeds))

	// 2. 逐个处理 Feed
	for _, feed := range pendingFeeds {
		// 检查上下文是否取消
		select {
		case <-ctx.Done():
			logrus.Info("批量评论被取消")
			return ctx.Err()
		default:
		}

		// 处理单个 Feed
		if err := s.processFeed(ctx, feed); err != nil {
			logrus.Errorf("处理 Feed %s 时出错: %v", feed.GetID(), err)
			// 继续处理下一个 Feed，不中断整个流程
		}

		// 添加延迟，避免请求过于频繁
		time.Sleep(2 * time.Second)
	}

	logrus.Info("批量评论流程完成")
	return nil
}