import { render, screen, waitFor } from '@testing-library/react';
import ArticleDetailContent from './ArticleDetailContent';
import type { Article } from '../api/client';

function makeArticle(overrides: Partial<Article> = {}): Article {
  return {
    id: 1,
    feed_id: 1,
    guid: 'g',
    title: '标题1',
    link: 'https://example.com/a',
    content: '<p>内容</p>',
    published_at: null,
    created_at: '2026-01-01T00:00:00Z',
    read: false,
    feed_title: 'Feed',
    ...overrides,
  };
}

test('为正文中的 video 标签补上播放控件', async () => {
  render(
    <ArticleDetailContent
      article={makeArticle({ content: '<video src="https://cdn.example.com/demo.mp4"></video>' })}
      displayLang="original"
    />
  );

  await waitFor(() => {
    const video = document.querySelector('video');
    expect(video).toBeInTheDocument();
    expect(video).toHaveAttribute('controls');
  });
});

test('将直链视频链接增强为可播放播放器', async () => {
  render(
    <ArticleDetailContent
      article={makeArticle({
        content: '<p><a href="https://cdn.example.com/demo.mp4">视频附件</a></p>',
      })}
      displayLang="original"
    />
  );

  await waitFor(() => {
    const video = document.querySelector('video');
    expect(video).toBeInTheDocument();
    expect(video).toHaveAttribute('src', 'https://cdn.example.com/demo.mp4');
    expect(screen.getByRole('link', { name: '视频附件' })).toBeInTheDocument();
  });
});

test('规范化 iframe 播放器并保留可播放能力', async () => {
  render(
    <ArticleDetailContent
      article={makeArticle({
        content:
          '<iframe src="https://player.bilibili.com/player.html?aid=116405710097058&amp;bvid=BV1PJQLBjEjf&amp;cid=37513658959&amp;p=1&amp;autoplay=0 scrolling=" frameborder="no" allowfullscreen="true" width="640" height="480"></iframe>',
      })}
      displayLang="original"
    />
  );

  await waitFor(() => {
    const iframe = document.querySelector('iframe');
    expect(iframe).toBeInTheDocument();
    expect(iframe).toHaveAttribute(
      'src',
      'https://player.bilibili.com/player.html?aid=116405710097058&bvid=BV1PJQLBjEjf&cid=37513658959&p=1&autoplay=0'
    );
    expect(iframe).toHaveAttribute('allowfullscreen', 'true');
    expect(iframe).toHaveAttribute('loading', 'lazy');
    expect(iframe).toHaveAttribute('title', '嵌入视频播放器');
  });
});

test('译文模式仍然展示纯文本内容', () => {
  render(
    <ArticleDetailContent
      article={makeArticle({
        content: '<video src="https://cdn.example.com/demo.mp4"></video>',
        content_translated: '这是一段译文',
      })}
      displayLang="translated"
    />
  );

  expect(screen.getByText('这是一段译文')).toBeInTheDocument();
  expect(document.querySelector('video')).not.toBeInTheDocument();
});
