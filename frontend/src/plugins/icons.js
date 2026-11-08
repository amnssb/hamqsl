// 图标按需注册。
//
// @element-plus/icons-vue@2.3.2 一共导出 293 个图标组件。旧写法在 main.js 里
// `import * as ElementPlusIconsVue` 然后全量 app.component()，等于把 293 个 SFC
// 全部塞进入口 chunk —— 首屏体积里很大一块是这堆永远用不到的图标。
//
// 本项目实际只用到下面这些。新增图标时在这里补一行即可：
// 漏补不会静默出错，模板里的 <Xxx /> 会渲染成空白，很容易发现。
import {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  Back,
  Bell,
  BellFilled,
  Box,
  Check,
  CircleCheck,
  CircleClose,
  Connection,
  DataBoard,
  Document,
  Headset,
  House,
  InfoFilled,
  Loading,
  Location,
  Lock,
  MagicStick,
  Menu,
  Message,
  Notebook,
  OfficeBuilding,
  Plus,
  Postcard,
  Promotion,
  Right,
  Search,
  Switch,
  SwitchButton,
  Tools,
  UploadFilled,
  User,
  Van,
} from '@element-plus/icons-vue'

// 同时以对象形式导出：Dashboard 等处用 <component :is="..."> 传组件对象，
// 不再依赖"字符串 → 全局组件名"的隐式解析。
export const icons = {
  ArrowDown,
  ArrowLeft,
  ArrowRight,
  Back,
  Bell,
  BellFilled,
  Box,
  Check,
  CircleCheck,
  CircleClose,
  Connection,
  DataBoard,
  Document,
  Headset,
  House,
  InfoFilled,
  Loading,
  Location,
  Lock,
  MagicStick,
  Menu,
  Message,
  Notebook,
  OfficeBuilding,
  Plus,
  Postcard,
  Promotion,
  Right,
  Search,
  Switch,
  SwitchButton,
  Tools,
  UploadFilled,
  User,
  Van,
}

export function registerIcons(app) {
  for (const [name, component] of Object.entries(icons)) {
    app.component(name, component)
  }
}
