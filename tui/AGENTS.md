# AGENTS.md — tapass-tui

tapass 密码管理器 TUI 客户端。

## 构建 & 测试

```bash
go build ./...                                    # 编译
go test ./internal/model/ -v                     # model 测试
go vet ./...                                      # 静态检查
go run ./cmd/tapass-tui/                          # 运行
```

## 包结构

```
cmd/tapass-tui/main.go     # 入口
internal/
  store/                    # 存储抽象层
    store.go                # Store 接口
    local/local.go          # 本地文件存储
  model/                    # 内存数据模型
    tree.go                 # Node + BuildTree（Key路径→树形结构）
    tree_test.go
  tui/                      # Bubble Tea 视图层
    app.go                  # 主 Model + 窗口路由 + 消息类型
    welcome.go              # 欢迎/打开/新建数据库
    mainview.go             # 主界面布局
    sidebar.go              # 左侧分组树
    entrylist.go            # 右侧条目列表
    entrydetail.go          # 条目详情/编辑属性
    dbconfig.go             # 数据库设置（改密）
    dialog.go               # 新建条目对话框
    styles.go               # 共享样式
```

## 依赖说明

- tui 通过 `go.mod` 的 `replace` 指令引用 `tools/vault`：
  ```
  replace github.com/tapass/tapass-tools => ../tools
  ```
- vault 包路径为 `github.com/tapass/tapass-tools/vault`（非 internal，允许跨模块访问）
- vault 源码只在 tools 中维护，tui 不包含 vault 副本
- vault API 变更会直接影响 tui 编译

## 二进制格式关键约束

- 头部 144 字节明文 + 变长密文体
- Magic（ASCII）大端序，其余多字节字段小端序
- 密钥派生：Argon2id(32B) → HKDF-SHA256 → HMAC Key(32B) + Encrypt Key(32B)
- HKDF info：`tapass-v1-hmac` / `tapass-v1-enc`
- 加密：XChaCha20-Poly1305，压缩在加密前执行（flate 裸 DEFLATE，非 zlib）
- 头部校验：Header MAC = SHA256(header[0:80])，Header HMAC = HMAC-SHA256(MAC, HMAC_Key)
- 数据段：追加式 KV，同 key 取最新时间戳，Type=0 表示删除
- Key 路径以 `/` 分隔，特殊属性名：PASSWD / SSH / TOTP

## 实现约定

- Key 路径格式：`/group/subgroup/entry/ATTR_NAME`，最后一段为属性名
- Node.ID 是路径标识符兼显示名，无 Name 字段
- 分组规则：有子节点自动推断为组
- 压缩使用 flate 裸 DEFLATE 流（非 zlib），与设计文档 "zlib/DEFLATE" 描述不同，为本项目明确选择
- Node.Path 统一以 `/` 开头，与 vault key 前缀一致
- 新建条目只创建路径前缀，用户在详情页添加属性
- 属性编辑先统一文本，PASSWD/TOTP/SSH 专属控件后续迭代
- 存储扩展通过实现 `store.Store` 接口（先 local，后续 WebDAV）
- Store.Save 需传入 path 参数（vault 不持有文件路径）
- vault 包不操作文件系统，文件 I/O 由 store 实现负责（local 使用原子写入）
- TUI 组件均持有 width/height，通过 SetSize 响应 resize
- 共享样式定义在 `tui/styles.go`
