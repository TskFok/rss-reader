type Props = {
  source: string;
  className?: string;
};

/** AI 总结正文以纯文本展示（不解析 Markdown，`*`、`#` 等原样显示） */
export default function SummaryMarkdownContent({ source, className }: Props) {
  const rootClass = ['feeds-summary-plain', className].filter(Boolean).join(' ');
  return <div className={rootClass}>{source}</div>;
}
