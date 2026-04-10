import type { Article } from '../api/client';
import {
  articleCategoryForDisplay,
  articleTitleForDisplay,
  type ArticleDisplayLang,
} from '../utils/articleDisplayLang';

export default function ArticleList({
  articles,
  onOpen,
  selectedId,
  displayLang = 'original',
}: {
  articles: Article[];
  onOpen: (a: Article) => void;
  selectedId?: number | null;
  displayLang?: ArticleDisplayLang;
}) {
  return (
    <ul className="article-list">
      {articles.map((a) => {
        const cat = articleCategoryForDisplay(a, displayLang);
        const title = articleTitleForDisplay(a, displayLang);
        return (
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
              title={title}
              aria-current={selectedId === a.id ? 'true' : undefined}
            >
              {a.ai_process_status === 'pending' && (a.feed_ai_classify_enabled || a.feed_ai_translate_enabled) ? (
                <span className="article-ai-pending">处理中… </span>
              ) : null}
              {title}
            </button>
          </div>
          <div className="article-meta">
            {cat && (
              <span className="article-ai-category" title="AI 领域分类（如财经、军事）">
                {cat}
              </span>
            )}
            {a.feed_title && <span className="feed">{a.feed_title}</span>}
          </div>
        </li>
        );
      })}
    </ul>
  );
}

