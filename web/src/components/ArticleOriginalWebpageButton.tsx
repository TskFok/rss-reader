export default function ArticleOriginalWebpageButton({
  showOriginalWebpage,
  onClick,
}: {
  showOriginalWebpage: boolean;
  onClick: () => void;
}) {
  const label = showOriginalWebpage ? '返回正文' : '查看原始网页';

  return (
    <button
      type="button"
      className="article-detail-original-webpage-trigger"
      onClick={onClick}
      aria-label={label}
      title={label}
    >
      {showOriginalWebpage ? (
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <path d="m10 7-5 5 5 5" />
          <path d="M5 12h10a4 4 0 0 1 4 4v1" />
        </svg>
      ) : (
        <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
          <rect x="4" y="4" width="16" height="16" rx="2" />
          <path d="M8 8h8M8 12h8M8 16h5" />
        </svg>
      )}
    </button>
  );
}
