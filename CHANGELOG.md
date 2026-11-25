# 变更说明 / Changelog

## 首次提交 / Initial import

### 中文

- 将 SG500Z MQTT Demo 的后端从 Java 迁移到 Go，处理采集器注册、参数下发和采集数据上报，并提供 HTTP API。
- 沿用 Vue 前端，支持中英文切换、设备和测点建模、通道绑定，以及指标、波形、频谱、音频和设备上报查询。
- 支持采集器日志上传与下载、固件升级进度跟踪，以及重启后版本核对。
- 使用 SQLite 和本地文件保存配置与采集数据。README 说明了如何启动项目和迁移 Java 版数据。

### English

- Ported the SG500Z MQTT demo backend from Java to Go. It handles collector registration, configuration delivery, and data reports, and exposes an HTTP API.
- Kept the Vue frontend with Chinese and English language options. It supports device and measurement point modeling, channel bindings, and views for metrics, waveforms, spectra, audio, and device reports.
- Added collector log uploads and downloads, firmware upgrade progress tracking, and version checks after a restart.
- Used SQLite and local files to store configuration and acquisition data. The README covers setup and migration from the Java demo.
