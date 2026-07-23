## [0.3.1](https://github.com/nestoca/joy-operator/compare/v0.3.0...v0.3.1) (2026-07-23)


### Bug Fixes

* **PL-6299:** update yoke controller runtime ([c4b11d6](https://github.com/nestoca/joy-operator/commit/c4b11d6c483e300169028abed78f676b3b09e54f))



# [0.3.0](https://github.com/nestoca/joy-operator/compare/v0.2.0...v0.3.0) (2026-07-23)


### Bug Fixes

* **PL-6299:** add controller filters ([92cf282](https://github.com/nestoca/joy-operator/commit/92cf282ee99561a28cbf6b3041ee6a9569ecfed1))


### Features

* **PL-6299:** add owner references ([e81c9ec](https://github.com/nestoca/joy-operator/commit/e81c9ecce0f58a51483c2871950663d6645343e3))



# [0.2.0](https://github.com/nestoca/joy-operator/compare/v0.1.2...v0.2.0) (2026-07-22)


### Features

* **PL-6299:** only manage configured known env destinations ([93a9e99](https://github.com/nestoca/joy-operator/commit/93a9e99dc4f812dfe9a055bdb0714fe00ffb23ac))



## [0.1.2](https://github.com/nestoca/joy-operator/compare/v0.1.1...v0.1.2) (2026-07-21)


### Bug Fixes

* **PL-6299:** add streams to release reconciler ([8539507](https://github.com/nestoca/joy-operator/commit/853950754545fc3712e26d5c439ec2a4e901e2a1))



## [0.1.1](https://github.com/nestoca/joy-operator/compare/v0.1.0...v0.1.1) (2026-07-21)


### Bug Fixes

* **PL-6299:** add option to chart to not install crds ([eb97e4d](https://github.com/nestoca/joy-operator/commit/eb97e4d1f5274a9e0971d032d571dce116ec44a3))



# [0.1.0](https://github.com/nestoca/joy-operator/compare/v0.0.13...v0.1.0) (2026-07-21)


### Features

* **PL-6299:** update joy to latest ([2d5d12b](https://github.com/nestoca/joy-operator/commit/2d5d12b1a86f2369bd75d377ad588418ee468dae))



## [0.0.13](https://github.com/nestoca/joy-operator/compare/v0.0.12...v0.0.13) (2026-07-20)


### Bug Fixes

* **PL-6888:** add chart version to helm puller logs ([a37dd94](https://github.com/nestoca/joy-operator/commit/a37dd94016d75b53cf6f9677ec78d6da83062caf))
* **PL-6888:** generate different catalog app by service prefixed by service name ([9ca9119](https://github.com/nestoca/joy-operator/commit/9ca91194f33069dc5a91f7ecdd59b7eb2eeb788b))



## [0.0.12](https://github.com/nestoca/joy-operator/compare/v0.0.11...v0.0.12) (2026-07-17)


### Bug Fixes

* **PL-6888:** update release reconciler logic ([49a6e30](https://github.com/nestoca/joy-operator/commit/49a6e30e869b7fed9ac76eb8122a4fda6f0a65a7))



## [0.0.11](https://github.com/nestoca/joy-operator/compare/v0.0.10...v0.0.11) (2026-07-17)


### Bug Fixes

* **PL-6888:** support concurrent chart pulls ([a384e65](https://github.com/nestoca/joy-operator/commit/a384e6502ac0b4097098839ad327f36655783217))



## [0.0.10](https://github.com/nestoca/joy-operator/compare/v0.0.9...v0.0.10) (2026-07-17)


### Bug Fixes

* **PL-6888:** use recurse for catalog app of apps ([5dafe0e](https://github.com/nestoca/joy-operator/commit/5dafe0e4c6c9459ecaf83a267eca62550b97fed7))



## [0.0.9](https://github.com/nestoca/joy-operator/compare/v0.0.8...v0.0.9) (2026-07-17)


### Bug Fixes

* **PL-6888:** argocd struct tags and force conflicts ([0797c6c](https://github.com/nestoca/joy-operator/commit/0797c6c8ddf67f3b18c84e73c0a08710f9c4deee))
* **PL-6888:** fix syncOptions serialization ([e94237f](https://github.com/nestoca/joy-operator/commit/e94237f42d0f19db34df04505b136f0a3b267558))



## [0.0.8](https://github.com/nestoca/joy-operator/compare/v0.0.7...v0.0.8) (2026-07-17)


### Bug Fixes

* **PL-6888:** fix struct tags for applicationSource ([86d28f8](https://github.com/nestoca/joy-operator/commit/86d28f85e985d19e6c3d99014416791ef8bce9d1))



## [0.0.7](https://github.com/nestoca/joy-operator/compare/v0.0.6...v0.0.7) (2026-07-17)


### Bug Fixes

* **PL-6888:** update self destination url for kubernetes ([160ffae](https://github.com/nestoca/joy-operator/commit/160ffae69ad07657964114c516ccf6fcabd4a4dd))



## [0.0.6](https://github.com/nestoca/joy-operator/compare/v0.0.5...v0.0.6) (2026-07-17)


### Bug Fixes

* **PL-6888:** fix chart common labels ([3f27543](https://github.com/nestoca/joy-operator/commit/3f275436afa061cd669d2ecea7c0a19e3c1487fd))



## [0.0.5](https://github.com/nestoca/joy-operator/compare/v0.0.4...v0.0.5) (2026-07-17)


### Bug Fixes

* **PL-6888:** fix destination server to https protocol ([a990c74](https://github.com/nestoca/joy-operator/commit/a990c7444e5c5cbad0779e113540fd1d5adb758d))



## [0.0.4](https://github.com/nestoca/joy-operator/compare/v0.0.3...v0.0.4) (2026-07-17)


### Bug Fixes

* **PL-6888:** add application source path and remove erroneous rbac ([638e843](https://github.com/nestoca/joy-operator/commit/638e84375e0f3f9361ab5918515c32e6237190be))



## [0.0.3](https://github.com/nestoca/joy-operator/compare/v0.0.2...v0.0.3) (2026-07-17)


### Bug Fixes

* **PL-6888:** fix sync policy automated to match expected logic ([3d83281](https://github.com/nestoca/joy-operator/commit/3d83281d59505b83d04e4298506f639874000cea))



## [0.0.2](https://github.com/nestoca/joy-operator/compare/v0.0.1...v0.0.2) (2026-07-16)


### Bug Fixes

* **PL-6290:** explicitly push tags ([52b18a9](https://github.com/nestoca/joy-operator/commit/52b18a9e846b09630ae4eb880fd8785fa79a7a7c))



## 0.0.1 (2026-07-16)


### Bug Fixes

* **PL-6287:** wire up catalog reconciler funcs ([f0e4430](https://github.com/nestoca/joy-operator/commit/f0e4430983b69345d5b36dafc42e58b826095b34))
* **PL-6290:** add empty changelog file ([ca47d87](https://github.com/nestoca/joy-operator/commit/ca47d87c66e54f439751db830548eac1b75ec987))
* **PL-6290:** change changelog fallback version ([5951a9a](https://github.com/nestoca/joy-operator/commit/5951a9ab96b2a052dc714077449fe3f49618fe4c))
* **PL-6290:** increase timeout on deletions in tests ([58b6ad1](https://github.com/nestoca/joy-operator/commit/58b6ad141cf428ac7510bc828469510a18276aaa))
* **PL-6290:** use allowed tg of conventional changelog action ([c9ac66a](https://github.com/nestoca/joy-operator/commit/c9ac66a9d79321a9a97d696ca1bcf48047599cd2))


### Features

* **PL-6131:** configure operator concurrency through envvar ([aeb8fb6](https://github.com/nestoca/joy-operator/commit/aeb8fb6df529f9d2ca2865f8e3604dd8f83013a4))
* **PL-6131:** implement joy operator poc ([51193b3](https://github.com/nestoca/joy-operator/commit/51193b37ce82d967cc66a04dde7facb5c7a9a8ad))
* **PL-6286:** implement app of apps pattern ([c400d64](https://github.com/nestoca/joy-operator/commit/c400d64c258006a1cae5d6e021f05eb5e1803978))
* **PL-6286:** use new catalog format ([24c8522](https://github.com/nestoca/joy-operator/commit/24c8522e85b56d9474838ac6aa9edec628c37f29))
* **PL-6287:** add crd-gen task ([da14bc8](https://github.com/nestoca/joy-operator/commit/da14bc8542d42db636724d285b8847a8dac9585b))
* **PL-6287:** add rbac permissions for catalogs ([99724fc](https://github.com/nestoca/joy-operator/commit/99724fcc12de700d94f19aa40d1a74ce2e70d151))
* **PL-6287:** integrate catalog api ([abaf7f4](https://github.com/nestoca/joy-operator/commit/abaf7f44fdaf116697834cb4d9fe5f24e3188f80))
* **PL-6287:** only reconcile known catalog ([a8a3c3d](https://github.com/nestoca/joy-operator/commit/a8a3c3dea4aebbe75071b230d9ad6097ae097819))
* **PL-6287:** update to latest joy ([b7b9a0e](https://github.com/nestoca/joy-operator/commit/b7b9a0e8f2c0eba1d9f64372069a7485d8cbad09))
* **PL-6289:** add initial e2e tests ([88e8f7a](https://github.com/nestoca/joy-operator/commit/88e8f7a46d714f0fc47a65fdd0214df08ef4725d))
* **PL-6289:** add testchart tests ([24e0c70](https://github.com/nestoca/joy-operator/commit/24e0c704bc7ecb6f0e905a8229f79cdd6585107f))
* **PL-6290:** add ci to build and publish ([295fe75](https://github.com/nestoca/joy-operator/commit/295fe75fcc2e4597ce31c156ea4d28460240a476))
* **PL-6290:** trigger ci ([9fb522c](https://github.com/nestoca/joy-operator/commit/9fb522ca61c25dbb09fed19e6d4135e4b939f39c))
* **PL-6792:** configurable env pattern ([195f5df](https://github.com/nestoca/joy-operator/commit/195f5df047448952a5076da9bf7860392caab273))



