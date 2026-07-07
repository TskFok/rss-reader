import { useSyncExternalStore } from 'react';
import { aiModelsApi, type AIModel } from '../api/client';

type AiModelsState = {
  models: AIModel[];
  loading: boolean;
  loaded: boolean;
};

let state: AiModelsState = { models: [], loading: false, loaded: false };
let inflight: Promise<AIModel[]> | null = null;
const listeners = new Set<() => void>();

function emit() {
  listeners.forEach((l) => l());
}

function setState(next: Partial<AiModelsState>) {
  state = { ...state, ...next };
  emit();
}

function subscribe(cb: () => void) {
  listeners.add(cb);
  return () => listeners.delete(cb);
}

function getSnapshot() {
  return state;
}

/** 登出时清空缓存 */
export function clearAiModelsCache() {
  state = { models: [], loading: false, loaded: false };
  inflight = null;
  emit();
}

/** 订阅列表等场景已拉取模型列表时写入缓存，避免重复请求 */
export function setAiModelsCache(models: AIModel[]) {
  state = { models, loading: false, loaded: true };
  inflight = null;
  emit();
}

/** 测试用：读取模块级状态 */
export function getAiModelsCacheForTest() {
  return state;
}

async function fetchAndCache(): Promise<AIModel[]> {
  setState({ loading: true });
  try {
    const { data } = await aiModelsApi.list();
    setState({ models: data, loading: false, loaded: true });
    return data;
  } catch {
    setState({ models: [], loading: false, loaded: true });
    return [];
  }
}

/** 已缓存则直接返回，否则请求一次并缓存 */
export function ensureAiModelsLoaded(): Promise<AIModel[]> {
  if (state.loaded && !state.loading) return Promise.resolve(state.models);
  if (inflight) return inflight;
  inflight = fetchAndCache().finally(() => {
    inflight = null;
  });
  return inflight;
}

/** 强制重新拉取（AI 模型增删改后） */
export function refreshAiModelsCache(): Promise<AIModel[]> {
  inflight = fetchAndCache().finally(() => {
    inflight = null;
  });
  return inflight;
}

export function useAiModels() {
  return useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
}
