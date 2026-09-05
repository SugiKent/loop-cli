# MVP 実装タスク（順序付き）

最終更新: 2026-09-05-1805

> このドキュメントは「実装の順序」を定義する。各タスクの詳細実装 spec は OpenSpec で別途設計する。
> 上から順番に着手し、前段が完了してから次へ進むこと。

仕様は [mvp.md](./mvp.md)、判断の根拠は [decisions.md](./decisions.md)、
分類ルールは [人の出番（判定ルール）](../domain/issue-driven-sdd/human-turn-signals.md) を見る。

主要フロー（最優先で通す体験）: **A 質問に答える** / **E 着手を承認する** / **B 方針を決める**。
A と B はどちらも「`question` が付いたものにコメントで答える」なので同じ実装（書き先が PR か issue か）。
各項目の `(P1)` `(P2)` `(P3)` は到達段階。P1 が揃えば最小で回る。

---

## 1. 基盤セットアップ

- [ ] `go mod init` と依存追加（Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 / Glamour / huh）
- [ ] gofmt / `go vet` / golangci-lint のセットアップ
- [ ] `cmd/sugi-loop` の hello world（起動して 1 フレーム描画、`q` で終了）で `go build` が通る状態にする
- [ ] CI（build / vet / test）を通す

## 2. 実装のための仕組み

DB は持たない（[D-002](./decisions.md)）。seed / testData の代わりに **fixture** を使う。

- [ ] `internal/config`: `~/.config/sugi-loop/config.yml` の読み込み
- [ ] `internal/gh`: `GHClient` interface、`gh` サブプロセス実装、JSON fixture の fake
- [ ] fixture 採取: 稼働中リポジトリ 1 件から `gh search` / `gh issue view` / `gh pr view` の JSON を取り、個人・組織情報を伏せて保存
- [ ] `internal/classify`: 局面 A〜G と「その他」バケットを純粋関数で実装し、fixture でテストする（V-1。このツールの価値の本体なので UI より先に書く）
- [ ] 動作確認用 CLI の導入（`go run ./cmd/sugi-loop-cli help`。用途: gh の生 JSON を fixture に保存、fixture に分類器をかけてキューをプレーンテキスト出力、テスト通知の発火。詳細は 85-create-cli を参照）

## 3. ユーザー体験順の実装

認証は `gh auth` を再利用するため認証章は無い。ユーザーが触る順番で並べる。各項目は OpenSpec で別途詳細 spec を作る単位。

- [ ] (P1) 横断取得: search 2 回 + 分類に必要な遅延詳細取得（[D-001](./decisions.md)）。PR title / body のパースで Issue に PR を紐づけてカード化
- [ ] (P1) 今やるキュー画面: 4 タブ（今やる / バックログ / 進行中 / 異常）+ 選択行プレビュー。狭い端末は 1 ペインにフォールバック
- [ ] (P1) カード詳細（Enter）: Glamour 表示、AI / 人のコメント区別（エスケープ済みマーカー・`## PR リスク評価` 見出しも AI 扱い）、紐づく PR 一覧
- [ ] (P1) **フロー A / B** `a` 回答: PR の `question` は `gh pr comment`、issue の `question` は `gh issue comment`。`## Q1.` と選択肢をパースして `Q1: A` テンプレートを事前入力、投稿前に `blocked-by:` 行を検出して警告。ラベルは触らない
- [ ] (P1) **フロー E**: `t` で stage:todo を付ける / 外す（1 操作 1 ラベルで呼ぶ）
- [ ] (P1) `o` ブラウザで開く / `R` 全件再取得 / `?` ヘルプ
- [ ] (P1) 自動更新（既定 120 秒）+ スナップショットキャッシュ + デスクトップ通知（更新前後で「今やる」を差分比較し、増えたカードを 1 件 1 通知。beeep）
- [ ] (P2) フロー C `m` merge（ガード: `question` / 1 行目 N>0 / checks 失敗 / draft を拒否）
- [ ] (P2) `n` Issue 作成フォーム（段階ラベルは付けない）/ `s` 即着手
- [ ] (P2) フロー D: review thread 一覧と `A` 返信（REST replies）
- [ ] (P2) カンバンビュー `v`、ラベル変遷タイムライン（REST timeline）、cross-reference による紐づけ補完
- [ ] (P2) レートリミットをステータスバーに表示し、書き込み後は対象 1 件だけを再取得する
- [ ] (P3) GraphQL 1 リクエストへの統合、`/` 絞り込み保存、通知クリックで該当カードを開く
- [ ] (P3) GitHub Notifications をソースに追加（設定に無いリポジトリの mention）
- [ ] (P3) Routine の実行状況（run 一覧）の表示

---

## 変更履歴

| 日時 | 変更内容 | 理由 |
| --- | --- | --- |
| 2026-09-05-1805 | 上流 `d8db3842` に同期。`u` の項目を削除し、フロー B を `a` の項目に統合。P2 の異常検知拡張（`restart: 3/3`、`question` 付き merge）を削除 | 人がラベルを触らない規約になり、これらは dispatcher が B に変換するため |
| 2026-09-05-1418 | Phase 1/2/3 構造を「基盤 → 仕組み → 体験順」の 4 章構造（認証章は不要のため省略）に再構成。主要フローを A / E / B に確定し、`u` を P1 に前倒し | 80-plan-mvp-impl-tasks の順序付きタスク形式に揃え、OpenSpec で 1 項目ずつ spec 化できる粒度にするため |
| 2026-09-05-1407 | `docs/mvp/design.md` の「MVP の段階」を実装タスクとして分離 | docs 管理規約が求める `docs/mvp/implementation-tasks.md` を実装順序の正本にするため |
