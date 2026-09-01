# Navidrome Music Room

[English](README.md)

> 当前状态：开发预览版。网关、`.ndp` 桥接、签名更新器、API 合同和 MusicMate 第一阶段兼容层已经建立，但在公开到互联网之前仍应完整阅读安全文档并完成 Beta 验收。

Navidrome Music Room 把自托管 Navidrome 曲库变成 MusicMate 的同步听歌房。管理员创建房间并分享链接或二维码，受邀的 Navidrome 用户使用自己的账号加入；音频、封面和歌词始终由客户端直接向 Navidrome 请求，不经过房间网关代理。

![MusicMate 房间界面预览](docs/assets/musicmate-demo.gif)

预览沿用现有 MusicMate 房间结构。聊天、贴纸、VIP、统计、排行榜和成就入口保留但处于 License 锁定状态。

## 为什么需要伴生网关

Navidrome v0.63.2 的 `.ndp` 是沙箱化 WebAssembly 插件，目前没有自定义 Web 页面、入站 HTTP/WebSocket 路由或播放器控制扩展点，也不能自行监听端口。因此本项目采用官方 `.ndp` 加本地 Go 网关：

```text
Navidrome 插件设置与用户授权
          │
          ▼
navidrome-music-room.ndp ──30 秒用户/管理员心跳──▶ 本地房间网关
                                                        ▲
                           MusicMate REST/WebSocket ─────┘
                           音频/封面/歌词 ───────────────▶ Navidrome
```

Navidrome Web 首版只负责安装、配置、授权和启停，完整房间体验在 MusicMate。详见[官方插件文档](https://www.navidrome.org/docs/usage/features/plugins/)和[架构说明](docs/ARCHITECTURE.md)。

## 免费与锁定功能

| 功能 | 首版 |
|---|---|
| 房间 CRUD、关闭/重开 | 免费 |
| 可撤销邀请、持久成员、成员移除 | 免费 |
| 在线状态、同步播放、队列、历史 | 免费 |
| Navidrome 搜索、收藏、歌单、封面、歌词、转码、音频流 | 复用每个用户自己的 OpenSubsonic 权限 |
| 分享链接、Deep Link、App 本地二维码 | 免费 |
| 聊天、贴纸、VIP、统计、排行、等级、成就 | `402 feature_locked` |
| 上传、第三方在线源、语音点歌、公共输出设备 | 暂不迁移 |

## Docker Compose 快速安装

准备 Linux amd64/arm64、Docker Compose v2、两个 HTTPS 域名，以及 Navidrome 与网关都可写的插件目录。

```bash
git clone https://github.com/mythezone/navidrome-music-room.git
cd navidrome-music-room
cp .env.example .env
```

编辑 `.env`：

```dotenv
NAVIDROME_PUBLIC_URL=https://music.example.com
MUSIC_ROOM_PUBLIC_URL=https://rooms.example.com
MUSIC_ROOM_PLUGIN_PAIRING_TOKEN=<至少 32 字符的随机密钥>
```

当前开发预览默认跟随已签名的 `beta` 容器标签；稳定版发布后默认频道会切换到 `latest`。生产环境建议始终固定具体版本。

启动：

```bash
mkdir -p data/navidrome data/plugins/navidrome-music-room/room-data music
chown -R 1000:1000 data
docker compose up -d
```

也可以使用 `deploy/compose/install.sh` 创建目录和随机配对密钥。随后在 Navidrome 的 **设置 → 插件** 中打开 Navidrome Music Room，填写内外部地址和配对密钥，明确选择插件可访问的用户，再启用插件。

两个服务都应置于 TLS 反向代理之后；[Nginx 示例](deploy/nginx.conf.example)已经包含 WebSocket 转发。不要把明文 HTTP 网关暴露到公网。

## 权限与邀请流程

1. App 在本地生成 OpenSubsonic salt 和 `md5(password + salt)`；原始密码只留在 Keychain/Credential Store。
2. 网关只校验自己配置的 Navidrome 地址，不接受客户端传入上游 URL。
3. 创建房间必须同时满足 Navidrome `adminRole=true` 和插件用户授权。
4. 普通用户先兑换邀请；成功后成为持久成员，之后登录即可直接重进。
5. 邀请默认 7 天、20 次，可设置单次、期限和次数，也可撤销。数据库只保存 256 位随机密钥的 SHA-256。
6. 邀请被撤销不会移除已加入成员；管理员可以单独移除成员并立即断开其房间 WebSocket。
7. 用户加入和点歌时都会检查 music folder 权限，绝不会借用房主账号绕过 Navidrome ACL。

分享链接把邀请放在 fragment 中，避免进入代理日志和 Referer：

```text
https://rooms.example.com/join/ROOM_ID#invite=SECRET
musicmate://join?server=...&gateway=...&room=...&invite=...
```

## 数据与隐私

房间数据完全独立于 Navidrome 数据表：

```text
${Plugins.Folder}/navidrome-music-room/room-data/
├── rooms.sqlite3
├── secrets/
├── backups/
├── releases/
└── logs/
```

SQLite 开启 WAL、外键和事务迁移；每次迁移与版本切换前生成一致性备份。敏感文件权限为 `0600`，目录为 `0700`。默认不发送遥测；管理员可在 MusicMate 中主动生成自动脱敏的 JSON 诊断包，内容仅包含汇总健康信息。卸载插件或容器不会删除 `room-data`，只有显式清理才会移除。

## 更新与回滚

管理员 API 支持检查、暂存、安装和回滚。更新器只读取 GitHub Releases，不执行 `git pull`；下载后先验证已签名的校验清单与 SHA-256，并离线校验归档、SPDX SBOM 与绑定归档摘要/仓库/Tag/工作流的 in-toto/SLSA 来源证明签名，再检查归档路径和必需文件。稳定 launcher 会在激活前复核解包文件摘要、备份数据库并原子替换网关和 `.ndp`；健康检查失败会恢复旧二进制、旧插件和旧数据库备份。

只有当上一版二进制、插件和升级前 SQLite 备份能被验证为同一次切换时，管理页才会启用“回滚”。手动回滚会同时恢复这三项，并保留向前恢复点，直到旧网关通过健康检查。

GPL 核心不包含可被客户端绕过的商业功能实现，只提供完全离线的 Ed25519 License 验证边界；签名声明不能解锁并未随核心发布的代码，格式见 [License 文件说明](docs/LICENSE_FILES.md)。

Navidrome 检测到 `.ndp` 变化后会禁用插件。MusicMate 管理员页面已经实现一键续接：JWT 只保存在本次流程内存中，经 v0.63.2 版本适配器完成 rescan/enable，并同时确认网关版本和插件版本心跳。详见 [UPDATES.md](docs/UPDATES.md)。

## 兼容与开发

- Navidrome 最低版本：v0.63.2；CI 同时覆盖最新稳定版。
- 服务端：Linux amd64/arm64，首版单实例。
- FAIO 房间继续由 `FAIORoomProvider` 提供；首版不导入 FAIO 历史数据。
- API：[`contracts/openapi.yaml`](contracts/openapi.yaml)；兼容 fixtures 位于 `contracts/fixtures/`。

```bash
make test
make plugin
make build
```

常见故障与完整安全说明请阅读英文 README、[SECURITY.md](SECURITY.md)、[安全模型](docs/SECURITY_MODEL.md)和[贡献指南](CONTRIBUTING.md)。社区核心采用 GPL-3.0-only。
