## [1.10.3](https://github.com/asaidimu/hestia/compare/v1.10.2...v1.10.3) (2026-08-30)


### Bug Fixes

* **memory:** S-15 — audit buffer lifecycle: eager construction, flush on stop, bounded write ([0a80303](https://github.com/asaidimu/hestia/commit/0a80303dfe60a8cd0645b6fd7bd96c51232d14e6))
* **security:** S-16 — stop leaking internal error causes to HTTP clients ([1fa38d6](https://github.com/asaidimu/hestia/commit/1fa38d6321654e352ed7bbbe3bfbc7956fd6f86d))
* **security:** S-17 — clamp client-supplied pagination limits ([09498c6](https://github.com/asaidimu/hestia/commit/09498c6482122c0347a14f9bc0d901770f55627e))
* **security:** S-2 — replace hardcoded default session secret with provisioned per-boot secret ([732ee21](https://github.com/asaidimu/hestia/commit/732ee211628108f986dbe56f1cc76dbb927a7bfa))
* **security:** S-20 — API-key allowlist fails closed on type confusion ([91b216d](https://github.com/asaidimu/hestia/commit/91b216d25170679932c43306426e57438d2f55c3))
* **security:** S-21 — hardening grab-bag (docs:list, core:reset GET, SMTP TLS, rate-limit fail-open signal, silent CEL skips) ([9c3aee6](https://github.com/asaidimu/hestia/commit/9c3aee62718dc76cc9ca67a5bd1f19f046f3c5ff))
* **security:** S-4 + S-13 — real logout via token blocklist; single-use reset tokens ([5efb841](https://github.com/asaidimu/hestia/commit/5efb841d72280e7154cd04b5d57ca550861b8927))
* **security:** S-7 + S-8 — trusted-proxy client IP and fasthttp resource limits ([11fadc3](https://github.com/asaidimu/hestia/commit/11fadc3843ea9120db6f565bf12251bc879a957b))

## [1.10.2](https://github.com/asaidimu/hestia/compare/v1.10.1...v1.10.2) (2026-08-30)


### Bug Fixes

* **auth:** implement critical security patches from batch 1 audit ([7640a10](https://github.com/asaidimu/hestia/commit/7640a109a5e7b758e20d658250ca332f46a885d8))
* **client:** add system:updates:update:discard route to client SDK ([060a08d](https://github.com/asaidimu/hestia/commit/060a08d8422b770e56d38755e2fa051206ca27ef))

## [1.10.1](https://github.com/asaidimu/hestia/compare/v1.10.0...v1.10.1) (2026-08-29)


### Bug Fixes

* **notifications:** fix stream cleanup logic and add service logger ([907a823](https://github.com/asaidimu/hestia/commit/907a8232343931c664ad5a0a6f95b079187aa30b))

# [1.10.0](https://github.com/asaidimu/hestia/compare/v1.9.0...v1.10.0) (2026-08-29)


### Features

* **updates:** add discard mechanism for failed updates ([c773ca4](https://github.com/asaidimu/hestia/commit/c773ca4be6744d4213732fe8c3e167dca8184d1a))

# [1.9.0](https://github.com/asaidimu/hestia/compare/v1.8.2...v1.9.0) (2026-08-29)


### Features

* **workflows:** add update method for workflow definitions ([64ef6db](https://github.com/asaidimu/hestia/commit/64ef6db1301d97d4867ea8ea53d61ab747a0777f))

## [1.8.2](https://github.com/asaidimu/hestia/compare/v1.8.1...v1.8.2) (2026-08-29)


### Bug Fixes

* **workflows:** add real-time SSE streaming for workflow runs ([a4eb044](https://github.com/asaidimu/hestia/commit/a4eb0443e63617911066932e1972885b78ee51c8))

## [1.8.1](https://github.com/asaidimu/hestia/compare/v1.8.0...v1.8.1) (2026-08-29)


### Bug Fixes

* **http:** add global panic recovery middleware to transport ([8953953](https://github.com/asaidimu/hestia/commit/8953953b16c7eb6d8143a03550eb5e856420f88c))

# [1.8.0](https://github.com/asaidimu/hestia/compare/v1.7.0...v1.8.0) (2026-08-29)


### Features

* **workflows:** add registry and runtime API endpoints ([b76b7c7](https://github.com/asaidimu/hestia/commit/b76b7c7855eff32bea393eb50e595f5d95f8b491))

# [1.7.0](https://github.com/asaidimu/hestia/compare/v1.6.2...v1.7.0) (2026-08-28)


### Features

* **workflows:** add workflow engine integration ([75aa399](https://github.com/asaidimu/hestia/commit/75aa3991cf8280bac588799cfa91adaa86c20449))

## [1.6.2](https://github.com/asaidimu/hestia/compare/v1.6.1...v1.6.2) (2026-08-27)


### Bug Fixes

* **config:** remove environment variable support for application version ([15cfe7e](https://github.com/asaidimu/hestia/commit/15cfe7ee7ff4eb59fedaa5552090c39a9072097b))

## [1.6.1](https://github.com/asaidimu/hestia/compare/v1.6.0...v1.6.1) (2026-08-27)


### Bug Fixes

* **notifications:** move stream registration into standard notification module ([45d56ce](https://github.com/asaidimu/hestia/commit/45d56ce998a648a95083f2e6ca1a3035f6d3084c))

# [1.6.0](https://github.com/asaidimu/hestia/compare/v1.5.0...v1.6.0) (2026-08-27)


### Features

* **system:** add blob management features and system log auditing ([db3ac00](https://github.com/asaidimu/hestia/commit/db3ac0045c8db163b86bd1dcf7756b54a4a2792d))

# [1.5.0](https://github.com/asaidimu/hestia/compare/v1.4.19...v1.5.0) (2026-08-26)


### Features

* **core:** implement system logging and audit log streaming ([aa5023f](https://github.com/asaidimu/hestia/commit/aa5023f73e0b48d1fad7822d7335e48b0a18bf76))

## [1.4.19](https://github.com/asaidimu/hestia/compare/v1.4.18...v1.4.19) (2026-08-26)


### Bug Fixes

* **system:** migrate hand-rolled models to generated collections ([f5ed14c](https://github.com/asaidimu/hestia/commit/f5ed14cb7b192d2acaf1d12cff44d1b3dfc98a5e))

## [1.4.18](https://github.com/asaidimu/hestia/compare/v1.4.17...v1.4.18) (2026-08-25)


### Bug Fixes

* Commit message fix ([f583f2f](https://github.com/asaidimu/hestia/commit/f583f2f31c477a65aa1308b3d07e84ca8a80ba4e))

## [1.4.17](https://github.com/asaidimu/hestia/compare/v1.4.16...v1.4.17) (2026-08-23)


### Bug Fixes

* **core:** remove abstraction leak in Input struct and header_fields ([bcbe1e5](https://github.com/asaidimu/hestia/commit/bcbe1e5d990a346f60c27a9ca16a71c68689f20a))

## [1.4.16](https://github.com/asaidimu/hestia/compare/v1.4.15...v1.4.16) (2026-08-23)


### Bug Fixes

* **runtime:** implement async-native dispatcher with callback support ([c2dd46c](https://github.com/asaidimu/hestia/commit/c2dd46c138cc980e0655deb68e6768516f0ee189))

## [1.4.15](https://github.com/asaidimu/hestia/compare/v1.4.14...v1.4.15) (2026-08-22)


### Bug Fixes

* **updates:** split update check into availability and staging ([93d65cd](https://github.com/asaidimu/hestia/commit/93d65cd894bb7840c46e3218ae0f382d562d438d))

## [1.4.14](https://github.com/asaidimu/hestia/compare/v1.4.13...v1.4.14) (2026-08-21)


### Bug Fixes

* **updates:** add update management system ([8c85ff4](https://github.com/asaidimu/hestia/commit/8c85ff4719ba1133483349bd6d0f567554a83b35))

## [1.4.13](https://github.com/asaidimu/hestia/compare/v1.4.12...v1.4.13) (2026-08-19)


### Bug Fixes

* **core:** add StaticFS support to SetupConfig ([72c305e](https://github.com/asaidimu/hestia/commit/72c305ec2e99084b33f54469615d8adf558393f9))

## [1.4.12](https://github.com/asaidimu/hestia/compare/v1.4.11...v1.4.12) (2026-08-19)


### Bug Fixes

* **auth:** consolidate auth handlers and remove deprecated bootstrap logic ([8ae1133](https://github.com/asaidimu/hestia/commit/8ae1133013d7352503e06bd64126d0e279153665))

## [1.4.11](https://github.com/asaidimu/hestia/compare/v1.4.10...v1.4.11) (2026-08-19)


### Bug Fixes

* **boot:** surface ephemeral API key on first run ([c1cd346](https://github.com/asaidimu/hestia/commit/c1cd3465e24f0109e2895c1726f42344523ded0d))

## [1.4.10](https://github.com/asaidimu/hestia/compare/v1.4.9...v1.4.10) (2026-08-19)


### Bug Fixes

* **system:** add systemd-native self-update support ([778222b](https://github.com/asaidimu/hestia/commit/778222b273458788045c8c448c932c8d8860d06d))

## [1.4.9](https://github.com/asaidimu/hestia/compare/v1.4.8...v1.4.9) (2026-08-19)


### Bug Fixes

* **system:** add systemd-native self-update support ([16c7fcb](https://github.com/asaidimu/hestia/commit/16c7fcb225983f9de778d649b0ebb25800bd5d32))

## [1.4.8](https://github.com/asaidimu/hestia/compare/v1.4.7...v1.4.8) (2026-08-18)


### Bug Fixes

* **runtime:** add self-update service ([1e5462b](https://github.com/asaidimu/hestia/commit/1e5462ba0585027aa999fb6cf73e6e32fef92e8a))

## [1.4.7](https://github.com/asaidimu/hestia/compare/v1.4.6...v1.4.7) (2026-08-18)


### Bug Fixes

* **client:** Write end-to-end integration tests ([c6bcb97](https://github.com/asaidimu/hestia/commit/c6bcb977d94e2af5cad2a4d89cec0330ff79481e))

## [1.4.6](https://github.com/asaidimu/hestia/compare/v1.4.5...v1.4.6) (2026-08-17)


### Bug Fixes

* reorganize core features to system architecture ([295bd84](https://github.com/asaidimu/hestia/commit/295bd84502a10765f8bca0da7405677713dad27c))

## [1.4.5](https://github.com/asaidimu/hestia/compare/v1.4.4...v1.4.5) (2026-08-12)


### Bug Fixes

* **blobs:** implement resumable upload protocol ([ea3a9ff](https://github.com/asaidimu/hestia/commit/ea3a9ff5b5a9521d7c9bde612e62119297a4b47a))

## [1.4.4](https://github.com/asaidimu/hestia/compare/v1.4.3...v1.4.4) (2026-08-12)


### Bug Fixes

* **blobs:** add resumable upload protocol ([0e8581b](https://github.com/asaidimu/hestia/commit/0e8581b96ee64df6be97dd6a41dad7abb42e427e))

## [1.4.3](https://github.com/asaidimu/hestia/compare/v1.4.2...v1.4.3) (2026-08-12)


### Bug Fixes

* **client:** implement user creation in HestiaUsers store ([b84666b](https://github.com/asaidimu/hestia/commit/b84666b0e8a88b4c2e26797263685e5f47920e40))

## [1.4.2](https://github.com/asaidimu/hestia/compare/v1.4.1...v1.4.2) (2026-08-11)


### Bug Fixes

* **core:** migrate features to model-based architecture and update schema management ([542d9f9](https://github.com/asaidimu/hestia/commit/542d9f943c277bcb252c9da3c5d2d54ef55e6b9e))

## [1.4.1](https://github.com/asaidimu/hestia/compare/v1.4.0...v1.4.1) (2026-07-30)


### Bug Fixes

* start refactor of core system for more robust code ([45d8478](https://github.com/asaidimu/hestia/commit/45d8478c7c67fddde292ad25f46be9a69a75674a))

# [1.4.0](https://github.com/asaidimu/hestia/compare/v1.3.0...v1.4.0) (2026-07-28)


### Features

* **core:** refactor messaging and add notification/scheduler systems ([6b70dc9](https://github.com/asaidimu/hestia/commit/6b70dc928a8fdc33ece60be322f2fffbb404403e))

# [1.3.0](https://github.com/asaidimu/hestia/compare/v1.2.3...v1.3.0) (2026-07-25)


### Features

* **core:** refactor persistence and multi-tenancy support ([4783c52](https://github.com/asaidimu/hestia/commit/4783c524efbd3911c288ac16d449a8aefde9a21c))

## [1.2.3](https://github.com/asaidimu/hestia/compare/v1.2.2...v1.2.3) (2026-07-22)


### Bug Fixes

* fix heartbeats ([6a8d7a8](https://github.com/asaidimu/hestia/commit/6a8d7a85199741b52f5b95a1cab2e27cec921eba))

## [1.2.2](https://github.com/asaidimu/hestia/compare/v1.2.1...v1.2.2) (2026-07-22)


### Bug Fixes

* **core:** flatten SetupConfig and fix port type to int ([993e593](https://github.com/asaidimu/hestia/commit/993e59306bfc72e33034bc3163c6cbbedc451272))
* fighting with wails ([046b838](https://github.com/asaidimu/hestia/commit/046b838df25d791e09c02f5fd8ae6c133ae99943))

## [1.2.1](https://github.com/asaidimu/hestia/compare/v1.2.0...v1.2.1) (2026-07-22)


### Bug Fixes

* **client:** polyfill requestIdleCallback for Wails webkit2_41 compatibility ([3f7e77a](https://github.com/asaidimu/hestia/commit/3f7e77a57d9edc73c7c72dc2f987b8e1297c8469))

# [1.2.0](https://github.com/asaidimu/hestia/compare/v1.1.0...v1.2.0) (2026-07-21)


### Bug Fixes

* **policies:** improve rule validation API and update dependencies ([96a86b8](https://github.com/asaidimu/hestia/commit/96a86b8cce68c885dba82cb38bb5436cc97ba6a9))


### Features

* **core:** restructure core package and introduce Wails transport ([f5494b4](https://github.com/asaidimu/hestia/commit/f5494b426ed2e284f76e60b8c14c1e12d6eec84b))

# [1.1.0](https://github.com/asaidimu/hestia/compare/v1.0.7...v1.1.0) (2026-07-20)


### Features

* **core:** introduce blob namespace dynamic routing and heartbeat support ([aa9971f](https://github.com/asaidimu/hestia/commit/aa9971f4bd18ebe1063d00f24a83d3a965e2e6d2))

## [1.0.7](https://github.com/asaidimu/hestia/compare/v1.0.6...v1.0.7) (2026-07-20)


### Bug Fixes

* fix auth flow in client ([5abac6d](https://github.com/asaidimu/hestia/commit/5abac6d7839246a872317d399c3c4c2b74c7f60d))

## [1.0.6](https://github.com/asaidimu/hestia/compare/v1.0.5...v1.0.6) (2026-07-20)


### Bug Fixes

* fix auth flow ([96be606](https://github.com/asaidimu/hestia/commit/96be6062c8c668a8789f3629006942e000e1b206))

## [1.0.5](https://github.com/asaidimu/hestia/compare/v1.0.4...v1.0.5) (2026-07-19)


### Bug Fixes

* fix migrations ([f4d07c2](https://github.com/asaidimu/hestia/commit/f4d07c2097eaf7d4a12514a32a179b709f914c70))

## [1.0.4](https://github.com/asaidimu/hestia/compare/v1.0.3...v1.0.4) (2026-07-19)


### Bug Fixes

* **auth:** replace JWT bearer tokens with server-managed HTTP-only session cookies ([1b5583f](https://github.com/asaidimu/hestia/commit/1b5583f3c4bc20c5b3769ba1afd85c070b0db2b8))

## [1.0.3](https://github.com/asaidimu/hestia/compare/v1.0.2...v1.0.3) (2026-07-18)


### Bug Fixes

* fix policies ([cabef5a](https://github.com/asaidimu/hestia/commit/cabef5af5778ed79f04a60c9761728fb19ce2b91))

## [1.0.2](https://github.com/asaidimu/hestia/compare/v1.0.1...v1.0.2) (2026-07-18)


### Bug Fixes

* fix client bundling ([58bde5d](https://github.com/asaidimu/hestia/commit/58bde5d366ac6f90aa8c929c88abb9e51195d65f))

## [1.0.1](https://github.com/asaidimu/hestia/compare/v1.0.0...v1.0.1) (2026-07-18)


### Bug Fixes

* **auth,api,policies:** centralize credential/identity providers, migrate to IAM rule system, restructure API interface ([6d228f9](https://github.com/asaidimu/hestia/commit/6d228f9ec575699306562ed4c21375b33f17f403))

# 1.0.0 (2026-07-15)


### Bug Fixes

* add tests and remove local references from go.mod ([1e9305a](https://github.com/asaidimu/hestia/commit/1e9305a3cedea950f145e6933ec74cd2b64ba52b))


### Features

* initial commit ([29c08e2](https://github.com/asaidimu/hestia/commit/29c08e2a9b19851eb2f19fe55fee0ae84077d5f9))
