/**
 * 界面缩放：让启动器在高 DPI / 小屏机器上不至于"塞不下"。
 *
 * 为什么需要它：wails 在 Windows 上只调用了 SetProcessDPIAware（系统级 DPI 感知，
 * 见 wails 的 pkg/application/init_windows.go），因此 main.go 里给的 Width/MinWidth
 * 会被当成**物理像素**。而整套 UI 是按 CSS 像素设计的——默认窗口 760px 在 200%
 * 缩放的屏幕上只等于 380 CSS 像素，内容被挤进半个窗口，观感就是"UI 特别大"。
 *
 * 这里不去动窗口尺寸（那会改变所有人的窗口行为），而是反过来把界面整体缩放：
 * CSS zoom 会改变布局视口，所以调小之后能放下更多内容，正是用户要的效果。
 * zoom 是 Chromium/WebKit 都支持的属性，且是**实时生效**，不需要重启。
 */

/** 配置键；值为 "auto" 或百分数字符串（如 "85"） */
export const UI_SCALE_KEY = "launcherUiScale";

/**
 * 自动比例的基准宽度（CSS 像素）。低于它就把界面整体调小。
 * 取 720 = main.go 里声明的 MinWidth：那是这套 UI 被设计成能正常排布的最小宽度，
 * 视口窄于它就意味着内容挤不下。
 */
export const UI_SCALE_DESIGN_WIDTH = 720;

/** 手动档的取值范围（50% ~ 150%） */
export const UI_SCALE_MIN = 0.5;
export const UI_SCALE_MAX = 1.5;

/**
 * 自动档的下限。自动只负责"挤不下时调小"，但再挤也不能小到看不清字，
 * 到这条线为止，剩下的交给用户手动档。
 */
export const UI_SCALE_AUTO_FLOOR = 0.6;

/** 缩放值收敛到可手动指定的范围 */
export function clampUiScale(value: number): number {
  if (!Number.isFinite(value)) return 1;

  return Math.min(UI_SCALE_MAX, Math.max(UI_SCALE_MIN, value));
}

/**
 * 量出"100% 缩放时"的视口宽度。
 *
 * 不能直接读 window.innerWidth：CSS zoom 会改变布局视口，边缩放边量会形成
 * 反馈环（缩放→视口变→重算缩放→……）。这里先把 zoom 摘掉量一次再还原——
 * 读 clientWidth 会强制一次同步布局，所以量到的是准确值；还原发生在同一次
 * 任务内，浏览器还没到下一次绘制，因此不会闪。
 */
export function measureBaseViewportWidth(): number {
  const root = document.documentElement;
  const previous = root.style.zoom;

  root.style.zoom = "";
  const width = root.clientWidth;

  root.style.zoom = previous;

  return width;
}

/**
 * 自动比例：视口越窄越小，视口够宽时保持 1。
 * 自动档只缩小、不放大——放大交给用户手动档，免得在大屏上把界面莫名撑大。
 */
export function computeAutoScale(baseViewportWidth: number): number {
  if (!(baseViewportWidth > 0)) return 1;
  const raw = baseViewportWidth / UI_SCALE_DESIGN_WIDTH;

  return Math.min(1, Math.max(UI_SCALE_AUTO_FLOOR, raw));
}

/**
 * 把缩放写到 <html> 上。zoom 会触发真正的重排（不像 transform 只是视觉缩放），
 * 所以面板宽度、换行、滚动都跟着一起变。1 时清掉属性，避免留一个无意义的
 * zoom:1 影响后续计算样式的判断。
 */
export function applyUiScale(scale: number): void {
  document.documentElement.style.zoom = scale === 1 ? "" : String(scale);
}

/** 配置值解析：空/损坏一律回落自动档 */
export function parseUiScale(raw: string): "auto" | number {
  const value = (raw ?? "").trim();

  if (!value || value === "auto") return "auto";

  const parsed = Number(value);

  if (!Number.isFinite(parsed) || parsed <= 0) return "auto";

  // 兼容两种写法：存百分比（"85"）或存比例（"0.85"）
  return clampUiScale(parsed > 10 ? parsed / 100 : parsed);
}

/** 缩放值 → 配置里存的字符串 */
export function serializeUiScale(scale: "auto" | number): string {
  if (scale === "auto") return "auto";

  return String(Math.round(scale * 100));
}
