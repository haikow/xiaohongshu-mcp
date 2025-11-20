package xiaohongshu

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

// CommentFeedAction 表示 Feed 评论动作
type CommentFeedAction struct {
	page *rod.Page
}

// NewCommentFeedAction 创建 Feed 评论动作
func NewCommentFeedAction(page *rod.Page) *CommentFeedAction {
	return &CommentFeedAction{page: page}
}

// PostComment 发表评论到 Feed
func (f *CommentFeedAction) PostComment(ctx context.Context, feedID, xsecToken, content string) error {
	page := f.page.Context(ctx).Timeout(60 * time.Second)

	// 构建详情页 URL
	url := makeFeedDetailURL(feedID, xsecToken)

	logrus.Infof("Opening feed detail page: %s", url)

	// 导航到详情页
	if err := page.Navigate(url); err != nil {
		logrus.Warnf("Failed to navigate to feed detail page: %v", err)
		return fmt.Errorf("无法打开帖子详情页，该帖子可能在网页端不可访问: %w", err)
	}

	if err := page.WaitStable(2 * time.Second); err != nil {
		logrus.Warnf("Failed to wait for page stable: %v", err)
		return fmt.Errorf("页面加载超时，该帖子可能在网页端不可访问: %w", err)
	}

	time.Sleep(1 * time.Second)

	// 查找评论输入框
	elem, err := page.Element("div.input-box div.content-edit span")
	if err != nil {
		logrus.Warnf("Failed to find comment input box: %v", err)
		return fmt.Errorf("未找到评论输入框，该帖子可能不支持评论或网页端不可访问: %w", err)
	}

	if err := elem.Click(proto.InputMouseButtonLeft, 1); err != nil {
		logrus.Warnf("Failed to click comment input box: %v", err)
		return fmt.Errorf("无法点击评论输入框: %w", err)
	}

	elem2, err := page.Element("div.input-box div.content-edit p.content-input")
	if err != nil {
		logrus.Warnf("Failed to find comment input field: %v", err)
		return fmt.Errorf("未找到评论输入区域: %w", err)
	}

	if err := elem2.Input(content); err != nil {
		logrus.Warnf("Failed to input comment content: %v", err)
		return fmt.Errorf("无法输入评论内容: %w", err)
	}

	time.Sleep(1 * time.Second)

	submitButton, err := page.Element("div.bottom button.submit")
	if err != nil {
		logrus.Warnf("Failed to find submit button: %v", err)
		return fmt.Errorf("未找到提交按钮: %w", err)
	}

	if err := submitButton.Click(proto.InputMouseButtonLeft, 1); err != nil {
		logrus.Warnf("Failed to click submit button: %v", err)
		return fmt.Errorf("无法点击提交按钮: %w", err)
	}

	time.Sleep(1 * time.Second)

	logrus.Infof("Comment posted successfully to feed: %s", feedID)
	return nil
}

// ReplyToComment 回复指定评论
func (f *CommentFeedAction) ReplyToComment(ctx context.Context, feedID, xsecToken, commentID, userID, content string) error {
	page := f.page.Context(ctx).Timeout(60 * time.Second)
	url := makeFeedDetailURL(feedID, xsecToken)
	logrus.Infof("Opening feed detail page for reply: %s", url)
	page.MustNavigate(url)
	page.MustWaitDOMStable()
	time.Sleep(3 * time.Second)

	// 等待评论容器加载
	waitForCommentsContainer(page)

	// 确保评论区域可见
	ensureCommentsVisible(page)

	// 额外等待确保评论内容加载完成
	time.Sleep(2 * time.Second)

	// 尝试多次查找评论元素
	var commentEl *rod.Element
	var err error
	maxAttempts := 15
	consecutiveFailures := 0

	for attempt := 0; attempt < maxAttempts; attempt++ {
		commentEl, err = findCommentElement(page, commentID, userID)
		if err == nil {
			logrus.Infof("成功找到评论元素，尝试次数: %d", attempt+1)
			break
		}

		logrus.Warnf("Attempt %d/%d: Failed to find comment: %v", attempt+1, maxAttempts, err)
		consecutiveFailures++

		if consecutiveFailures >= 5 {
			logrus.Warnf("连续 %d 次查找失败，等待更长时间", consecutiveFailures)
			time.Sleep(3 * time.Second)
		} else {
			time.Sleep(1500 * time.Millisecond)
		}

		ensureCommentsVisible(page)

		if reachedCommentsEnd(page) {
			logrus.Infof("已到达评论底部，在第 %d 次尝试后", attempt+1)
			if attempt >= maxAttempts-3 {
				logrus.Warnf("已到达评论底部且接近最大尝试次数，提前结束")
				break
			}
		}
	}

	if err != nil {
		return fmt.Errorf("无法找到评论: %w", err)
	}

	// 滚动到评论位置
	_, _ = commentEl.Eval(`() => { try { this.scrollIntoView({behavior: "instant", block: "center"}); } catch (e) {} return true }`)
	time.Sleep(1 * time.Second)

	// 尝试多次点击回复按钮
	var replyBtn *rod.Element
	for attempt := 0; attempt < 5; attempt++ {
		replyBtn, err = findReplyButton(commentEl)
		if err == nil {
			if tryClickChainForComment(replyBtn) {
				break
			}
		}
		logrus.Warnf("Attempt %d: Failed to click reply button: %v", attempt+1, err)
		time.Sleep(1 * time.Second)
	}

	if err != nil || replyBtn == nil {
		return fmt.Errorf("无法点击回复按钮")
	}

	time.Sleep(2 * time.Second)

	// 查找回复输入框
	inputEl, err := findReplyInput(page, commentEl)
	if err != nil {
		return fmt.Errorf("无法找到回复输入框: %w", err)
	}

	// 聚焦并输入内容
	if _, evalErr := inputEl.Eval(`() => { try { this.focus(); } catch (e) {} return true }`); evalErr != nil {
		logrus.Warnf("focus reply input failed: %v", evalErr)
	}

	inputEl.MustInput(content)
	time.Sleep(500 * time.Millisecond)

	// 查找并点击提交按钮
	submitBtn, err := findSubmitButton(page)
	if err != nil {
		return fmt.Errorf("无法找到提交按钮: %w", err)
	}

	if !tryClickChainForComment(submitBtn) {
		return fmt.Errorf("点击回复提交按钮失败")
	}

	time.Sleep(3 * time.Second)
	return nil
}

func findCommentElement(page *rod.Page, commentID, userID string) (*rod.Element, error) {
	var lastErr error

	ensureCommentsVisible(page)

	identifier := buildIdentifier(commentID, userID)
	maxAttempts := 60
	attempt := 0
	consecutiveNoContentLoad := 0
	lastScrollPos := 0
	totalClickedButtons := 0

	previousScrollHeight := getScrollHeight(page)

	// 第一阶段：向下滚动到底部，同时点击所有"更多"按钮
	logrus.Infof("开始第一阶段：向下滚动到底部并展开所有\"更多\"")
	for attempt < maxAttempts {
		attempt++
		logrus.Infof("查找评论，尝试次数: %d/%d", attempt, maxAttempts)

		// 每次滚动都点击"更多"按钮
		clicked := clickShowMoreButtons(page)
		if clicked > 0 {
			totalClickedButtons += clicked
			logrus.Infof("点击了 %d 个\"更多\"按钮，累计: %d", clicked, totalClickedButtons)
			time.Sleep(300 * time.Millisecond)

			// 点击后立即查找
			el, err := locateCommentElement(page, commentID, userID)
			if err == nil && el != nil {
				logrus.Infof("点击\"更多\"后立即找到评论")
				return el, nil
			}
		}

		// 查找目标评论
		el, err := locateCommentElement(page, commentID, userID)
		if err == nil && el != nil {
			logrus.Infof("成功找到评论，共点击了 %d 个\"更多\"按钮", totalClickedButtons)
			return el, nil
		}
		if err != nil {
			lastErr = err
		}

		time.Sleep(1500 * time.Millisecond)

		isVisualEnd := reachedCommentsEnd(page)

		// 滚动
		scrolled := scrollComments(page)
		if !scrolled {
			performFullScroll(page)
			time.Sleep(1000 * time.Millisecond)
		}

		// 滚动后再次点击"更多"按钮（因为新内容可能有新的"更多"）
		clicked = clickShowMoreButtons(page)
		if clicked > 0 {
			totalClickedButtons += clicked
			logrus.Infof("滚动后点击了 %d 个\"更多\"按钮", clicked)
			time.Sleep(300 * time.Millisecond)
		}

		newScrollPos := getCurrentScrollPosition(page)
		newScrollHeight := getScrollHeight(page)

		actuallyScrolled := newScrollPos > lastScrollPos
		scrollHeightIncreased := newScrollHeight > previousScrollHeight

		if actuallyScrolled || scrollHeightIncreased {
			lastScrollPos = newScrollPos
			consecutiveNoContentLoad = 0
		} else {
			consecutiveNoContentLoad++
		}

		// 到达底部
		if consecutiveNoContentLoad >= 8 || isVisualEnd {
			logrus.Infof("已到达底部，开始第二阶段：从头到尾再次查找")
			break
		}

		previousScrollHeight = newScrollHeight
	}

	// 第二阶段：从头到尾点击所有"更多"按钮并查找
	logrus.Infof("第二阶段：滚回顶部")
	scrollToTop(page)
	time.Sleep(2 * time.Second)

	// 重新从顶部开始，这次更细致地查找
	logrus.Infof("第二阶段：从顶部开始细致查找")
	previousScrollHeight = 0
	lastScrollPos = 0
	consecutiveNoContentLoad = 0
	secondPhaseAttempts := 0
	maxSecondPhaseAttempts := 40

	for secondPhaseAttempts < maxSecondPhaseAttempts {
		secondPhaseAttempts++
		logrus.Infof("第二阶段查找，尝试次数: %d/%d", secondPhaseAttempts, maxSecondPhaseAttempts)

		// 点击当前视口内所有"更多"按钮
		clicked := clickShowMoreButtons(page)
		if clicked > 0 {
			totalClickedButtons += clicked
			logrus.Infof("第二阶段点击了 %d 个\"更多\"按钮", clicked)
			time.Sleep(500 * time.Millisecond)

			// 点击后等待内容展开，再点击一次
			for i := 0; i < 3; i++ {
				time.Sleep(300 * time.Millisecond)
				clicked2 := clickShowMoreButtons(page)
				if clicked2 > 0 {
					totalClickedButtons += clicked2
					logrus.Infof("第二阶段额外点击了 %d 个\"更多\"按钮", clicked2)
				}
			}
		}

		// 查找目标评论
		el, err := locateCommentElement(page, commentID, userID)
		if err == nil && el != nil {
			logrus.Infof("第二阶段找到评论，共点击了 %d 个\"更多\"按钮", totalClickedButtons)
			return el, nil
		}

		time.Sleep(1000 * time.Millisecond)

		// 缓慢滚动，确保不错过任何内容
		scrolled := scrollCommentsSlowly(page)
		if !scrolled {
			logrus.Infof("第二阶段无法继续滚动，可能已到底部")
			break
		}

		newScrollPos := getCurrentScrollPosition(page)
		if newScrollPos <= lastScrollPos {
			consecutiveNoContentLoad++
		} else {
			lastScrollPos = newScrollPos
			consecutiveNoContentLoad = 0
		}

		if consecutiveNoContentLoad >= 5 {
			logrus.Infof("第二阶段连续 %d 次无法滚动，结束查找", consecutiveNoContentLoad)
			break
		}
	}

	logrus.Infof("结束查找评论，总尝试次数: %d，共点击了 %d 个\"更多\"按钮", attempt+secondPhaseAttempts, totalClickedButtons)
	if lastErr != nil {
		return nil, lastErr
	}
	if identifier != "" {
		return nil, fmt.Errorf("未找到评论: %s", identifier)
	}
	return nil, fmt.Errorf("未找到目标评论")
}

// scrollToTop 滚动到页面顶部
func scrollToTop(page *rod.Page) {
	scrollJS := `() => {
		const commentsContainer = document.querySelector('.comments-container');
		if (commentsContainer) {
			commentsContainer.scrollIntoView({behavior: 'instant', block: 'start'});
		}
		window.scrollTo(0, 0);
		const scrollRoot = document.scrollingElement || document.documentElement || document.body;
		scrollRoot.scrollTop = 0;
		return true;
	}`
	page.Eval(scrollJS)
}

// scrollCommentsSlowly 缓慢滚动评论区，每次滚动少一点
func scrollCommentsSlowly(page *rod.Page) bool {
	scrollJS := `() => {
		try {
			const DELTA = 300; // 每次只滚动300px
			
			const scrollRoot = document.scrollingElement || document.documentElement || document.body;
			const container = document.querySelector('.comments-container');
			
			const metrics = (el) => {
				if (!el) {
					return { top: 0, max: 0 };
				}
				if (el === window || el === document || el === document.body || el === document.documentElement) {
					const root = scrollRoot;
					return {
						top: root.scrollTop,
						max: Math.max(root.scrollHeight - root.clientHeight, 0)
					};
				}
				return {
					top: el.scrollTop,
					max: Math.max(el.scrollHeight - el.clientHeight, 0)
				};
			};
			
			const setScrollTop = (el, value) => {
				if (!el) return;
				if (el === window || el === document || el === document.body || el === document.documentElement || el === scrollRoot) {
					scrollRoot.scrollTop = value;
					window.scrollBy(0, value - scrollRoot.scrollTop);
				} else {
					el.scrollTop = value;
				}
			};
			
			// 优先滚动评论容器
			let target = container || scrollRoot;
			const before = metrics(target);
			const desired = Math.min(before.top + DELTA, before.max);
			
			if (desired > before.top) {
				setScrollTop(target, desired);
				return true;
			}
			
			return false;
		} catch (err) {
			console.debug('scrollCommentsSlowly error', err);
			return false;
		}
	}`
	res, err := page.Eval(scrollJS)
	if err != nil {
		logrus.Warnf("缓慢滚动失败: %v", err)
		return false
	}
	if res == nil {
		return false
	}
	return res.Value.Bool()
}

// clickShowMoreButtons 点击所有"更多"按钮
func clickShowMoreButtons(page *rod.Page) int {
	clickJS := `() => {
		const selectors = [
			'.show-more',
			'.show-more-btn',
			'[class*="show-more"]',
			'[class*="showMore"]'
		];
		
		const clickedElements = new Set();
		let clickedCount = 0;
		
		selectors.forEach((selector) => {
			try {
				const elements = document.querySelectorAll(selector);
				elements.forEach((el) => {
					if (clickedElements.has(el)) return;
					
					const text = el.textContent || '';
					const className = el.className || '';
					const shouldClick = text.includes('更多') || 
					                   className.includes('show-more') || 
					                   className.includes('showMore');
					
					if (!shouldClick) return;
					
					const rect = el.getBoundingClientRect();
					const style = window.getComputedStyle(el);
					const isVisible = (
						rect.height > 0 &&
						rect.width > 0 &&
						style.display !== 'none' &&
						style.visibility !== 'hidden' &&
						style.opacity !== '0' &&
						rect.top < window.innerHeight + 500 &&
						rect.bottom > -500
					);
					
					if (isVisible) {
						try {
							el.click();
							
							if (el.parentElement && el.parentElement.classList.contains('show-more')) {
								el.parentElement.click();
							}
							
							clickedElements.add(el);
							clickedCount++;
						} catch (err) {
							console.debug('点击失败', err);
						}
					}
				});
			} catch (err) {
				console.debug('选择器错误: ' + selector, err);
			}
		});
		
		return clickedCount;
	}`

	res, err := page.Eval(clickJS)
	if err != nil {
		logrus.Warnf("点击\"更多\"按钮失败: %v", err)
		return 0
	}

	if res == nil || res.Value.Num() == 0 {
		return 0
	}

	return int(res.Value.Num())
}

func locateCommentElement(page *rod.Page, commentID, userID string) (*rod.Element, error) {
	if commentID != "" {
		if el, err := locateCommentElementByCommentID(page, commentID); err == nil && el != nil {
			return el, nil
		}
	}
	if userID != "" {
		if el, err := locateCommentElementByUserID(page, userID); err == nil && el != nil {
			return el, nil
		}
	}

	identifier := buildIdentifier(commentID, userID)
	if identifier == "" {
		return nil, fmt.Errorf("未提供评论标识")
	}
	return nil, fmt.Errorf("未找到评论: %s", identifier)
}

func locateCommentElementByCommentID(page *rod.Page, commentID string) (*rod.Element, error) {
	if commentID == "" {
		return nil, fmt.Errorf("评论ID为空")
	}

	idSelector := fmt.Sprintf("#comment-%s", commentID)
	el, err := page.Element(idSelector)
	if err != nil {
		return nil, fmt.Errorf("未找到评论ID: %s", commentID)
	}
	err = el.WaitVisible()
	if err != nil {
		return nil, fmt.Errorf("评论ID %s 不可见: %w", commentID, err)
	}
	return el, nil
}

func locateCommentElementByUserID(page *rod.Page, userID string) (*rod.Element, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID为空")
	}

	selectors := []string{
		fmt.Sprintf(`[data-user-id="%s"]`, userID),
	}

	for _, selector := range selectors {
		el, err := page.Element(selector)
		if err != nil {
			continue
		}
		err = el.WaitVisible()
		if err != nil {
			logrus.Warnf("用户ID %s 的评论元素不可见: %v", userID, err)
			continue
		}

		jsCode := `() => {
			let current = this;
			while (current) {
				if (current.classList && (current.classList.contains('comment-item') || current.classList.contains('comment'))) {
					return current;
				}
				current = current.parentElement;
			}
			return this;
		}`
		if _, err := el.Eval(jsCode); err == nil {
			return el, nil
		}
		return el, nil
	}

	return nil, fmt.Errorf("未找到用户ID: %s", userID)
}

func waitForCommentsContainer(page *rod.Page) {
	jsCode := `() => {
		let attempts = 0;
		const maxAttempts = 10;
		
		const checkContainer = () => {
			const container = document.querySelector('.comments-container');
			if (container) {
				const comments = container.querySelectorAll('.comment-item, .comment');
				return comments.length > 0;
			}
			return false;
		};
		
		const interval = setInterval(() => {
			attempts++;
			if (checkContainer() || attempts >= maxAttempts) {
				clearInterval(interval);
			}
		}, 500);
		
		return checkContainer();
	}`

	page.Eval(jsCode)
	time.Sleep(2 * time.Second)
}

func ensureCommentsVisible(page *rod.Page) {
	jsCode := `() => {
		const commentsContainer = document.querySelector('.comments-container');
		if (!commentsContainer) {
			return false;
		}
		if (commentsContainer.dataset._xhEnsured === '1') {
			return true;
		}
		commentsContainer.dataset._xhEnsured = '1';
		try {
			commentsContainer.scrollIntoView({behavior: 'instant', block: 'start'});
		} catch (e) {}
		return true;
	}`

	page.Eval(jsCode)
	time.Sleep(1 * time.Second)
}

func scrollComments(page *rod.Page) bool {
	scrollJS := `() => {
		try {
			const END_SELECTOR = '.end-container';
			const DELTA_MIN = 480;
			const MAX_FRAME_WAIT = 4;

			const reachedEnd = () => {
				const endEl = document.querySelector(END_SELECTOR);
				if (!endEl) return false;
				const text = (endEl.textContent || '').toUpperCase();
				return text.includes('THE END');
			};

			if (reachedEnd()) {
				return false;
			}

			const scrollRoot = document.scrollingElement || document.documentElement || document.body;
			const container = document.querySelector('.comments-container');
			const candidatesSet = new Set();

			const pushCandidate = (node) => {
				if (node && node instanceof HTMLElement) {
					candidatesSet.add(node);
				}
			};

			if (container) {
				let current = container;
				while (current) {
					pushCandidate(current);
					if (current === document.body || current === document.documentElement) {
						break;
					}
					current = current.parentElement;
				}
				container.querySelectorAll('.comments-el, .list-container, [data-v-4a19279a][name="list"]').forEach(pushCandidate);
			}

			pushCandidate(scrollRoot);
			pushCandidate(document.body);
			pushCandidate(document.documentElement);

			const metrics = (el) => {
				if (!el) {
					return { top: 0, max: 0, client: window.innerHeight };
				}
				if (el === window || el === document || el === document.body || el === document.documentElement) {
					const root = scrollRoot;
					return {
						top: root.scrollTop,
						max: Math.max(root.scrollHeight - root.clientHeight, 0),
						client: root.clientHeight || window.innerHeight
					};
				}
				return {
					top: el.scrollTop,
					max: Math.max(el.scrollHeight - el.clientHeight, 0),
					client: el.clientHeight
				};
			};

			const setScrollTop = (el, value) => {
				if (!el) return;
				if (el === window || el === document || el === document.body || el === document.documentElement || el === scrollRoot) {
					scrollRoot.scrollTop = value;
				} else {
					el.scrollTop = value;
				}
			};

			const dispatchWheel = (el, delta) => {
				if (!el) return;
				try {
					el.dispatchEvent(new Event('scroll', { bubbles: true }));
					if (typeof WheelEvent === 'function' && delta !== 0) {
						const wheel = new WheelEvent('wheel', { deltaY: delta, bubbles: true, cancelable: true });
						el.dispatchEvent(wheel);
					}
				} catch (e) {
					console.debug('dispatchWheel error', e);
				}
			};

			const waitForUpdatedScrollTop = (el, beforeTop) => {
				let tries = 0;
				return new Promise((resolve) => {
					const check = () => {
						tries++;
						const current = metrics(el).top;
						if (Math.abs(current - beforeTop) >= 5 || tries >= MAX_FRAME_WAIT) {
							resolve(Math.abs(current - beforeTop) >= 5);
							return;
						}
						setTimeout(check, 60);
					};
					setTimeout(check, 60);
				});
			};

			const weighted = Array.from(candidatesSet).map((node) => {
				const style = window.getComputedStyle(node);
				const overflowY = style.overflowY;
				const scrollable = node.scrollHeight - node.clientHeight > 40;
				const hasScrollStyle = /auto|scroll|overlay/i.test(overflowY);
				const weight =
					(container && node === container ? 1200 : 0) +
					(container && node.contains && node !== container && node.contains(container) ? 800 : 0) +
					(hasScrollStyle ? 300 : 0) +
					(scrollable ? 300 : 0) -
					(node === document.body || node === document.documentElement ? 50 : 0);
				return { node, weight };
			});

			weighted.sort((a, b) => b.weight - a.weight);

			const candidates = weighted.slice(0, 6);
			const tryScroll = async (node) => {
				const before = metrics(node);
				const delta = Math.max(before.client * 0.85, DELTA_MIN);
				const desired = before.max > 0 ? Math.min(before.top + delta, before.max) : before.top + delta;
				const applied = Math.max(0, desired - before.top);

				setScrollTop(node, desired);
				dispatchWheel(node, applied);
				window.scrollBy(0, applied);

				const moved = await waitForUpdatedScrollTop(node, before.top);
				if (!moved && node !== scrollRoot) {
					const rootBefore = metrics(scrollRoot).top;
					setScrollTop(scrollRoot, rootBefore + delta);
					dispatchWheel(scrollRoot, delta);
					window.scrollBy(0, delta);
					return waitForUpdatedScrollTop(scrollRoot, rootBefore);
				}
				return moved;
			};

			const run = async () => {
				for (const { node } of candidates) {
					if (await tryScroll(node)) {
						return true;
					}
				}
				return false;
			};

			return run().then((moved) => moved && !reachedEnd());
		} catch (err) {
			console.debug('scrollComments error', err);
			return false;
		}
	}`
	res, err := page.Eval(scrollJS)
	if err != nil {
		logrus.Warnf("scroll comments failed: %v", err)
		return false
	}
	if res == nil {
		return false
	}
	return res.Value.Bool()
}

func performFullScroll(page *rod.Page) {
	logrus.Infof("执行彻底滚动策略")

	scrollPositionsJS := `() => {
		try {
			const END_SELECTOR = '.end-container';
			const DELTA_MIN = 480;
			const MAX_FRAME_WAIT = 4;
			const MAX_ATTEMPTS = 5;

			const reachedEnd = () => {
				const endEl = document.querySelector(END_SELECTOR);
				if (!endEl) return false;
				return (endEl.textContent || '').toUpperCase().includes('THE END');
			};

			const scrollRoot = document.scrollingElement || document.documentElement || document.body;
			const container = document.querySelector('.comments-container');
			const candidatesSet = new Set();

			const pushCandidate = (node) => {
				if (node && node instanceof HTMLElement) {
					candidatesSet.add(node);
				}
			};

			if (container) {
				let current = container;
				while (current) {
					pushCandidate(current);
					if (current === document.body || current === document.documentElement) {
						break;
					}
					current = current.parentElement;
				}
				container.querySelectorAll('.comments-el, .list-container, [data-v-4a19279a][name="list"]').forEach(pushCandidate);
			}

			pushCandidate(scrollRoot);
			pushCandidate(document.body);
			pushCandidate(document.documentElement);

			const metrics = (el) => {
				if (!el) {
					return { top: 0, max: 0, client: window.innerHeight };
				}
				if (el === window || el === document || el === document.body || el === document.documentElement) {
					const root = scrollRoot;
					return {
						top: root.scrollTop,
						max: Math.max(root.scrollHeight - root.clientHeight, 0),
						client: root.clientHeight || window.innerHeight
					};
				}
				return {
					top: el.scrollTop,
					max: Math.max(el.scrollHeight - el.clientHeight, 0),
					client: el.clientHeight
				};
			};

			const setScrollTop = (el, value) => {
				if (!el) return;
				if (el === window || el === document || el === document.body || el === document.documentElement || el === scrollRoot) {
					scrollRoot.scrollTop = value;
				} else {
					el.scrollTop = value;
				}
			};

			const dispatchWheel = (el, delta) => {
				if (!el) return;
				try {
					el.dispatchEvent(new Event('scroll', { bubbles: true }));
					if (typeof WheelEvent === 'function' && delta !== 0) {
						const wheel = new WheelEvent('wheel', { deltaY: delta, bubbles: true, cancelable: true });
						el.dispatchEvent(wheel);
					}
				} catch (e) {
					console.debug('dispatchWheel error', e);
				}
			};

			const waitForUpdatedScrollTop = (el, beforeTop) => {
				let tries = 0;
				return new Promise((resolve) => {
					const check = () => {
						tries++;
						const current = metrics(el).top;
						if (Math.abs(current - beforeTop) >= 5 || tries >= MAX_FRAME_WAIT) {
							resolve(Math.abs(current - beforeTop) >= 5);
							return;
						}
						setTimeout(check, 60);
					};
					setTimeout(check, 60);
				});
			};

			const weighted = Array.from(candidatesSet).map((node) => {
				const style = window.getComputedStyle(node);
				const overflowY = style.overflowY;
				const scrollable = node.scrollHeight - node.clientHeight > 40;
				const hasScrollStyle = /auto|scroll|overlay/i.test(overflowY);
				const weight =
					(container && node === container ? 1200 : 0) +
					(container && node.contains && node !== container && node.contains(container) ? 800 : 0) +
					(hasScrollStyle ? 300 : 0) +
					(scrollable ? 300 : 0) -
					(node === document.body || node === document.documentElement ? 50 : 0);
				return { node, weight };
			});

			weighted.sort((a, b) => b.weight - a.weight);
			const candidates = weighted.slice(0, 6);

			const tryScroll = async (node) => {
				const before = metrics(node);
				const delta = Math.max(before.client * 0.85, DELTA_MIN);
				const desired = before.max > 0 ? Math.min(before.top + delta, before.max) : before.top + delta;
				const applied = Math.max(0, desired - before.top);

				setScrollTop(node, desired);
				dispatchWheel(node, applied);
				window.scrollBy(0, applied);

				let moved = await waitForUpdatedScrollTop(node, before.top);
				if (!moved && node !== scrollRoot) {
					const rootBefore = metrics(scrollRoot).top;
					setScrollTop(scrollRoot, rootBefore + delta);
					dispatchWheel(scrollRoot, delta);
					window.scrollBy(0, delta);
					moved = await waitForUpdatedScrollTop(scrollRoot, rootBefore);
				}
				return moved;
			};

			const runSequence = async () => {
				for (let attempt = 0; attempt < MAX_ATTEMPTS; attempt++) {
					for (const { node } of candidates) {
						if (await tryScroll(node)) {
							return true;
						}
					}
					if (reachedEnd()) {
						return true;
					}
				}
				return false;
			};

			if (reachedEnd()) {
				return false;
			}

			return runSequence().then((moved) => moved && !reachedEnd());
		} catch (err) {
			console.debug('performFullScroll error', err);
			return false;
		}
	}`

	if _, err := page.Eval(scrollPositionsJS); err != nil {
		logrus.Warnf("彻底滚动失败: %v", err)
	}
}

func buildIdentifier(commentID, userID string) string {
	if commentID != "" && userID != "" {
		return fmt.Sprintf("comment_id=%s / user_id=%s", commentID, userID)
	}
	if commentID != "" {
		return commentID
	}
	return userID
}

func findReplyButton(commentEl *rod.Element) (*rod.Element, error) {
	if commentEl == nil {
		return nil, fmt.Errorf("评论元素为空")
	}

	selector := ".right .interactions .reply"
	btn, err := commentEl.Element(selector)
	if err != nil || btn == nil {
		logrus.Warnf("未找到回复按钮，选择器: %s, err: %v", selector, err)
		return nil, fmt.Errorf("未找到回复按钮")
	}

	logrus.Infof("通过选择器 %s 找到回复按钮", selector)
	return btn, nil
}

func verifyClickSuccess(clickedEl *rod.Element) bool {
	page := clickedEl.Page()

	selectors := []string{
		"div.input-box div.content-edit p.content-input",
	}

	for _, selector := range selectors {
		if el, err := page.Element(selector); err == nil && el != nil {
			if visible, _ := el.Visible(); visible {
				logrus.Infof("验证成功：找到可见的回复输入框 (%s)", selector)
				return true
			}
		}
	}
	logrus.Infof("验证失败：没有找到回复输入框")
	return false
}

func findReplyInput(page *rod.Page, commentEl *rod.Element) (*rod.Element, error) {
	activeEditableJS := `() => {
        const active = document.activeElement;
        if (active && active.getAttribute && active.getAttribute('contenteditable') === 'true') {
            return active;
        }
        return null;
    }`
	if el, err := page.ElementByJS(rod.Eval(activeEditableJS)); err == nil && el != nil {
		return el, nil
	}
	selectors := []string{
		"div.input-box div.content-edit p.content-input",
	}
	for _, selector := range selectors {
		if el, err := page.Element(selector); err == nil && el != nil {
			return el, nil
		}
	}
	return nil, fmt.Errorf("未找到回复输入框")
}

func tryClickChainForComment(el *rod.Element) bool {
	if el == nil {
		logrus.Errorf("要点击的元素为空")
		return false
	}

	text, _ := el.Text()
	classAttr, _ := el.Attribute("class")
	class := ""
	if classAttr != nil {
		class = *classAttr
	}
	tagName := ""
	if desc, err := el.Describe(0, false); err == nil && desc != nil {
		tagName = desc.NodeName
	}
	logrus.Infof("准备点击元素 - 文本: '%s', 类: '%s', 标签: %s", text, class, tagName)

	visible, _ := el.Visible()
	logrus.Infof("元素可见性: %v", visible)

	_, _ = el.Eval(`() => { try { this.scrollIntoView({behavior: "instant", block: "center"}); } catch (e) {} return true }`)
	time.Sleep(500 * time.Millisecond)

	clickMethods := []struct {
		name string
		fn   func(*rod.Element) bool
	}{
		{"直接点击", func(e *rod.Element) bool {
			if err := e.Click(proto.InputMouseButtonLeft, 1); err != nil {
				logrus.Warnf("直接点击失败: %v", err)
				return false
			}
			logrus.Infof("直接点击成功")
			return true
		}},
	}

	for i, method := range clickMethods {
		logrus.Infof("尝试点击方法 %d: %s", i+1, method.name)
		if method.fn(el) {
			time.Sleep(1 * time.Second)

			success := verifyClickSuccess(el)
			if success {
				logrus.Infof("点击方法 %s 执行成功且有效", method.name)
				return true
			} else {
				logrus.Warnf("点击方法 %s 执行成功但无效（没有出现回复输入框）", method.name)
			}
		}
	}

	logrus.Errorf("所有点击方法都失败")
	return false
}

func findSubmitButton(page *rod.Page) (*rod.Element, error) {
	selectors := []string{
		"div.bottom button.submit",
	}
	for _, selector := range selectors {
		if el, err := page.Element(selector); err == nil && el != nil {
			disabled, _ := el.Attribute("disabled")
			if disabled == nil {
				return el, nil
			}
		}
	}
	return nil, fmt.Errorf("未找到回复发布按钮")
}

func getCurrentScrollPosition(page *rod.Page) int {
	js := `() => document.scrollingElement.scrollTop`
	res, err := page.Eval(js)
	if err != nil {
		logrus.Warnf("Failed to get current scroll position: %v", err)
		return 0
	}
	valStr := res.Value.Str()
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		logrus.Warnf("Failed to parse scroll position string '%s' to float64: %v", valStr, err)
		return 0
	}
	return int(val)
}

func reachedCommentsEnd(page *rod.Page) bool {
	js := `() => {
		const END_SELECTOR = '.end-container';
		const endEl = document.querySelector(END_SELECTOR);
		if (!endEl) return false;
		const text = (endEl.textContent || '').toUpperCase();
		return text.includes('THE END');
	}`
	res, err := page.Eval(js)
	if err != nil {
		logrus.Warnf("Failed to check if comments end reached: %v", err)
		return false
	}
	valStr := res.Value.Str()
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		logrus.Warnf("Failed to parse boolean string '%s': %v", valStr, err)
		return false
	}
	return val
}

func getScrollHeight(page *rod.Page) int {
	js := `() => document.scrollingElement.scrollHeight`
	res, err := page.Eval(js)
	if err != nil {
		logrus.Warnf("Failed to get scroll height: %v", err)
		return 0
	}
	valStr := res.Value.Str()
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		logrus.Warnf("Failed to parse scroll height string '%s' to float64: %v", valStr, err)
		return 0
	}
	return int(val)
}
