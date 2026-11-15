import { onUnmounted, ref } from 'vue'

// 组件 setup 时读一次，再用 change 跟上旋转和分屏。本项目是纯浏览器 SPA。
export function useMediaQuery(query) {
  const mql = window.matchMedia(query)
  const matches = ref(mql.matches)
  const onChange = () => { matches.value = mql.matches }
  mql.addEventListener('change', onChange)
  onUnmounted(() => mql.removeEventListener('change', onChange))
  return matches
}
