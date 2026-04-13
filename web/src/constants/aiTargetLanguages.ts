/** 与后端手动翻译、订阅翻译目标语言一致（存 ISO 风格代码供模型理解） */
export const AI_TARGET_LANGUAGES = [
  { code: 'zh-CN', label: '中文' },
  { code: 'en', label: '英语' },
  { code: 'fr', label: '法语' },
  { code: 'de', label: '德语' },
  { code: 'ar', label: '阿拉伯语' },
] as const;

export type AITargetLanguageCode = (typeof AI_TARGET_LANGUAGES)[number]['code'];

export function isKnownTargetLanguageCode(code: string): boolean {
  return AI_TARGET_LANGUAGES.some((x) => x.code === code);
}
