package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/xiaohongshu"
	"strings"
	"time"
)

// MCP 工具处理函数

// handleCheckLoginStatus 处理检查登录状态
func (s *AppServer) handleCheckLoginStatus(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 检查登录状态")

	status, err := s.xiaohongshuService.CheckLoginStatus(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "检查登录状态失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	resultText := fmt.Sprintf("登录状态检查成功: %+v", status)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleGetLoginQrcode 处理获取登录二维码请求。
// 返回二维码图片的 Base64 编码和超时时间，供前端展示扫码登录。
func (s *AppServer) handleGetLoginQrcode(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取登录扫码图片")

	result, err := s.xiaohongshuService.GetLoginQrcode(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "获取登录扫码图片失败: " + err.Error()}},
			IsError: true,
		}
	}

	if result.IsLoggedIn {
		return &MCPToolResult{
			Content: []MCPContent{{Type: "text", Text: "你当前已处于登录状态"}},
		}
	}

	now := time.Now()
	deadline := func() string {
		d, err := time.ParseDuration(result.Timeout)
		if err != nil {
			return now.Format("2006-01-02 15:04:05")
		}
		return now.Add(d).Format("2006-01-02 15:04:05")
	}()

	// 已登录：文本 + 图片
	contents := []MCPContent{
		{Type: "text", Text: "请用小红书 App 在 " + deadline + " 前扫码登录 👇"},
		{
			Type:     "image",
			MimeType: "image/png",
			Data:     strings.TrimPrefix(result.Img, "data:image/png;base64,"),
		},
	}
	return &MCPToolResult{Content: contents}
}

// handlePublishContent 处理发布内容
func (s *AppServer) handlePublishContent(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 发布内容")

	// 解析参数
	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	imagePathsInterface, _ := args["images"].([]interface{})
	tagsInterface, _ := args["tags"].([]interface{})

	var imagePaths []string
	for _, path := range imagePathsInterface {
		if pathStr, ok := path.(string); ok {
			imagePaths = append(imagePaths, pathStr)
		}
	}

	var tags []string
	for _, tag := range tagsInterface {
		if tagStr, ok := tag.(string); ok {
			tags = append(tags, tagStr)
		}
	}

	logrus.Infof("MCP: 发布内容 - 标题: %s, 图片数量: %d, 标签数量: %d", title, len(imagePaths), len(tags))

	// 构建发布请求
	req := &PublishRequest{
		Title:   title,
		Content: content,
		Images:  imagePaths,
		Tags:    tags,
	}

	// 执行发布
	result, err := s.xiaohongshuService.PublishContent(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发布失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	resultText := fmt.Sprintf("内容发布成功: %+v", result)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handlePublishVideo 处理发布视频内容（仅本地单个视频文件）
func (s *AppServer) handlePublishVideo(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 发布视频内容（本地）")

	title, _ := args["title"].(string)
	content, _ := args["content"].(string)
	videoPath, _ := args["video"].(string)
	tagsInterface, _ := args["tags"].([]interface{})

	var tags []string
	for _, tag := range tagsInterface {
		if tagStr, ok := tag.(string); ok {
			tags = append(tags, tagStr)
		}
	}

	if videoPath == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发布失败: 缺少本地视频文件路径",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 发布视频 - 标题: %s, 标签数量: %d", title, len(tags))

	// 构建发布请求
	req := &PublishVideoRequest{
		Title:   title,
		Content: content,
		Video:   videoPath,
		Tags:    tags,
	}

	// 执行发布
	result, err := s.xiaohongshuService.PublishVideo(ctx, req)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发布失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	resultText := fmt.Sprintf("视频发布成功: %+v", result)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleListFeeds 处理获取Feeds列表
func (s *AppServer) handleListFeeds(ctx context.Context) *MCPToolResult {
	logrus.Info("MCP: 获取Feeds列表")

	result, err := s.xiaohongshuService.ListFeeds(ctx)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feeds列表失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取Feeds列表成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleSearchFeeds 处理搜索Feeds
func (s *AppServer) handleSearchFeeds(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 搜索Feeds")

	// 解析参数
	keyword, ok := args["keyword"].(string)
	if !ok || keyword == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "搜索Feeds失败: 缺少关键词参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 搜索Feeds - 关键词: %s", keyword)

	result, err := s.xiaohongshuService.SearchFeeds(ctx, keyword)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "搜索Feeds失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("搜索Feeds成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleGetFeedDetail 处理获取Feed详情
func (s *AppServer) handleGetFeedDetail(ctx context.Context, args map[string]any) *MCPToolResult {
	logrus.Info("MCP: 获取Feed详情")

	// 解析参数
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feed详情失败: 缺少feed_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feed详情失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 获取Feed详情 - Feed ID: %s", feedID)

	result, err := s.xiaohongshuService.GetFeedDetail(ctx, feedID, xsecToken)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取Feed详情失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取Feed详情成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleUserProfile 获取用户主页
func (s *AppServer) handleUserProfile(ctx context.Context, args map[string]any) *MCPToolResult {
	logrus.Info("MCP: 获取用户主页")

	// 解析参数
	userID, ok := args["user_id"].(string)
	if !ok || userID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取用户主页失败: 缺少user_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取用户主页失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 获取用户主页 - User ID: %s", userID)

	result, err := s.xiaohongshuService.UserProfile(ctx, userID, xsecToken)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "获取用户主页失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 格式化输出，转换为JSON字符串
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取用户主页，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleLikeFeed 处理点赞/取消点赞
func (s *AppServer) handleLikeFeed(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少feed_id参数"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少xsec_token参数"}}, IsError: true}
	}
	unlike, _ := args["unlike"].(bool)
	
	var res *ActionResult
	var err error
	
	if unlike {
		res, err = s.xiaohongshuService.UnlikeFeed(ctx, feedID, xsecToken)
	} else {
		res, err = s.xiaohongshuService.LikeFeed(ctx, feedID, xsecToken)
	}
	
	if err != nil {
		action := "点赞"
		if unlike {
			action = "取消点赞"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + "失败: " + err.Error()}}, IsError: true}
	}
	
	action := "点赞"
	if unlike {
		action = "取消点赞"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s成功 - Feed ID: %s", action, res.FeedID)}}}
}

// handleFavoriteFeed 处理收藏/取消收藏
func (s *AppServer) handleFavoriteFeed(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少feed_id参数"}}, IsError: true}
	}
	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: "操作失败: 缺少xsec_token参数"}}, IsError: true}
	}
	unfavorite, _ := args["unfavorite"].(bool)
	
	var res *ActionResult
	var err error
	
	if unfavorite {
		res, err = s.xiaohongshuService.UnfavoriteFeed(ctx, feedID, xsecToken)
	} else {
		res, err = s.xiaohongshuService.FavoriteFeed(ctx, feedID, xsecToken)
	}
	
	if err != nil {
		action := "收藏"
		if unfavorite {
			action = "取消收藏"
		}
		return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: action + "失败: " + err.Error()}}, IsError: true}
	}
	
	action := "收藏"
	if unfavorite {
		action = "取消收藏"
	}
	return &MCPToolResult{Content: []MCPContent{{Type: "text", Text: fmt.Sprintf("%s成功 - Feed ID: %s", action, res.FeedID)}}}
}

// handlePostComment 处理发表评论到Feed
func (s *AppServer) handlePostComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 发表评论到Feed")

	// 解析参数
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: 缺少feed_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: 缺少content参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 发表评论 - Feed ID: %s, 内容长度: %d", feedID, len(content))

	// 发表评论
	result, err := s.xiaohongshuService.PostCommentToFeed(ctx, feedID, xsecToken, content)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "发表评论失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 返回成功结果，只包含feed_id
	resultText := fmt.Sprintf("评论发表成功 - Feed ID: %s", result.FeedID)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleReplyComment 处理回复评论
func (s *AppServer) handleReplyComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 回复评论")

	// 解析参数
	feedID, ok := args["feed_id"].(string)
	if !ok || feedID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少feed_id参数",
			}},
			IsError: true,
		}
	}

	xsecToken, ok := args["xsec_token"].(string)
	if !ok || xsecToken == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少xsec_token参数",
			}},
			IsError: true,
		}
	}

	commentID, _ := args["comment_id"].(string)
	userID, _ := args["user_id"].(string)
	if commentID == "" && userID == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少comment_id或user_id参数",
			}},
			IsError: true,
		}
	}

	content, ok := args["content"].(string)
	if !ok || content == "" {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: 缺少content参数",
			}},
			IsError: true,
		}
	}

	logrus.Infof("MCP: 回复评论 - Feed ID: %s, Comment ID: %s, User ID: %s, 内容长度: %d", feedID, commentID, userID, len(content))

	// 回复评论
	result, err := s.xiaohongshuService.ReplyCommentToFeed(ctx, feedID, xsecToken, commentID, userID, content)
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "回复评论失败: " + err.Error(),
			}},
			IsError: true,
		}
	}

	// 返回成功结果
	responseText := fmt.Sprintf("评论回复成功 - Feed ID: %s, Comment ID: %s, User ID: %s", result.FeedID, result.TargetCommentID, result.TargetUserID)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: responseText,
		}},
	}
}

// handleBatchComment 处理批量评论
func (s *AppServer) handleBatchComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 批量评论")

	// 解析参数
	limit, _ := args["limit"].(int)
	if limit <= 0 {
		limit = 0 // 0表示处理所有待评论的Feed
	}

	logrus.Infof("MCP: 批量评论 - 限制数量: %d", limit)

	// 获取待评论的Feed列表
	pendingFeeds := xiaohongshu.GlobalFeedStorage.GetPendingFeeds()
	if len(pendingFeeds) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "没有待评论的Feed",
			}},
		}
	}

	// 限制处理数量
	if limit > 0 && limit < len(pendingFeeds) {
		pendingFeeds = pendingFeeds[:limit]
	}

	// 返回Feed列表，让大模型为每个Feed生成评论内容
	feedList := make([]map[string]interface{}, 0, len(pendingFeeds))
	for _, feed := range pendingFeeds {
		feedInfo := map[string]interface{}{
			"feed_id":     feed.GetID(),
			"xsec_token":  feed.GetXsecToken(),
		}
		feedList = append(feedList, feedInfo)
	}

	// 格式化输出
	jsonData, err := json.MarshalIndent(feedList, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取待评论Feed列表失败，序列化错误: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleGetCommentStatus 处理获取评论状态
func (s *AppServer) handleGetCommentStatus(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 获取评论状态")

	// 创建批量评论服务
	batchService := NewBatchCommentService(s.xiaohongshuService, xiaohongshu.GlobalFeedStorage)

	// 获取状态统计
	status := batchService.GetCommentStatus()
	
	// 获取所有Feed详情
	feeds := xiaohongshu.GlobalFeedStorage.GetFeeds()
	feedDetails := make([]map[string]interface{}, 0, len(feeds))
	
	for _, feed := range feeds {
		detail := map[string]interface{}{
			"feed_id":     feed.GetID(),
			"title":       feed.Feed.NoteCard.DisplayTitle,
			"status":      feed.Status.String(),
			"xsec_token":  feed.GetXsecToken(),
		}
		feedDetails = append(feedDetails, detail)
	}

	result := map[string]interface{}{
		"summary": status,
		"feeds":   feedDetails,
	}

	// 格式化输出
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取评论状态成功，但序列化失败: %v", err),
			}},
			IsError: true,
		}
	}

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: string(jsonData),
		}},
	}
}

// handleResetFailedComments 处理重置失败的评论
func (s *AppServer) handleResetFailedComments(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 重置失败的评论")

	// 创建批量评论服务
	batchService := NewBatchCommentService(s.xiaohongshuService, xiaohongshu.GlobalFeedStorage)

	// 重置失败的Feed
	batchService.ResetFailedFeeds()

	// 获取重置后的状态
	status := batchService.GetCommentStatus()
	statusText := fmt.Sprintf("已重置失败的评论 - 当前状态 - 待评论: %d, 成功: %d, 失败: %d", 
		status["pending"], status["success"], status["failed"])

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: statusText,
		}},
	}
}

// handleExecuteBatchComment 处理执行批量评论
func (s *AppServer) handleExecuteBatchComment(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 执行批量评论")

	// 解析参数
	limit, _ := args["limit"].(int)
	if limit <= 0 {
		limit = 0 // 0表示处理所有待评论的Feed
	}

	logrus.Infof("MCP: 执行批量评论 - 限制数量: %d", limit)

	// 检查是否有待评论的Feed
	pendingFeeds := xiaohongshu.GlobalFeedStorage.GetPendingFeeds()
	if len(pendingFeeds) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "没有待评论的Feed",
			}},
		}
	}

	logrus.Infof("发现 %d 个待评论的Feed，但此工具需要大模型提供评论内容", len(pendingFeeds))

	// 返回Feed列表，让大模型为每个Feed生成评论内容
	feedList := make([]map[string]interface{}, 0, len(pendingFeeds))
	for _, feed := range pendingFeeds {
		feedInfo := map[string]interface{}{
			"feed_id":     feed.GetID(),
			"xsec_token":  feed.GetXsecToken(),
		}
		feedList = append(feedList, feedInfo)
	}

	// 格式化输出
	jsonData, err := json.MarshalIndent(feedList, "", "  ")
	if err != nil {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: fmt.Sprintf("获取待评论Feed列表失败，序列化错误: %v", err),
			}},
			IsError: true,
		}
	}

	// 返回Feed列表和说明
	instruction := "请为以下每个Feed生成评论内容，然后使用post_comment_to_feed工具逐个发表评论。完成一个评论后再处理下一个，避免同时打开多个页面。"
	resultText := fmt.Sprintf("%s\n\n%s", instruction, string(jsonData))

	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}

// handleExecuteBatchCommentWithPageReuse 处理执行批量评论（页面复用版本）
func (s *AppServer) handleExecuteBatchCommentWithPageReuse(ctx context.Context, args map[string]interface{}) *MCPToolResult {
	logrus.Info("MCP: 执行批量评论（页面复用版本）")

	// 解析参数
	limit, _ := args["limit"].(int)
	if limit <= 0 {
		limit = 0 // 0表示处理所有待评论的Feed
	}

	logrus.Infof("MCP: 执行批量评论 - 限制数量: %d", limit)

	// 检查是否有待评论的Feed
	pendingFeeds := xiaohongshu.GlobalFeedStorage.GetPendingFeeds()
	if len(pendingFeeds) == 0 {
		return &MCPToolResult{
			Content: []MCPContent{{
				Type: "text",
				Text: "没有待评论的Feed",
			}},
		}
	}

	logrus.Infof("发现 %d 个待评论的Feed，开始逐个处理（页面复用模式）", len(pendingFeeds))

	// 限制处理数量
	if limit > 0 && limit < len(pendingFeeds) {
		pendingFeeds = pendingFeeds[:limit]
	}

	// 统计结果
	successCount := 0
	failedCount := 0

	// 逐个处理Feed
	for i, feed := range pendingFeeds {
		feedID := feed.GetID()
		xsecToken := feed.GetXsecToken()
		
		logrus.Infof("处理第 %d/%d 个Feed: %s", i+1, len(pendingFeeds), feedID)

		// 获取详情页并保持页面打开
		_, page, err := s.xiaohongshuService.GetFeedDetailWithPage(ctx, feedID, xsecToken)
		if err != nil {
			logrus.Errorf("获取Feed详情失败: %v", err)
			failedCount++
			xiaohongshu.GlobalFeedStorage.MarkFeedAsCommented(feedID, false)
			continue
		}

		// 在同一页面上发表评论
		commentContent := "很棒的内容，感谢分享！" // 这里可以由大模型生成
		_, err = s.xiaohongshuService.PostCommentToFeedOnPage(ctx, feedID, xsecToken, commentContent, page)
		
		// 关闭页面
		page.Close()
		
		if err != nil {
			logrus.Errorf("发表评论失败: %v", err)
			failedCount++
			xiaohongshu.GlobalFeedStorage.MarkFeedAsCommented(feedID, false)
		} else {
			logrus.Infof("成功评论Feed: %s", feedID)
			successCount++
			xiaohongshu.GlobalFeedStorage.MarkFeedAsCommented(feedID, true)
		}

		// 添加延迟，避免请求过于频繁
		if i < len(pendingFeeds)-1 {
			logrus.Infof("等待2秒后处理下一个Feed...")
			time.Sleep(2 * time.Second)
		}
	}

	resultText := fmt.Sprintf("批量评论完成 - 成功: %d, 失败: %d", successCount, failedCount)
	return &MCPToolResult{
		Content: []MCPContent{{
			Type: "text",
			Text: resultText,
		}},
	}
}
