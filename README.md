# Git 增量打包工具 (Go)

基于 **Wails v2 + Go + React + TypeScript + Tailwind CSS** 的桌面端 Maven 增量打包工具。
针对**多模块 / Git 子群组（工作区）项目**优化：根据选中的 Git 提交，自动定位变更模块、计算依赖、仅构建受影响模块并收集产物 JAR。

> 由早期 Python 版重构为 Go 版，启动更快、单文件分发、无运行时依赖。

---

## ✨ 功能特性

- **提交驱动增量打包**：勾选若干 Git 提交 → 自动汇总变更文件 → 定位所属 Maven 模块 → 只构建这些模块。
- **智能依赖扩展**：开启后自动把被变更模块依赖的上游模块加入构建计划（`-am`），避免缺包。
- **Git 子群组 / 工作区支持**：一个根目录下含多个子 Git 仓库时，自动发现全部子仓，提交列表跨仓合并展示与选择。
- **分支管理**：列出 / 切换分支、切换后自动 `pull`，工作区脏时提示。
- **提交 Diff 预览**：单提交 / 单文件 diff 查看，大文件自动截断。
- **流式构建日志**：实时输出 Maven 日志、进度、单模块耗时，失败可中断。
- **打包耗时统计**：按模块统计构建耗时，定位瓶颈模块。
- **配置持久化**：项目根、输出目录、速度/范围档位、主题等保存在本地配置文件。

## 🏗 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22 + Wails v2 |
| 前端 | React 19 + TypeScript + Vite 6 |
| 样式 | Tailwind CSS 3 |
| 外部依赖 | 本机 `git`、`mvn`（需加入 PATH） |

## 📂 项目结构

```
maven-packager-go/
├── main.go                 # Wails 入口，窗口与资源装配
├── app.go                  # 对前端暴露的 API 与生命周期
├── wails.json              # Wails 构建配置
├── internal/
│   ├── project/            # 项目根定位、工作区单次扫描
│   ├── git/                # Git 命令封装、多仓工作区
│   ├── maven/              # 模块图解析、依赖扩展、构建编排
│   ├── timing/             # 构建耗时采集
│   ├── history/            # 打包历史记录
│   ├── procutil/           # 进程工具（隐藏子进程窗口等）
│   └── config/             # 配置读写
├── frontend/
│   ├── src/                # React 源码（App / components / hooks）
│   └── wailsjs/            # Wails 自动生成的绑定（勿手改）
└── build.bat               # Windows 一键打包脚本
```

## 🚀 快速开始

### 环境要求

- [Go](https://go.dev/dl/) ≥ 1.22
- [Node.js](https://nodejs.org/) ≥ 18（含 npm）
- [Wails CLI](https://wails.io/) v2：`go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`
- 本机已安装 `git` 与 `mvn`（Maven），并加入系统 PATH
- Windows 构建（WebView2）需 Windows 10/11

### 开发模式

```bash
wails dev
```

热重载：Go / 前端改动自动重新编译并刷新窗口。

### 打包 EXE

```bash
# 方式一：一键脚本（装依赖 + 前端构建 + Go 编译）
build.bat

# 方式二：直接用 Wails CLI
wails build -clean
```

产物：`build/bin/maven-packager-go.exe`（约 10 MB，单文件可直接分发）。

## 🧰 使用说明

1. 启动后程序自动探测项目根（优先 `.git`，其次 Maven 聚合根，再次含子仓/子 pom 的目录）。
   - 探测错误时点左上角「选择项目根目录」手动指定。
2. 顶部切换到目标分支（会自动拉取最新代码）。
3. 在提交列表勾选要纳入本次打包的提交。
4. 选择速度档位（快速 / 标准 / 兼容）与范围模式（稳妥 / 严格增量），按需开启「智能依赖」。
5. 设置输出目录，点击「开始打包」。
6. 实时查看日志与进度；完成后 JAR 汇总到输出目录，可在日志面板查看各模块耗时。

## ⚙️ 性能说明（启动优化）

针对 Git 子群组 / 多模块大仓，启动期的工作区扫描已合并为**单次遍历**，同时产出子仓列表、pom 存在性与模块数，避免首屏对同一目录反复全树 walk。跳过 `.git` / `target` / `node_modules` 等目录，且不进入深层构建残留。

如启动仍偏慢，可观察「转圈停留时长」判断瓶颈：转圈前白屏久 → WebView2 / 前端冷启动；转圈停顿久 → 项目根扫描。

## 📄 许可证

[MIT License](LICENSE)
