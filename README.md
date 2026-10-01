# brainium-tools

org-mode で書いた個人の知識リポジトリ(brainium 形式)を扱う Go 製のツール群。

- `orglint` — 規約の検査と、本文段落の折り返しの自動修正
- `notesearch` — `notes/` の BM25 全文検索。原文の抜粋か見出しの木を返す
- `notetree` — ノートの見出しを行範囲付きの目次として出す。agent が目次
  をたどって読む範囲を選ぶためのもので、引数なしなら `notes/` 全体を第
  1 階層まで、パスを渡すとそのファイルを全階層で出す
- `notestatus` — 精錬待ち(`draft` と `:STALE_AFTER:` 超過)のノートの列挙
- `notesearch-eval` — 評価クエリ集で `notesearch` のランキングを測る

汎用化はしていない。下の構成と規約に揃えたリポジトリで使う前提で、
ディレクトリ名や検査内容は設定で変えられない。

## 使い方

```console
$ nix run github:Warashi/brainium-tools#orglint -- .
$ nix run github:Warashi/brainium-tools#notesearch -- <query>
```

自分のリポジトリの flake から使うときは input に足し、
`packages.<system>.default` を devShell に入れる。リポジトリルートは
作業ディレクトリから `.git` を辿って決めるので、どのディレクトリからでも呼べる。

## 前提とするリポジトリの構成

- `inbox.org` — org-capture の投入先
- `tasks.org` — TODO 管理。`tasks.org_archive` は ID の供給元としてだけ読む
- `journal/<YYYY>/<YYYY-MM-DD>.org` — デイリーノート
- `notes/` — 恒久ノート。フラット配置で、ファイル名は英語の kebab-case

## orglint が検査する規約

- ID の重複と `id:` リンク先の存在
- ファイル配置と命名規則
- `README.org` を除く全ファイルのファイルレベル `:ID:` と `#+title:`
- `inbox.org` の capture 見出しの `:ID:` と `:CAPTURED:`
- `notes/` の成熟度。`:STATUS:` は `draft` / `stable` / `deprecated` の
  いずれかで、`stable` なら `:VERIFIED:` が必須。`:VERIFIED:` と
  `:STALE_AFTER:` は org の日付として読めること
- 本文段落の折り返し(全角幅換算 70 桁、Emacs の `fill-column` 70 相当)
- `=…=` / `~…~` のコード表記の前後の文字
- `tasks.org` の 1 task が 40 行以内であること

`orglint -fix` は折り返しだけを直す。`orglint -fix-only` は検査せずに
折り返しだけを直すので、treefmt の formatter に使える。

## notesearch

```console
$ notesearch <query>                 # 原文の抜粋付き
$ notesearch -format tree <query>    # ヒットしたノートの見出しの木と一致した節
$ notesearch -format json <query>
$ notesearch -format paths <query>
```

## 開発

```console
$ nix develop
$ go test ./...
$ nix flake check
$ nix fmt
```
