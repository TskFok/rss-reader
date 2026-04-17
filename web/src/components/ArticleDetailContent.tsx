import { useEffect, useRef } from 'react';
import type { Article } from '../api/client';
import type { ArticleDisplayLang } from '../utils/articleDisplayLang';

const DIRECT_VIDEO_EXT_RE = /\.(mp4|webm|ogg|ogv|mov|m4v)(?:[?#].*)?$/i;
const EMBED_ALLOW =
  'accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share; fullscreen';
const HTML_LIKE_RE = /<\s*[a-z!/][^>]*>/i;

function normalizeIframeSrc(src: string) {
  const trimmed = src.trim();
  const whitespaceIndex = trimmed.search(/\s/);
  if (whitespaceIndex === -1) return trimmed;
  return trimmed.slice(0, whitespaceIndex);
}

function enhanceMedia(container: HTMLDivElement) {
  container.querySelectorAll('video').forEach((node) => {
    node.controls = true;
    node.preload = 'metadata';
    node.playsInline = true;
  });

  container.querySelectorAll('audio').forEach((node) => {
    node.controls = true;
    node.preload = 'metadata';
  });

  container.querySelectorAll<HTMLAnchorElement>('a[href]').forEach((anchor) => {
    const href = anchor.getAttribute('href')?.trim();
    if (!href || !DIRECT_VIDEO_EXT_RE.test(href)) return;
    if (anchor.dataset.videoEnhanced === 'true') return;
    if (anchor.nextElementSibling instanceof HTMLVideoElement) return;

    const video = document.createElement('video');
    video.src = href;
    video.controls = true;
    video.preload = 'metadata';
    video.playsInline = true;
    video.className = 'article-detail-inline-video';

    anchor.dataset.videoEnhanced = 'true';
    anchor.insertAdjacentElement('afterend', video);
  });

  container.querySelectorAll('iframe').forEach((node) => {
    const rawSrc = node.getAttribute('src')?.trim();
    if (rawSrc) {
      node.setAttribute('src', normalizeIframeSrc(rawSrc));
    }
    node.setAttribute('loading', 'lazy');
    node.setAttribute('allow', node.getAttribute('allow') || EMBED_ALLOW);
    node.setAttribute('referrerpolicy', node.getAttribute('referrerpolicy') || 'strict-origin-when-cross-origin');
    node.setAttribute('allowfullscreen', 'true');
    if (!node.getAttribute('title')) {
      node.setAttribute('title', '嵌入视频播放器');
    }
  });
}

function looksLikeHTML(content?: string) {
  return !!content && HTML_LIKE_RE.test(content);
}

export default function ArticleDetailContent({
  article,
  displayLang,
}: {
  article: Article;
  displayLang: ArticleDisplayLang;
}) {
  const contentRef = useRef<HTMLDivElement>(null);
  const translatedContent = article.content_translated?.trim() || '';
  const translatedIsHTML = looksLikeHTML(translatedContent);

  useEffect(() => {
    if (!contentRef.current) return;
    if (displayLang === 'translated' && translatedContent && !translatedIsHTML) return;
    enhanceMedia(contentRef.current);
  }, [article.content, translatedContent, translatedIsHTML, displayLang]);

  if (displayLang === 'translated' && translatedContent) {
    if (!translatedIsHTML) {
      return <div className="article-detail-content article-detail-plain">{translatedContent}</div>;
    }
    return (
      <div
        ref={contentRef}
        className="article-detail-content"
        dangerouslySetInnerHTML={{ __html: translatedContent }}
      />
    );
  }

  return (
    <div
      ref={contentRef}
      className="article-detail-content"
      dangerouslySetInnerHTML={{ __html: article.content || '<p>(暂无内容)</p>' }}
    />
  );
}
