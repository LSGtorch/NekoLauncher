/**
 * 界面缩放的运行时控制器（模块级单例 + 订阅）。
 *
 * 为什么不挂进 BackgroundProvider：缩放必须**在窗口揭幕之前**就落定，否则用户会
 * 先看到 100% 的一帧再跳变，正好破坏背景层那套"首帧即终态"的努力；而缩放本身
 * 与背景无关。做成独立小 store，启动流程能 await 它，设置页也能实时跟随。
 *
 * 生效时机：zoom 是实时属性，改完立刻见效，无需重启。自动档还会跟随窗口尺寸
 * 变化重算。
 */
import { useCallback, useEffect, useReducer } from "react";

import { GetValue, SetValue } from "../../wailsjs/go/bindings/ConfigAPI";

import {
  UI_SCALE_KEY,
  applyUiScale,
  clampUiScale,
  computeAutoScale,
  measureBaseViewportWidth,
  parseUiScale,
  serializeUiScale,
} from "./ui-scale";

/** 档位："auto" = 自动按窗口宽度；数值 = 手动比例 */
export type UiScaleMode = "auto" | number;

let mode: UiScaleMode = "auto";
let effective = 1;
let ready = false;

const listeners = new Set<() => void>();

function notify(): void {
  listeners.forEach((listener) => listener());
}

/** 按当前档位重算并应用；自动档以"100% 缩放时的视口宽度"为输入 */
function sync(): void {
  effective =
    mode === "auto"
      ? computeAutoScale(measureBaseViewportWidth())
      : clampUiScale(mode);

  applyUiScale(effective);
  notify();
}

/** 自动档跟随窗口尺寸；手动档不响应，避免用户定好的比例被窗口拖动改掉 */
function onResize(): void {
  if (mode === "auto") sync();
}

/**
 * 启动时调用一次：读配置 → 应用 → 挂 resize 监听。
 * 返回 Promise，供揭幕流程等待（见 layouts/index.tsx 的 WindowReveal）。
 */
export async function initUiScale(): Promise<void> {
  try {
    mode = parseUiScale(await GetValue(UI_SCALE_KEY));
  } catch {
    mode = "auto"; // 读配置失败按默认自动档
  }

  sync();
  window.addEventListener("resize", onResize);
  ready = true;
  notify();
}

/** 设置页调用：先本地生效再落盘。落盘失败不回滚——本次运行内已生效，只是重启后回到旧值 */
export async function setUiScale(next: UiScaleMode): Promise<void> {
  mode = next;
  sync();

  try {
    await SetValue(UI_SCALE_KEY, serializeUiScale(next));
  } catch {
    /* 写配置失败：保持已生效的缩放，不回滚也不报错 */
  }
}

/** 订阅缩放状态：返回当前档位、实际生效比例与是否已就绪 */
export function useUiScale(): {
  mode: UiScaleMode;
  effective: number;
  ready: boolean;
} {
  const [, bump] = useReducer((n: number) => n + 1, 0);
  const subscribe = useCallback((listener: () => void) => {
    listeners.add(listener);

    return () => {
      listeners.delete(listener);
    };
  }, []);

  useEffect(() => subscribe(bump), [subscribe, bump]);

  return { mode, effective, ready };
}
