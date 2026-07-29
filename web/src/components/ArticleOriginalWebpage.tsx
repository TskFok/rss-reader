export default function ArticleOriginalWebpage({ url }: { url: string }) {
  return (
    <section className="article-original-webpage" aria-label="原始网页">
      <iframe
        className="article-original-webpage-frame"
        src={url}
        aria-label="原始网页"
        loading="lazy"
        referrerPolicy="strict-origin-when-cross-origin"
      />
      <p className="article-original-webpage-fallback">
        如果网页无法嵌入，请{' '}
        <a href={url} target="_blank" rel="noopener noreferrer">
          在新标签页打开原文
        </a>
        。
      </p>
    </section>
  );
}
