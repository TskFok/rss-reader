# Android 示例：Retrofit API 定义

本目录为 **可直接拷贝到 Android Studio 工程** 的接口与数据类示例，依赖需在本机 `build.gradle` 中自行添加，例如：

```gradle
implementation "com.squareup.retrofit2:retrofit:2.9.0"
implementation "com.squareup.retrofit2:converter-moshi:2.9.0" // 或 converter-gson
implementation "com.squareup.okhttp3:logging-interceptor:4.12.0"
```

初始化示例：

```kotlin
val client = OkHttpClient.Builder()
    .addInterceptor { chain ->
        val token = userToken // 从 DataStore/EncryptedSharedPreferences 读取
        val req = chain.request().newBuilder()
        if (!token.isNullOrBlank()) {
            req.header("Authorization", "Bearer $token")
        }
        chain.proceed(req.build())
    }
    .build()

val api = Retrofit.Builder()
    .baseUrl("https://your-host.com/api/") // 必须以 / 结尾
    .client(client)
    .addConverterFactory(MoshiConverterFactory.create())
    .build()
    .create(RssReaderApi::class.java)
```

更完整的 REST 说明见上级目录 `../ANDROID_REST_API.md` 与 `../openapi.yaml`。
