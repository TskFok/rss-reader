import { type UiStyle, useTheme } from '../contexts/ThemeContext';

const UI_STYLE_OPTIONS: { value: UiStyle; label: string }[] = [
  { value: 'glass', label: '玻璃态' },
  { value: 'eink', label: '电子纸' },
  { value: 'kinetic', label: '动态排版' },
  { value: 'motion-driven', label: 'Motion-Driven' },
  { value: 'retro-futurism', label: '复古未来' },
  { value: 'hud-scifi-fui', label: 'HUD / Sci-Fi FUI' },
  { value: 'vibrant-block', label: 'Vibrant & Block-based' },
  { value: 'aurora', label: 'Aurora UI' },
  { value: 'aurora-evolved', label: 'Gradient Mesh / Aurora Evolved' },
  { value: 'memphis', label: 'Memphis Design' },
  { value: 'y2k', label: 'Y2K 美学' },
  { value: 'cyberpunk', label: '赛博朋克' },
  { value: 'pixel-art', label: '像素风 (Pixel Art)' },
];

type Props = {
  /** 顶栏内联样式；登录页等使用 fixed 右上角 */
  variant?: 'header' | 'floating';
};

export default function UiStyleControl({ variant = 'header' }: Props) {
  const { uiStyle, setUiStyle } = useTheme();

  const select = (
    <select
      id={variant === 'floating' ? 'ui-style-select-floating' : 'ui-style-select-header'}
      className={variant === 'floating' ? 'ui-style-select ui-style-select--floating' : 'ui-style-select'}
      value={uiStyle}
      onChange={(e) => setUiStyle(e.target.value as UiStyle)}
      aria-label="界面风格：玻璃态、电子纸、动态排版、Motion-Driven、复古未来、HUD / Sci-Fi FUI、Vibrant、Aurora UI、Gradient Mesh / Aurora Evolved、Memphis Design、Y2K 美学、赛博朋克或像素风"
      title="在多种界面风格间切换；Y2K 为千禧年镭射高光与气泡圆角；像素风为 8-bit 硬边与阶梯阴影"
    >
      {UI_STYLE_OPTIONS.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );

  if (variant === 'floating') {
    return (
      <div className="ui-style-floating-wrap">
        <label className="ui-style-floating-label" htmlFor="ui-style-select-floating">
          界面
        </label>
        {select}
      </div>
    );
  }

  return select;
}
