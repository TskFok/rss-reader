import type { Article } from '../api/client';

export default function ArticleList({
  articles,
  onOpen,
  selectedId,
}: {
  articles: Article[];
  onOpen: (a: Article) => void;
  selectedId?: number | null;
}) {
  return (
    <ul className="article-list">
      {articles.map((a) => (
        <li
          key={a.id}
          data-article-id={a.id}
          className={[
            a.read ? 'read' : '',
            selectedId === a.id ? 'active' : '',
          ]
            .filter(Boolean)
            .join(' ')}
        >
          <div className="article-header">
            <button
              type="button"
              className="article-title-btn"
              onClick={() => onOpen(a)}
              title={a.title || '(无标题)'}
              aria-current={selectedId === a.id ? 'true' : undefined}
            >
              {a.title || '(无标题)'}
            </button>
          </div>
          <div className="article-meta">
            {a.feed_title && <span className="feed">{a.feed_title}</span>}
          </div>
        </li>
      ))}
    </ul>
  );
}

