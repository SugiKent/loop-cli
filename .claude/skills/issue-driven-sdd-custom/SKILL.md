---
name: issue-driven-sdd-custom
description: issue-driven-sdd plugin の routine がこのプロジェクトで従う固有の調整。worker が routine-common の直後に読む。単独では実行しない。
disable-model-invocation: true
---

## routines

（`routines-setup` が控える Routine の id。worker は読まない。`RemoteTrigger list` は 1 ページ目しか返さないので、ここが正本）

| 役割 | id |
| --- | --- |
| dispatch | |
| propose | |
| apply | |
| archive | |
| sweep | |
| assess | `trig_01GmEgPjsRSnK7kobNEEr4Wx` |

dispatch / propose / apply / archive / sweep は 2026-09-11 時点で未記録。`RemoteTrigger list` の 1 ページ目が
worker の作った再確認リマインダーで埋まっており API から引けないので、`claude.ai/code/routines` の UI で
id を確かめて埋める。

assess は 2026-09-11 に追加した。トリガーは `pull_request.labeled` + `pr.labels IN [ai-assess:requested]`、
環境は `env_01EXzBS7YGsKLjSqpgCHVvgr`（loop-cli）、model は `claude-sonnet-5`、本文は
`` `assess-pr-risk` skill を読み、そのとおりに実行する ``。

## 共通

（E2E の要否、スクリーンショットの方針、アーティファクトの作り先、着手してはいけない領域、教訓の書き残し先など。空なら既定どおり）

### assess は merge しない

`assess-pr-risk` は評価をコメントして `ai-assess:requested` を外すだけで、merge はしない。merge するかは
人の判断で、loop-cli の今やるタブ（局面 C）はその判断点を人に届けるためにある。

### worker は再確認リマインダーを作らない

PR を作ったあとの追跡は `autofix_on_pr_create` のセッションと sweep が持つ。`RemoteTrigger` で
「PR #n を再確認する」リマインダーを自分で作らない。2026-09-11 時点で merge 済み PR の残骸が 17 本あり、
`RemoteTrigger list` の 1 ページ目を埋めて本物の Routine を引けなくしている。

## propose

## apply

## archive
