package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ushopal/rss-reader/internal/config"
	"github.com/ushopal/rss-reader/internal/services"
)

// FeishuHandler 处理飞书登录/绑定
type FeishuHandler struct {
	cfg        *config.FeishuConfig
	api        services.FeishuAPI
	authSvc    *services.AuthService
	feishuAuth *services.FeishuAuthService
}

func NewFeishuHandler(cfg *config.FeishuConfig, api services.FeishuAPI, authSvc *services.AuthService, feishuAuth *services.FeishuAuthService) *FeishuHandler {
	return &FeishuHandler{
		cfg:        cfg,
		api:        api,
		authSvc:    authSvc,
		feishuAuth: feishuAuth,
	}
}

// LoginURL 返回飞书扫码登录用的 URL 与 goto（供前端弹窗内嵌二维码，不跳转）
// GET /api/auth/feishu/login-url
func (h *FeishuHandler) LoginURL(c *gin.Context) {
	if h.cfg.AppID == "" || h.cfg.Redirect == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置飞书登录"})
		return
	}
	state := "login"
	params := url.Values{}
	params.Set("client_id", h.cfg.AppID)
	params.Set("redirect_uri", h.cfg.Redirect)
	params.Set("response_type", "code")
	params.Set("state", state)
	gotoURL := "https://www.feishu.cn/suite/passport/oauth/authorize?" + params.Encode()
	loginURL := "/api/auth/feishu/login?state=" + state
	c.JSON(http.StatusOK, gin.H{"url": loginURL, "goto": gotoURL})
}

// LoginRedirect 跳转到飞书登录页（passport 体系，与扫码/网页授权一致）
// GET /api/auth/feishu/login?state=...
// 必须使用 www.feishu.cn/suite/passport/oauth/authorize，否则扫码会报 4401
func (h *FeishuHandler) LoginRedirect(c *gin.Context) {
	if h.cfg.AppID == "" || h.cfg.Redirect == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未配置飞书登录"})
		return
	}
	state := c.Query("state")
	if state == "" {
		state = "STATE"
	}
	params := url.Values{}
	params.Set("client_id", h.cfg.AppID)
	params.Set("redirect_uri", h.cfg.Redirect)
	params.Set("response_type", "code")
	params.Set("state", state)
	authURL := "https://www.feishu.cn/suite/passport/oauth/authorize?" + params.Encode()
	c.Redirect(http.StatusFound, authURL)
}

// bindCallbackHTML 返回绑定结果页。若在弹窗中打开（window.opener 存在）则 postMessage 给 opener 并关闭弹窗，否则 postMessage 给 parent（iframe 场景），避免主页面跳转
func bindCallbackHTML(msgType, messageJSON string) string {
	const successScript = `(function(){
		var target = window.opener || window.parent;
		try { target.postMessage({type:'feishu_bind_success'}, '*'); } catch(e) {}
		if (window.opener) try { window.close(); } catch(e) {}
	})();`
	const errorScriptFmt = `(function(){
		var target = window.opener || window.parent;
		try { target.postMessage({type:'feishu_bind_error',message:%s}, '*'); } catch(e) {}
		if (window.opener) try { window.close(); } catch(e) {}
	})();`
	if msgType == "feishu_bind_success" {
		return `<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><p>绑定成功，窗口将自动关闭。</p><script>` + successScript + `</script></body></html>`
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><p>绑定失败</p><script>%s</script></body></html>`, fmt.Sprintf(errorScriptFmt, messageJSON))
}

// Callback 飞书回调
// GET /api/auth/feishu/callback?code=...&state=...
func (h *FeishuHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")
	if code == "" {
		c.String(http.StatusBadRequest, "缺少 code")
		return
	}
	info, err := h.api.GetUserInfo(code)
	if err != nil {
		c.String(http.StatusBadRequest, "获取飞书用户信息失败: %v", err)
		return
	}

	if strings.HasPrefix(state, "bind:") {
		// 绑定模式：bind:<userID>；回调在 iframe 中加载，通过 postMessage 通知父页面，避免整页跳转
		var uid uint
		_, err := fmt.Sscanf(state, "bind:%d", &uid)
		if err != nil || uid == 0 {
			c.Header("Content-Type", "text/html; charset=utf-8")
			msgBytes, _ := json.Marshal("无效的绑定 state")
			c.String(http.StatusOK, bindCallbackHTML("feishu_bind_error", string(msgBytes)))
			return
		}
		if err := h.feishuAuth.BindFeishuToUser(uid, info); err != nil {
			c.Header("Content-Type", "text/html; charset=utf-8")
			msgBytes, _ := json.Marshal(err.Error())
			c.String(http.StatusOK, bindCallbackHTML("feishu_bind_error", string(msgBytes)))
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(http.StatusOK, bindCallbackHTML("feishu_bind_success", ""))
		return
	}

	// 登录模式：回调在 iframe 中加载，通过 postMessage 把 token/user 或错误传给父页面，避免整页跳转
	user, created, locked, err := h.feishuAuth.LoginOrCreateByFeishu(info)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		msgBytes, _ := json.Marshal("登录失败: " + err.Error())
		c.String(http.StatusOK, loginCallbackHTML("feishu_login_error", "", "", string(msgBytes)))
		return
	}
	if locked {
		msg := "账号已锁定，请联系管理员解锁后再登录。"
		if created {
			msg = "已为你创建账号，但当前处于锁定状态，请联系管理员解锁后再登录。"
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		msgBytes, _ := json.Marshal(msg)
		c.String(http.StatusOK, loginCallbackHTML("feishu_login_error", "", "", string(msgBytes)))
		return
	}

	token, err := h.authSvc.GenerateTokenForUser(user)
	if err != nil {
		c.Header("Content-Type", "text/html; charset=utf-8")
		msgBytes, _ := json.Marshal("生成 token 失败")
		c.String(http.StatusOK, loginCallbackHTML("feishu_login_error", "", "", string(msgBytes)))
		return
	}

	frontUser := struct {
		ID           uint      `json:"id"`
		Username     string    `json:"username"`
		Status       string    `json:"status"`
		IsSuperAdmin bool      `json:"is_super_admin"`
		CreatedAt    time.Time `json:"created_at"`
	}{
		ID:           user.ID,
		Username:     user.Username,
		Status:       user.Status,
		IsSuperAdmin: user.IsSuperAdmin,
		CreatedAt:    user.CreatedAt,
	}
	userJSON, _ := json.Marshal(frontUser)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, loginCallbackHTML("feishu_login_success", token, string(userJSON), ""))
}

// loginCallbackHTML 返回登录结果页，通过 postMessage 通知父页面（iframe 场景），不跳转
func loginCallbackHTML(msgType, token, userJSON, messageJSON string) string {
	if msgType == "feishu_login_success" {
		// 将 token、user 传给父页面，由前端写入 localStorage 并跳转
		return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><script>
(function(){
  var target = window.opener || window.parent;
  try { target.postMessage({type:'feishu_login_success',token:%q,user:%s}, '*'); } catch(e) {}
  if (window.opener) try { window.close(); } catch(e) {}
})();
</script></body></html>`, token, userJSON)
	}
	return fmt.Sprintf(`<!DOCTYPE html><html><head><meta charset="utf-8"></head><body><script>
(function(){
  var target = window.opener || window.parent;
  try { target.postMessage({type:'feishu_login_error',message:%s}, '*'); } catch(e) {}
  if (window.opener) try { window.close(); } catch(e) {}
})();
</script></body></html>`, messageJSON)
}

// BindURL 生成绑定飞书的登录地址，供前端跳转或飞书扫码 SDK 使用
// GET /api/admin/users/:id/feishu/bind-url
// 返回 url（本站登录入口）与 goto（飞书 passport 授权页完整 URL，用于 SDK 的 goto 参数）
func (h *FeishuHandler) BindURL(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}
	loginURL := fmt.Sprintf("/api/auth/feishu/login?state=bind:%d", id)
	out := gin.H{"url": loginURL}
	if h.cfg.AppID != "" && h.cfg.Redirect != "" {
		params := url.Values{}
		params.Set("client_id", h.cfg.AppID)
		params.Set("redirect_uri", h.cfg.Redirect)
		params.Set("response_type", "code")
		params.Set("state", fmt.Sprintf("bind:%d", id))
		out["goto"] = "https://www.feishu.cn/suite/passport/oauth/authorize?" + params.Encode()
	}
	c.JSON(http.StatusOK, out)
}


