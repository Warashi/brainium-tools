# brainium-tools

org-mode で書いた個人の知識リポジトリ(brainium 形式)を扱う Go 製のツール群。

- `orglint` — 規約の検査と、本文段落の折り返しの自動修正
- `notesearch` — `notes/` の BM25 全文検索。原文の抜粋か見出しの木を返す
- `notetree` — ノートの目次を出す。agent が目次をたどって読む範囲を選ぶ
  ためのもので、引数なしなら `notes/` 全体を冒頭の文と第 1 階層の見出し
  で、パスを渡すとそのファイルを行範囲付きの木で出す。木は丸ごと読むほ
  うが安い部分を畳み、出力は `-max-bytes` に収まるまで削る
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
- `work/` — 進行中の作業ファイル。配置と命名は `notes/` と同じ

## orglint が検査する規約

- ID の重複と `id:` リンク先の存在
- ファイル配置と命名規則。`notes/` と `work/` には org 以外も含めてサブディレクトリを置かない
- `README.org` を除く全ファイルのファイルレベル `:ID:` と `#+title:`
- `inbox.org` の capture 見出しの `:ID:` と `:CAPTURED:`
- `notes/` の成熟度。`:STATUS:` は `draft` / `stable` / `deprecated` の
  いずれかで、`stable` なら `:VERIFIED:` が必須。`:VERIFIED:` と
  `:STALE_AFTER:` は org の日付として読めること
- 本文段落の折り返し(全角幅換算 70 桁、Emacs の `fill-column` 70 相当)。
  行のつなぎ方は `fill-separate-heterogeneous-words-with-space` を `t` に
  した Emacs に合わせ、つなぎ目の両側が和文のときだけ詰める
- `=…=` / `~…~` のコード表記の前後の文字
- `tasks.org` の 1 task が 40 行以内であること

`orglint -fix` は折り返しと、和文に直接続くコード表記の空白を直す。空
白は、前が和文の文字か開き括弧以外の全角の句読点・記号、または後ろが
和文の文字のときだけ補う。後ろが全角の句読点・閉じ括弧のものは文意で
直し方が変わるので報告だけする。`orglint -fix-only` は検査せずに同じ修
正だけをするので、treefmt の formatter に使える。

## notesearch

```console
$ notesearch <query>                 # 原文の抜粋付き
$ notesearch -format tree <query>    # 一致した節の位置と第 1 階層の目次
$ notesearch -format json <query>
$ notesearch -format paths <query>
```

text と tree は、ヒットしたノートとリンクでつながる `notes/` のノートを
`Links:` に並べる。順位には効かない。検索語が重ならなくても 1 手で届く
ノートを見せるためのもの。

## 開発

```console
$ nix develop
$ go test ./...
$ nix flake check
$ nix fmt
```
