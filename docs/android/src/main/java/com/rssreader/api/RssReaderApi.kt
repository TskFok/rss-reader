package com.rssreader.api

import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Query

/**
 * 与 rss-reader 后端 /api 前缀对齐的 Retrofit 接口（节选：登录、飞书入口、订阅、文章、总结历史）。
 * 注销：无服务端接口，客户端删除本地 token 即可。
 */
interface RssReaderApi {

    @POST("auth/login")
    suspend fun login(@Body body: LoginRequest): LoginResponse

    /** 飞书：获取 goto 授权 URL（公开） */
    @GET("auth/feishu/login-url")
    suspend fun feishuLoginUrl(): FeishuLoginUrlResponse

    @GET("feeds")
    suspend fun listFeeds(): List<FeedJson>

    /**
     * 文章列表；点击某一订阅时传入 [feedId]。
     */
    @GET("articles")
    suspend fun listArticles(
        @Query("feed_id") feedId: Long? = null,
        @Query("read") read: Boolean? = null,
        @Query("page") page: Int? = null,
        @Query("page_size") pageSize: Int? = null,
    ): ArticleListResponse

    @GET("summary-histories")
    suspend fun listSummaryHistories(
        @Query("page") page: Int? = null,
        @Query("page_size") pageSize: Int? = null,
    ): SummaryHistoryListResponse
}

// --- Request / Response（可按项目改用 Moshi @Json 或 Gson） ---

data class LoginRequest(
    val username: String,
    val password: String,
)

data class LoginResponse(
    val token: String,
    val user: UserJson,
)

data class UserJson(
    val id: Long,
    val username: String,
    val status: String,
    val is_super_admin: Boolean,
    val feishu_id: String? = null,
    val feishu_name: String? = null,
    val created_at: String? = null,
)

data class FeishuLoginUrlResponse(
    val url: String,
    val goto: String,
)

/** 与后端 JSON 字段一致时可简化；此处为常用字段子集 */
data class FeedJson(
    val id: Long,
    val user_id: Long,
    val url: String,
    val title: String,
    val update_interval_minutes: Int,
    val expire_days: Int,
    val ai_model_id: Long? = null,
    val ai_classify_enabled: Boolean = false,
    val ai_translate_enabled: Boolean = false,
    val ai_target_language: String = "",
    val last_fetched_at: String? = null,
    val created_at: String? = null,
)

data class ArticleListResponse(
    val items: List<ArticleJson>,
    val total: Long,
)

data class ArticleJson(
    val id: Long,
    val feed_id: Long,
    val guid: String,
    val title: String,
    val link: String,
    val content: String,
    val published_at: String? = null,
    val created_at: String? = null,
    val read: Boolean = false,
    val favorite: Boolean = false,
    val feed_title: String? = null,
    val ai_category: String? = null,
    val title_translated: String? = null,
    val content_translated: String? = null,
    val feed_ai_translate_enabled: Boolean? = null,
    val feed_ai_classify_enabled: Boolean? = null,
)

data class SummaryHistoryListResponse(
    val items: List<SummaryHistoryItemJson>,
    val total: Long,
)

data class SummaryHistoryItemJson(
    val id: Long,
    val ai_model_id: Long,
    val ai_model_name: String,
    val summary_template_id: Long? = null,
    val summary_template_name: String = "",
    val start_time: String = "",
    val end_time: String = "",
    val page: Int = 1,
    val page_size: Int = 20,
    val order: String = "desc",
    val article_count: Int = 0,
    val total: Long = 0,
    val content: String = "",
    val error: String = "",
    val created_at: String? = null,
)
