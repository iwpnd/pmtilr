## [1.1.0](https://github.com/iwpnd/pmtilr/compare/v1.0.3...v1.1.0) (2026-04-27)

### ✨ Features

* ✨ revert require go1.26.2 ([f161402](https://github.com/iwpnd/pmtilr/commit/f161402f4e16d20718916d1752a49986dbe888f8))

### 🧹 Miscellaneous

* 🔧 explicit copy err ([28bb33e](https://github.com/iwpnd/pmtilr/commit/28bb33eef8a2e10b58287543000c4e996cf55c5d))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#86](https://github.com/iwpnd/pmtilr/issues/86)) ([4746e4f](https://github.com/iwpnd/pmtilr/commit/4746e4f9b510dcc1071dc2afaebfcc674059e867))

## [1.0.3](https://github.com/iwpnd/pmtilr/compare/v1.0.2...v1.0.3) (2026-04-12)

### 🐛 Bug Fixes

* 🐛 .Ext() on tilejson endpoint ([0cbd64e](https://github.com/iwpnd/pmtilr/commit/0cbd64ed3ae6acf73fa2a1400d3eb6d4fad9de1a))

## [1.0.2](https://github.com/iwpnd/pmtilr/compare/v1.0.1...v1.0.2) (2026-04-12)

### 🐛 Bug Fixes

* 🐛 add tilejson method on source to build valid tilejson from meta and header ([aa0ff54](https://github.com/iwpnd/pmtilr/commit/aa0ff54fa0b7bef381a299a3945778e61051415f))

## [1.0.1](https://github.com/iwpnd/pmtilr/compare/v1.0.0...v1.0.1) (2026-04-11)

### 🧹 Miscellaneous

* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#82](https://github.com/iwpnd/pmtilr/issues/82)) ([5402811](https://github.com/iwpnd/pmtilr/commit/540281196a12535671d386d1406b2faf51d153ba))

## 1.0.0 (2026-04-11)

### ✨ Features

* ✨ http range reader ([67a4aeb](https://github.com/iwpnd/pmtilr/commit/67a4aeb21ac044e0faee77515c95d6dd1257a4fa))
* add and test fast hilbert as per pmtiles[#383](https://github.com/iwpnd/pmtilr/issues/383) ([cfada59](https://github.com/iwpnd/pmtilr/commit/cfada59aa83c45da892fd0b974b13fcf6bdf7fe2))
* add optional singleflight ([f7d32ec](https://github.com/iwpnd/pmtilr/commit/f7d32ec439e2c2366715067a5ebe3ab1d373ab6b))
* configurable default ristretto cache ([c2c2be6](https://github.com/iwpnd/pmtilr/commit/c2c2be69e7ab38efe09f035c880682e1243095f4))
* initial commit ([b73c171](https://github.com/iwpnd/pmtilr/commit/b73c1719156c0eb899a2fe58402c107d75f975bf))
* NewRangeReader from uri ([51c1b95](https://github.com/iwpnd/pmtilr/commit/51c1b95bc03dc93e0b7df5eb43e484082ddf30a5))
* s3 range reader, extend test suite for range ([a93355e](https://github.com/iwpnd/pmtilr/commit/a93355edf3706698595a9e017a242e7bc93434f4))
* source config option pattern to change decompress fn ([9af1487](https://github.com/iwpnd/pmtilr/commit/9af148718b8aaf78a6220c27658f83b4072de215))

### 🐛 Bug Fixes

* 🐛 add http range reader to helper ([0c047f9](https://github.com/iwpnd/pmtilr/commit/0c047f9add4c9e4ec66b41ba4d6349e829938cab))
* 🐛 configurable source ([4faffc3](https://github.com/iwpnd/pmtilr/commit/4faffc30cdb2c0c5f01090f2e6ef60a4210dc5f2))
* 🐛 disable singleflight for every request ([b56648e](https://github.com/iwpnd/pmtilr/commit/b56648e9e38f409afd756bf994c0aa826abc6675))
* 🐛 http range reader constructor ([d6c5000](https://github.com/iwpnd/pmtilr/commit/d6c50007260465ffdcdc78f8112863133b521d4c))
* 🐛 ristretto default configuration ([8435ec1](https://github.com/iwpnd/pmtilr/commit/8435ec1ef9527114b5b449f345fae7b026b60447))
* append and unmarshal vector_layers from metadata ([e9ba62a](https://github.com/iwpnd/pmtilr/commit/e9ba62a3138126cd0832628c636133cca19abd59))
* decoding latitude and longitude ([d4459a4](https://github.com/iwpnd/pmtilr/commit/d4459a4738259b1da0de8cb00deb83514abe889c))
* directory traversal ([89432cd](https://github.com/iwpnd/pmtilr/commit/89432cd572a8411631f9d12787274207076797a0))
* s3 config, move test data to testdata dir ([44afda5](https://github.com/iwpnd/pmtilr/commit/44afda51474de3db40439a3c29db0931fa57a415))

### 🚀 Performance

* ⚡️ pre-allocate buffer size ([274dabc](https://github.com/iwpnd/pmtilr/commit/274dabcdf3ece2d48c694bbf0af695642389dca1))
* ⚡️ use iocopy instead of ioreadall for reading tile data ([416938e](https://github.com/iwpnd/pmtilr/commit/416938ed03293984063b0a9f82bb9dd8b73c0b0e))

### 🧹 Miscellaneous

* ♻️ cache string representations of header and metadata ([570482a](https://github.com/iwpnd/pmtilr/commit/570482a8da15d586c12f834a9b35dfb4b25320b9))
* ♻️ deduplicate cache key generation ([33eb060](https://github.com/iwpnd/pmtilr/commit/33eb060932ce558022f57f7dc3c6d7ad41a50cf5))
* ♻️ drop ristretto in favour of otter ([cee6358](https://github.com/iwpnd/pmtilr/commit/cee63587df8d25857fabe68f89b56e893ee5126c))
* ♻️ ensure entries deserialized in correct order ([2a5261e](https://github.com/iwpnd/pmtilr/commit/2a5261e4ef07553e5cfef32193de11b42dafa1f8))
* ♻️ ensure reader closed correctly in loop ([b82cef8](https://github.com/iwpnd/pmtilr/commit/b82cef868109bbd08ef5b0d60357598734961ee5))
* ♻️ gzip reader pool ([e444935](https://github.com/iwpnd/pmtilr/commit/e444935b980d894aa01fe52c777a15e3b49a1ad4))
* ♻️ improve string parsing on hot path ([dabc8ba](https://github.com/iwpnd/pmtilr/commit/dabc8ba28596c73c658d49eaa49b862d3f6450a2))
* ♻️ protect directory with sharded singleflight ([b1043b4](https://github.com/iwpnd/pmtilr/commit/b1043b4a83f3a305192430c47412a2e8f005bf28))
* 🔧 add mlt support ([1437f77](https://github.com/iwpnd/pmtilr/commit/1437f773ab6c3b623dc7526c8540846bbef30c46))
* 🔧 bump golang to 1.25.0 ([db3c7b5](https://github.com/iwpnd/pmtilr/commit/db3c7b5c2cba974cb9bcb19b1d3a04c04cb484d6))
* 🔧 do not inflate tile data ([9810467](https://github.com/iwpnd/pmtilr/commit/9810467c569b75526cbb9821f30c451f996fc3e2))
* 🔧 drop unused entry string serialization ([f31d6b0](https://github.com/iwpnd/pmtilr/commit/f31d6b09b90d20a5700b63e4a308907edf41e865))
* 🔧 explicit returns ([b6b60a6](https://github.com/iwpnd/pmtilr/commit/b6b60a69eac9f5ee30f53020229950d4d8358d62))
* 🔧 fix .releaserc ([786ae07](https://github.com/iwpnd/pmtilr/commit/786ae071dcaefe7150e0917f0875d0ed01183bd4))
* 🔧 fix linter ([3603b20](https://github.com/iwpnd/pmtilr/commit/3603b20e7c1f59ee3f8a9109c612272db3da34c5))
* 🔧 fix mise tparse ref ([9148e5d](https://github.com/iwpnd/pmtilr/commit/9148e5df82fd391830682623d1c615e5054d9470))
* 🔧 increase default timeout on http range reader ([20ff77b](https://github.com/iwpnd/pmtilr/commit/20ff77bfec49386b7ed4e3d37a75925229a3c8be))
* 🔧 let ristretto calculate cost ([9399ed9](https://github.com/iwpnd/pmtilr/commit/9399ed92df3a2154b59f9c1b5ab422f9c67e7b2e))
* 🔧 linter ([613c070](https://github.com/iwpnd/pmtilr/commit/613c070a605ad62a63f03b387e6ffe50717ababd))
* 🔧 remove singleflight, refactor cache interface ([7857da2](https://github.com/iwpnd/pmtilr/commit/7857da23f037fd5ff70a3fb787150bb1a9617622))
* 🔧 switch between in memory caches for testing ([3722ef7](https://github.com/iwpnd/pmtilr/commit/3722ef7bc6cabea3847696cba8dfa334000705be))
* 🔧 try out otter cache instead of ristretto ([323ace9](https://github.com/iwpnd/pmtilr/commit/323ace94b30a4ffda5c78390db88b479f7337048))
* 🔧 user otter v2 as default ([b419fb6](https://github.com/iwpnd/pmtilr/commit/b419fb6a1270e83500809450947ff619bfc7d34c))
* 🔧 vector layer struct for metadata ([b89c393](https://github.com/iwpnd/pmtilr/commit/b89c3937012470bc7bcb8e1566df1ae0e9981b46))
* add readme, add close method on source, update readme to ignore tmp clis ([b22b62d](https://github.com/iwpnd/pmtilr/commit/b22b62dc6d79eb39625acadfde0bc954b3664b8e))
* consolidate hilbert methods ([4354ea1](https://github.com/iwpnd/pmtilr/commit/4354ea1c3a6bececa31f6473bb65b261b1d4c8e9))
* const on top ([58ff521](https://github.com/iwpnd/pmtilr/commit/58ff521d143ba75941fcaa927b7644b53d604b6d))
* decouple header meta and source ([1dc385b](https://github.com/iwpnd/pmtilr/commit/1dc385b7745bea638c7db113f42ebf410461b430))
* **deps-dev:** 🔧 update precommit config ([ab87d04](https://github.com/iwpnd/pmtilr/commit/ab87d048b2e64f9ff21bef817def362202fd2913))
* **deps-dev:** update ([28948e6](https://github.com/iwpnd/pmtilr/commit/28948e6c3d5dad962cd7a8dd34b479d2e75d8968))
* **deps:** 🔗 update ([0e8c5bf](https://github.com/iwpnd/pmtilr/commit/0e8c5bff6bd7027f8e91212746199da1980aa71f))
* **deps:** bump github.com/aws/aws-sdk-go-v2 from 1.41.0 to 1.41.1 ([#65](https://github.com/iwpnd/pmtilr/issues/65)) ([208f42e](https://github.com/iwpnd/pmtilr/commit/208f42ef1435b93c2908c8b3ab9cfef95c3e305f))
* **deps:** bump github.com/aws/aws-sdk-go-v2 from 1.41.3 to 1.41.4 ([#78](https://github.com/iwpnd/pmtilr/issues/78)) ([7f715bb](https://github.com/iwpnd/pmtilr/commit/7f715bb83bf7dc625108c094412b3b1046abb231))
* **deps:** bump github.com/aws/aws-sdk-go-v2 from 1.41.4 to 1.41.5 ([#79](https://github.com/iwpnd/pmtilr/issues/79)) ([3549c40](https://github.com/iwpnd/pmtilr/commit/3549c401612ffd5aa309a43be1bae57e0fb92fe9))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([c28e1df](https://github.com/iwpnd/pmtilr/commit/c28e1df267ad7bfc214a49f0da4cbe7c18311441))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([17e360f](https://github.com/iwpnd/pmtilr/commit/17e360fd0dd9317b796aeeabbff8f72784212a9e))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#54](https://github.com/iwpnd/pmtilr/issues/54)) ([bc839c8](https://github.com/iwpnd/pmtilr/commit/bc839c86a65f3f3c48c4f96c6ea9ae33752b5ddd))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#57](https://github.com/iwpnd/pmtilr/issues/57)) ([40a6366](https://github.com/iwpnd/pmtilr/commit/40a6366fc504e7cc83078302b0ec4ad0436501c9))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#63](https://github.com/iwpnd/pmtilr/issues/63)) ([650c247](https://github.com/iwpnd/pmtilr/commit/650c247db4e4a1dfd69cf8e01b3b3224998c09a0))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#68](https://github.com/iwpnd/pmtilr/issues/68)) ([4b4f7e5](https://github.com/iwpnd/pmtilr/commit/4b4f7e5aee0637614f01dd14850e9d64dc47a1ea))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#73](https://github.com/iwpnd/pmtilr/issues/73)) ([a25caa4](https://github.com/iwpnd/pmtilr/commit/a25caa4dc78cbf905fd4b140d9bbb6bbd180db28))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#76](https://github.com/iwpnd/pmtilr/issues/76)) ([e2bea76](https://github.com/iwpnd/pmtilr/commit/e2bea768f49e5bae2f6f2f5cfb06f47fa3b87fcf))
* **deps:** bump github.com/aws/aws-sdk-go-v2/config ([#83](https://github.com/iwpnd/pmtilr/issues/83)) ([2f7a2da](https://github.com/iwpnd/pmtilr/commit/2f7a2da41c011ab09eee0bea3980b6a57ae6d558))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([3334df6](https://github.com/iwpnd/pmtilr/commit/3334df69916c00080f89cb94bfe38dccd1795ba7))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([c446d6f](https://github.com/iwpnd/pmtilr/commit/c446d6fc2262bf10e0d68a118f8d3c7b31002413))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#55](https://github.com/iwpnd/pmtilr/issues/55)) ([4fb09fe](https://github.com/iwpnd/pmtilr/commit/4fb09fe6af84a9b6229e01336cb5f46000727169))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#64](https://github.com/iwpnd/pmtilr/issues/64)) ([dc0bfff](https://github.com/iwpnd/pmtilr/commit/dc0bfffa7e84916e2a6e85ae58dc8aa20112529e))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#67](https://github.com/iwpnd/pmtilr/issues/67)) ([7646e70](https://github.com/iwpnd/pmtilr/commit/7646e7000e94221da416c3a383b2600c5212e7f5))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#75](https://github.com/iwpnd/pmtilr/issues/75)) ([22e5092](https://github.com/iwpnd/pmtilr/commit/22e50924133d4a3a553d0ea25fd086aa09cd8d96))
* **deps:** bump github.com/aws/aws-sdk-go-v2/service/s3 ([#77](https://github.com/iwpnd/pmtilr/issues/77)) ([880d92a](https://github.com/iwpnd/pmtilr/commit/880d92aa4c42063d8236f7ad3abb8bbc09a3fc48))
* **deps:** bump github.com/dgraph-io/ristretto/v2 from 2.3.0 to 2.4.0 ([#66](https://github.com/iwpnd/pmtilr/issues/66)) ([401eb67](https://github.com/iwpnd/pmtilr/commit/401eb677017b631fbb37bad5ccb5eac9cf3e5ec0))
* drop sprintf in favour of sync pool ([fa3c467](https://github.com/iwpnd/pmtilr/commit/fa3c467752acfbfedec1f578944202979d441498))
* godocs, align size <> length as per pmtiles instead ([f35f5b4](https://github.com/iwpnd/pmtilr/commit/f35f5b43e62c547f83408c98cfeb5378b09a4b08))
* linter ([f590fd7](https://github.com/iwpnd/pmtilr/commit/f590fd77bd9cab56ef5296f2e54b3bc940533854))
* linter fixes ([d196ed2](https://github.com/iwpnd/pmtilr/commit/d196ed2fe327816b51e9e1aa2f78092929639ed8))
* parsing uri ([cd6f2eb](https://github.com/iwpnd/pmtilr/commit/cd6f2eb9bb8dcbabbf89bc5f5a4e4f8119f546c3))
* readrange returns readcloser ([1981b71](https://github.com/iwpnd/pmtilr/commit/1981b714d17f48e422879aed6046eb5a696b21da))
* remove unnecessary bytereader alloc in decompress func ([7acbe55](https://github.com/iwpnd/pmtilr/commit/7acbe558836029cd2bbc266afaed2449104d0c20))
* simpler ranger, add etag to cache key ([e22ba0c](https://github.com/iwpnd/pmtilr/commit/e22ba0c9e64a1d795190cf7dac321186fd09e331))
* use context for ReadFrom ([956eef7](https://github.com/iwpnd/pmtilr/commit/956eef7a99b60b43b2bc32dbe4777d889f7fc35e))
* use FastZXYtoHilbertTileID for 2x speed ([7a13702](https://github.com/iwpnd/pmtilr/commit/7a137027560725363a231b19bd266fdc33c46240))
* use reader pool for directory deserialization ([6e002a0](https://github.com/iwpnd/pmtilr/commit/6e002a0b6b3dda0f34d35cc5ee0d220e4787307b))
* use uri to instantiate new source ([a5796aa](https://github.com/iwpnd/pmtilr/commit/a5796aa5e2f9d164d6597b73c73a50afef387019))
