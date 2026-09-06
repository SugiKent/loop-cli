# go install 配布の版判定と落とし穴

最終更新: 2026-09-06-1200

loop-cli は `go install` だけで配る（[MVP 定義](../mvp/mvp.md)）。自己更新まわり（`internal/version`、`loop-cli update`）を触るときに要る 3 点。
いずれも Go 1.26.6 で実測して確かめた。

## 手元 build の判別は版の文字列ではなく `vcs.revision`

Go は作業ツリーでの `go build` にも VCS 由来の擬似バージョン（`v0.0.0-<日時>-<hash>+dirty`）を刻む。
`(devel)` になるのは `-buildvcs=false` のときだけなので、版の文字列では `go install <module>@<version>` と区別できない。
`vcs.*` の build setting は module cache から入れたバイナリには付かないので、`debug.ReadBuildInfo()` の `Settings` に
`vcs.revision` があるかどうかで見分ける（`go version -m <バイナリ>` で目視できる）。

## `go` を呼ぶときは module の外で走らせる

`go list -m -json <module>@latest` も `go install <module>@latest` も、vendor ディレクトリを持つプロジェクトの中では
`-mod=vendor` が自動で効き `cannot query module due to -mod=vendor` で落ちる。開発中のプロジェクトのディレクトリで打つ道具なので、
`exec.Cmd.Dir` を `os.TempDir()` に固定する。main module のチェックアウトの中では `@latest` は解決できるので、症状は vendor 特有。

## `go list` が通っても `go install` が通るとは限らない

proxy は `.info` を返すときに `go.mod` の `module` 行を検証しない。モジュールパスが公開リポジトリと食い違っていても
`go list -m -json <module>@latest` は成功して `Version` を返し、`go install` だけが
`module declares its path as: …  but was required as: …` で落ちる。
配布が壊れていないことを確かめるときは、必ず `go install` を実行する。

```sh
cd $(mktemp -d) && GOBIN=$PWD go install github.com/SugiKent/loop-cli/cmd/loop-cli@latest
```
