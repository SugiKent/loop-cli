---
paths:
  - "openspec/changes/archive/**"
  - "openspec/specs/**"
---

# openspec の archive とメイン spec は直接編集しない

- `openspec/changes/archive/**` は完了した change の履歴。当時の記述をそのまま残す。旧いモジュールパス・旧い名前・当時しか実行しなかったコマンドが出てきても、現在の値に直さない
- `openspec/specs/**` は archive 時に change の delta（`openspec/changes/<id>/specs/**`）が更新する正本。仕様を変えたいときはメイン spec を直接書き換えず、進行中の change に ADDED / MODIFIED の delta を書く

リポジトリ全体に機械的な置換（rename・import パスの変更・API 名の変更）をかけるときは、この 2 つを対象から外す。
置換の後に `git diff --stat -- openspec/changes/archive openspec/specs` が空であることを確認する。
巻き込むと、履歴が書き換わるうえ、archive 時に矛盾した記述がメイン spec へ恒久的に取り込まれる。
