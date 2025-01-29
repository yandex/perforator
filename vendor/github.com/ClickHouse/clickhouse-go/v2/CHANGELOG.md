# v2.18.0, 2024-02-01 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Add WithAllocBufferColStrProvider string column allocator for batch insert performance boost by @hongker in https://github.com/ClickHouse/clickhouse-go/pull/1181
### Fixes 🐛
* Fix bind for seconds scale DateTime by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1184
### Other Changes 🛠
* resolves #1163 debugF function is not respected by @omurbekjk in https://github.com/ClickHouse/clickhouse-go/pull/1166

## New Contributors
* @omurbekjk made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1166
* @hongker made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1181

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.17.1...v2.18.0

# v2.17.1, 2023-12-27 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* fix panic in contextWatchDog nil pointer check by @nityanandagohain in https://github.com/ClickHouse/clickhouse-go/pull/1168

## New Contributors
* @nityanandagohain made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1168

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.17.0...v2.17.1

# v2.17.0, 2023-12-21 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Iterable ordered map alternative with improved performance by @hanjm in https://github.com/ClickHouse/clickhouse-go/pull/1152
* Support bool alias type by @yogasw in https://github.com/ClickHouse/clickhouse-go/pull/1156
### Fixes 🐛
* Update README - mention HTTP protocol usable only with `database/sql` interface by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1160
* Fix README example for Debugf by @aramperes in https://github.com/ClickHouse/clickhouse-go/pull/1153

## New Contributors
* @yogasw made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1156
* @aramperes made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1153

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.16.0...v2.17.0

# v2.16.0, 2023-12-01 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Add sql.Valuer support for all types by @deankarn in https://github.com/ClickHouse/clickhouse-go/pull/1144
### Fixes 🐛
* Fix DateTime64 range to actual supported range per ClickHouse documentation by @phil-schreiber in https://github.com/ClickHouse/clickhouse-go/pull/1148

## New Contributors
* @phil-schreiber made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1148
* @deankarn made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1144

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.15.0...v2.16.0

# v2.14.3, 2023-10-12 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* Fix insertion of empty map into JSON column by using _dummy subcolumn by @leodido in https://github.com/ClickHouse/clickhouse-go/pull/1116
### Other Changes 🛠
* chore: specify method field on compression in example by @rdaniels6813 in https://github.com/ClickHouse/clickhouse-go/pull/1111
* chore: remove extra error checks by @rutaka-n in https://github.com/ClickHouse/clickhouse-go/pull/1095

## New Contributors
* @leodido made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1116
* @rdaniels6813 made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1111
* @rutaka-n made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1095

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.14.2...v2.14.3

# v2.14.2, 2023-10-04 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* Fix: Block stream read process would be terminated by empty block with zero rows by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/1104
* Free compressor's buffer when FreeBufOnConnRelease enabled by @cergxx in https://github.com/ClickHouse/clickhouse-go/pull/1100
* Fix truncate ` for HTTP adapter by @beck917 in https://github.com/ClickHouse/clickhouse-go/pull/1103
### Other Changes 🛠
* docs: update readme.md by @rfyiamcool in https://github.com/ClickHouse/clickhouse-go/pull/1068
* Remove dependency on github.com/satori/go.uuid by @srikanthccv in https://github.com/ClickHouse/clickhouse-go/pull/1085

## New Contributors
* @rfyiamcool made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1068
* @beck917 made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1103
* @srikanthccv made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1085

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.14.1...v2.14.2

# v2.14.1, 2023-09-14 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* parseDSN: support connection pool settings (#1082) by @hanjm in https://github.com/ClickHouse/clickhouse-go/pull/1084

## New Contributors
* @hanjm made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1084

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.14.0...v2.14.1

# v2.14.0, 2023-09-12 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Add FreeBufOnConnRelease to clickhouse.Options by @cergxx in https://github.com/ClickHouse/clickhouse-go/pull/1091
* Improving object allocation for (positional) parameter binding by @mdonkers in https://github.com/ClickHouse/clickhouse-go/pull/1092
### Fixes 🐛
* Fix escaping double quote in SQL statement in prepare batch by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1083
### Other Changes 🛠
* Update Go & ClickHouse versions by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1079
* Return status code from any http error by @RoryCrispin in https://github.com/ClickHouse/clickhouse-go/pull/1090
* tests: fix dropped error by @alrs in https://github.com/ClickHouse/clickhouse-go/pull/1081
* chore: unnecessary use of fmt.Sprintf by @testwill in https://github.com/ClickHouse/clickhouse-go/pull/1080
* Run CI on self hosted runner by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1094

## New Contributors
* @cergxx made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1091
* @alrs made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1081
* @testwill made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1080

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.13.4...v2.14

# v2.13.4, 2023-08-30 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* fix(proto): add TCP protocol version in query packet by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1077


**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.13.3...v2.13.4

# v2.13.3, 2023-08-23 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* fix(column.json): fix bool type handling by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1073


**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.13.2...v2.13.3

# v2.13.2, 2023-08-18 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* fix: update ch-go to remove string length limit by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1071
### Other Changes 🛠
* Test against latest and head CH by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1060


**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.13.1...v2.13.2

# v2.13.1, 2023-08-17 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* fix: native format Date32 representation by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1069


**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.13.0...v2.13.1

# v2.13.0, 2023-08-10 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Support scan from uint8 to bool by @ValManP in https://github.com/ClickHouse/clickhouse-go/pull/1051
* Binding arguments for AsyncInsert interface by @mdonkers in https://github.com/ClickHouse/clickhouse-go/pull/1052
* Batch rows count API by @EpicStep in https://github.com/ClickHouse/clickhouse-go/pull/1063
* Implement release connection in batch by @EpicStep in https://github.com/ClickHouse/clickhouse-go/pull/1062
### Other Changes 🛠
* Restore test against CH 23.7 by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1059

## New Contributors
* @ValManP made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1051

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.12.1...v2.13.0

# v2.12.1, 2023-08-02 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* Fix InsertAsync typo in docs  by @et in https://github.com/ClickHouse/clickhouse-go/pull/1044
* Fix panic and releasing in batch column by @EpicStep in https://github.com/ClickHouse/clickhouse-go/pull/1055
* Docs/changelog fixes by @jmaicher in https://github.com/ClickHouse/clickhouse-go/pull/1046
* Clarify error message re custom serializaion support by @RoryCrispin in https://github.com/ClickHouse/clickhouse-go/pull/1056
* Fix send query on batch retry by @EpicStep in https://github.com/ClickHouse/clickhouse-go/pull/1045
### Other Changes 🛠
* Update ClickHouse versions by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1054

## New Contributors
* @et made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1044
* @EpicStep made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1055
* @jmaicher made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1046
* @RoryCrispin made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1056

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.12.0...v2.12.1

# v2.12.0, 2023-07-27 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Implement elapsed time in query progress by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1039
### Fixes 🐛
* Release connection slot on connection acquire timeout by @sentanos in https://github.com/ClickHouse/clickhouse-go/pull/1042

## New Contributors
* @sentanos made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1042

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.11.0...v2.12.0

# v2.11.0, 2023-07-20 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Retry for batch API by @djosephsen in https://github.com/ClickHouse/clickhouse-go/pull/941
### Fixes 🐛
* Fix startAutoCloseIdleConnections cause goroutine leak by @YenchangChan in https://github.com/ClickHouse/clickhouse-go/pull/1011
* Fix netip.Addr pointer panic by @anjmao in https://github.com/ClickHouse/clickhouse-go/pull/1029
### Other Changes 🛠
* Git actions terraform by @gingerwizard in https://github.com/ClickHouse/clickhouse-go/pull/1023

## New Contributors
* @YenchangChan made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1011
* @djosephsen made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/941
* @anjmao made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1029

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.10.1...v2.11.0

# v2.10.1, 2023-06-06 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Other Changes 🛠
* Update outdated README.md by @kokizzu in https://github.com/ClickHouse/clickhouse-go/pull/1006
* Remove incorrect usage of KeepAlive in DialContext by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/1009

## New Contributors
* @kokizzu made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/1006

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.10.0...v2.10.1

# v2.10.0, 2023-05-17 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Support [16]byte/[]byte typed scan/append for IPv6 column by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/996
* Add custom dialer option to http protocol by @stephaniehingtgen in https://github.com/ClickHouse/clickhouse-go/pull/998
### Fixes 🐛
* Tuple scan respects both value and pointer variable by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/971
* Auto close idle connections in native protocol in respect of ConnMaxLifetime option by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/999

## New Contributors
* @stephaniehingtgen made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/998

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.9.3...v2.10.0

# v2.9.2, 2023-05-08 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Fixes 🐛
* Pass http.ProxyFromEnvironment configuration to http.Transport by @slvrtrn in https://github.com/ClickHouse/clickhouse-go/pull/987
### Other Changes 🛠
* Use `any` instead of `interface{}` by @candiduslynx in https://github.com/ClickHouse/clickhouse-go/pull/984

## New Contributors
* @candiduslynx made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/984
* @slvrtrn made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/987

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.9.1...v2.9.2

# v2.9.1, 2023-04-24 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* Do not return hard error on unparsable version in HTTP proto by @hexchain in https://github.com/ClickHouse/clickhouse-go/pull/975
### Fixes 🐛
* Return ErrBadConn in stdDriver Prepare if connection is broken by @czubocha in https://github.com/ClickHouse/clickhouse-go/pull/977

## New Contributors
* @czubocha made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/977
* @hexchain made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/975

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.9.0...v2.9.1

# v2.9.0, 2023-04-13 <!-- Release notes generated using configuration in .github/release.yml at main -->

## What's Changed
### Enhancements 🎉
* External tables support for HTTP protocol by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/942
* Support driver.Valuer in String and FixedString columns by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/946
* Support boolean and pointer type parameter binding by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/963
* Support insert/scan IPv4 using UInt32/*UInt32 types by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/966
### Fixes 🐛
* Reset the pointer to the nullable field by @xiaochaoren1 in https://github.com/ClickHouse/clickhouse-go/pull/964
* Enable to use ternary operator with named arguments by @crisismaple in https://github.com/ClickHouse/clickhouse-go/pull/965
### Other Changes 🛠
* chore: explain async insert in docs by @jkaflik in https://github.com/ClickHouse/clickhouse-go/pull/969

## New Contributors
* @xiaochaoren1 made their first contribution in https://github.com/ClickHouse/clickhouse-go/pull/964

**Full Changelog**: https://github.com/ClickHouse/clickhouse-go/compare/v2.8.3...v2.9.0

## 2.8.3, 2023-04-03

### Bug fixes

- Revert: Expire idle connections no longer acquired during lifetime [#958](https://github.com/ClickHouse/clickhouse-go/pull/958) by @jkaflik

## 2.8.2, 2023-03-31

### Bug fixes

- Expire idle connections no longer acquired during lifetime [#945](https://github.com/ClickHouse/clickhouse-go/pull/945) by @jkaflik

## 2.8.1, 2023-03-29

### Bug fixes

- Fix idle connection check for TLS connections [#951](https://github.com/ClickHouse/clickhouse-go/pull/951) by @jkaflik & @alekar

## 2.8.0, 2023-03-27

### New features

- Support customized "url path" in http connection [#938](https://github.com/ClickHouse/clickhouse-go/pull/938) by @crisismaple
- Allow Auth.Database option to be empty [#926](https://github.com/ClickHouse/clickhouse-go/pull/938) by @v4run

### Chores

- Bump github.com/stretchr/testify from 1.8.1 to 1.8.2 [#933](https://github.com/ClickHouse/clickhouse-go/pull/933)
- fix: small typo in the text of an error [#936](https://github.com/ClickHouse/clickhouse-go/pull/936) by @lspgn
- Improved bug template [#916](https://github.com/ClickHouse/clickhouse-go/pull/916) by @mshustov

## 2.7.0, 2023-03-08

### New features

- Date type with user location [#923](https://github.com/ClickHouse/clickhouse-go/pull/923) by @jkaflik
- Add AppendRow function to BatchColumn [#927](https://github.com/ClickHouse/clickhouse-go/pull/927) by @pikot

### Bug fixes

- fix: fix connect.compression's format verb [#924](https://github.com/ClickHouse/clickhouse-go/pull/924) by @mind1949
- Add extra padding for strings shorter than FixedColumn length [#910](https://github.com/ClickHouse/clickhouse-go/pull/910) by @jkaflik

### Chore

- Bump github.com/andybalholm/brotli from 1.0.4 to 1.0.5 [#911](https://github.com/ClickHouse/clickhouse-go/pull/911)
- Bump github.com/paulmach/orb from 0.8.0 to 0.9.0 [#912](https://github.com/ClickHouse/clickhouse-go/pull/912)
- Bump golang.org/x/net from 0.0.0-20220722155237-a158d28d115b to 0.7.0 [#928](https://github.com/ClickHouse/clickhouse-go/pull/928)

## 2.6.5, 2023-02-28

### Bug fixes

- Fix array parameter formatting in binding mechanism [#921](https://github.com/ClickHouse/clickhouse-go/pull/921) by @genzgd

## 2.6.4, 2023-02-23

### Bug fixes

- Fixed concurrency issue in stdConnOpener [#918](https://github.com/ClickHouse/clickhouse-go/pull/918) by @jkaflik

## 2.6.3, 2023-02-22

### Bug fixes

- Fixed `lib/binary/string_safe.go` for non 64bit arch [#914](https://github.com/ClickHouse/clickhouse-go/pull/914) by @atoulme
 
## 2.6.2, 2023-02-20

### Bug fixes

- Fix decimal encoding with non-standard exponential representation [#909](https://github.com/ClickHouse/clickhouse-go/pull/909) by @vogrelord
- Add extra padding for strings shorter than FixedColumn length [#910](https://github.com/ClickHouse/clickhouse-go/pull/910) by @jkaflik

### Chore

- Remove Yandex ClickHouse image from Makefile [#895](https://github.com/ClickHouse/clickhouse-go/pull/895) by @alexey-milovidov
- Remove duplicate of error handling [#898](https://github.com/ClickHouse/clickhouse-go/pull/898) by @Astemirdum
- Bump github.com/ClickHouse/ch-go from 0.51.2 to 0.52.1 [#901](https://github.com/ClickHouse/clickhouse-go/pull/901)

## 2.6.1, 2023-02-13

### Bug fixes

- Do not reuse expired connections (`ConnMaxLifetime`) [#892](https://github.com/ClickHouse/clickhouse-go/pull/892) by @iamluc
- Extend default dial timeout value to 30s [#893](https://github.com/ClickHouse/clickhouse-go/pull/893) by @jkaflik
- Compression name fixed in sendQuery log  [#884](https://github.com/ClickHouse/clickhouse-go/pull/884) by @fredngr

## 2.6.0, 2023-01-27

### New features

- Client info specification implementation [#876](https://github.com/ClickHouse/clickhouse-go/pull/876) by @jkaflik

### Bug fixes

- Better handling for broken connection errors in the std interface [#879](https://github.com/ClickHouse/clickhouse-go/pull/879) by @n-oden

### Chore

- Document way to provide table or database identifier with query parameters [#875](https://github.com/ClickHouse/clickhouse-go/pull/875) by @jkaflik
- Bump github.com/ClickHouse/ch-go from 0.51.0 to 0.51.2 [#881](https://github.com/ClickHouse/clickhouse-go/pull/881)

## 2.5.1, 2023-01-10

### Bug fixes

- Flag connection as closed on broken pipe [#871](https://github.com/ClickHouse/clickhouse-go/pull/871) by @n-oden

## 2.5.0, 2023-01-10

### New features

- Buffered compression column by column for a native protocol. Introduces the `MaxCompressionBuffer` option - max size (bytes) of compression buffer during column-by-column compression (default 10MiB) [#808](https://github.com/ClickHouse/clickhouse-go/pull/808) by @gingerwizard and @jkaflik
- Support custom types that implement `sql.Scanner` interface (e.g. `type customString string`) [#850](https://github.com/ClickHouse/clickhouse-go/pull/850) by @DarkDrim
- Append query options to the context instead of overwriting [#860](https://github.com/ClickHouse/clickhouse-go/pull/860) by @aaron276h
- Query parameters support [#854](https://github.com/ClickHouse/clickhouse-go/pull/854) by @jkaflik
- Expose `DialStrategy` function to the user for custom connection routing. [#855](https://github.com/ClickHouse/clickhouse-go/pull/855) by @jkaflik

### Bug fixes

- Close connection on `Cancel`. This is to make sure context timed out/canceled connection is not reused further [#764](https://github.com/ClickHouse/clickhouse-go/pull/764) by @gingerwizard
- Fully parse `secure` and `skip_verify` in DSN query parameters. [#862](https://github.com/ClickHouse/clickhouse-go/pull/862) by @n-oden

### Chore

- Added tests covering read-only user queries [#837](https://github.com/ClickHouse/clickhouse-go/pull/837) by @jkaflik
- Agreed on a batch append fail semantics [#853](https://github.com/ClickHouse/clickhouse-go/pull/853) by @jkaflik

## 2.4.3, 2022-11-30
### Bug Fixes
* Fix in batch concurrency - batch could panic if used in separate go routines. <br/>
The issue was originally detected due to the use of a batch in a go routine and Abort being called after the connection was released on the batch. This would invalidate the connection which had been subsequently reassigned. <br/>
This issue could occur as soon as the conn is released (this can happen in a number of places e.g. after Send or an Append error), and it potentially returns to the pool for use in another go routine. Subsequent releases could then occur e.g., the user calls Abort mainly but also Send would do it. The result is the connection being closed in the release function while another batch or query potentially used it. <br/>
This release includes a guard to prevent release from being called more than once on a batch. It assumes that batches are not thread-safe - they aren't (only connections are).
## 2.4.2, 2022-11-24
### Bug Fixes
- Don't panic on `Send()` on batch after invalid `Append`. [#830](https://github.com/ClickHouse/clickhouse-go/pull/830)
- Fix JSON issue with `nil` if column order is inconsisent. [#824](https://github.com/ClickHouse/clickhouse-go/pull/824)

## 2.4.1, 2022-11-23
### Bug Fixes
- Patch release to fix "Regression - escape character was not considered when comparing column names". [#828](https://github.com/ClickHouse/clickhouse-go/issues/828)

## 2.4.0, 2022-11-22
### New Features
- Support for Nullables in Tuples. [#821](https://github.com/ClickHouse/clickhouse-go/pull/821) [#817](https://github.com/ClickHouse/clickhouse-go/pull/817)
- Use headers for auth and not url if SSL. [#811](https://github.com/ClickHouse/clickhouse-go/pull/811)
- Support additional headers. [#811](https://github.com/ClickHouse/clickhouse-go/pull/811)
- Support int64 for DateTime. [#807](https://github.com/ClickHouse/clickhouse-go/pull/807)
- Support inserting Enums as int8/int16/int. [#802](https://github.com/ClickHouse/clickhouse-go/pull/802)
- Print error if unsupported server. [#792](https://github.com/ClickHouse/clickhouse-go/pull/792)
- Allow block buffer size to tuned for performance - see `BlockBufferSize`. [#776](https://github.com/ClickHouse/clickhouse-go/pull/776)
- Support custom datetime in Scan. [#767](https://github.com/ClickHouse/clickhouse-go/pull/767)
- Support insertion of an orderedmap. [#763](https://github.com/ClickHouse/clickhouse-go/pull/763)

### Bug Fixes
- Decompress errors over HTTP. [#792](https://github.com/ClickHouse/clickhouse-go/pull/792)
- Use `timezone` vs `timeZone` so we work on older versions. [#781](https://github.com/ClickHouse/clickhouse-go/pull/781)
- Ensure only columns specified in INSERT are required in batch. [#790](https://github.com/ClickHouse/clickhouse-go/pull/790)
- Respect order of columns in insert for batch. [#790](https://github.com/ClickHouse/clickhouse-go/pull/790)
- Handle double pointers for Nullable columns when batch inserting. [#774](https://github.com/ClickHouse/clickhouse-go/pull/774)
- Use nil for `LowCardinality(Nullable(X))`. [#768](https://github.com/ClickHouse/clickhouse-go/pull/768)

### Breaking Changes
- Align timezone handling with spec. [#776](https://github.com/ClickHouse/clickhouse-go/pull/766), specifically:
    - If parsing strings for datetime, datetime64 or dates we assume the locale is Local (i.e. the client) if not specified in the string.
    - The server (or column tz) is used for datetime and datetime64 rendering. For date/date32, these have no tz info in the server. For now, they will be rendered as UTC - consistent with the clickhouse-client
    - Addresses bind when no location is set
