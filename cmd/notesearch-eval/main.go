// Command notesearch-eval は凍結した評価クエリで notesearch を測る。
//
//	notesearch-eval -queries <path> [-split train|test|all] [-diff] [-maxgram N]
//
// 素の BM25 を基準に、比較したい設定の指標を並べて出す。-diff は 2 つの
// 設定の間で正解ノートの順位が動いたクエリだけを出す。既定の split は
// train で、test はクエリ集に添えた手順どおり 1 回だけ引く。
//
// クエリ集はノートの内容から作るのでツールとは別に置き、パスを必ず渡す。
// 相対パスは作業ディレクトリから解決する。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Warashi/brainium-tools/internal/notesearch"
	"github.com/Warashi/brainium-tools/internal/notesearch/anchor"
	"github.com/Warashi/brainium-tools/internal/notesearch/eval"
	"github.com/Warashi/brainium-tools/internal/notesearch/tokenize"
	"github.com/Warashi/brainium-tools/internal/repo"
)

// candidate は基準と比べる設定。名前は結果表にそのまま出す。
type candidate struct {
	name string
	p    notesearch.Params
	// bilingual なら問いの列ではなく日英併記版の列で引く
	bilingual bool
}

func (c candidate) rank() func(*notesearch.Index, eval.Query, notesearch.Params) int {
	if c.bilingual {
		return eval.RankBilingual
	}
	return eval.Rank
}

// candidates は基準の BM25 と、README に記録のある候補。新しい候補を測る
// ときはここに 1 行足す。
var candidates = []candidate{
	{"bm25", notesearch.Params{}, false},
	{"prf(5,40,0.6)", notesearch.Params{PRFDocs: 5, PRFTerms: 40, PRFAlpha: 0.6}, false},
	{"anchor(w=3)", notesearch.Params{AnchorW: 3}, false},
	{"translit(d=1,w=1.0)", notesearch.Params{TranslitDist: 1, TranslitW: 1.0}, false},
	{"translit(d=0,w=1.0)", notesearch.Params{TranslitDist: 0, TranslitW: 1.0}, false},
	{"translit(d=1,loose=4,w=1.0)", notesearch.Params{TranslitDist: 1, TranslitLoose: 4, TranslitW: 1.0}, false},
	{"bilingual", notesearch.Params{}, true},

	// 診断用。研究計画ノートは LSI 系統を閉じているが、「橋渡しした先の
	// 語が弁別しない」が LSI でも起きるかを見るためだけに並べる。採用の
	// 判断には使わない
	{"lsi(20,1.0)", notesearch.Params{LSIRank: 20, LSILambda: 1.0}, false},
	{"lsi+anchor", notesearch.Params{LSIRank: 20, LSILambda: 1.0, AnchorW: 3}, false},
	{"lsi+translit", notesearch.Params{LSIRank: 20, LSILambda: 1.0, TranslitDist: 0, TranslitW: 1.0}, false},
	{"lsi+bilingual", notesearch.Params{LSIRank: 20, LSILambda: 1.0}, true},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "notesearch-eval:", err)
		os.Exit(2)
	}
}

func run() error {
	queriesPath := flag.String("queries", "", "評価クエリ集の TSV のパス(必須)")
	split := flag.String("split", "train", "train / test / all")
	diff := flag.Bool("diff", false, "基準と候補で順位が動いたクエリだけ出す")
	ranks := flag.Bool("ranks", false, "全クエリの正解ノートの順位を候補ごとに並べる")
	gapOnly := flag.Bool("gap", false, "-ranks を gap の行だけに絞る")
	maxGram := flag.Int("maxgram", tokenize.MaxGram, "日本語 n-gram の上限。README の maxGram 12 の行を再現するときに使う")
	flag.Parse()
	tokenize.MaxGram = *maxGram
	if *queriesPath == "" {
		return fmt.Errorf("-queries でクエリ集のパスを指定する")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := repo.Root(cwd)
	if err != nil {
		return err
	}
	f, err := os.Open(*queriesPath)
	if err != nil {
		return err
	}
	defer f.Close()
	all, err := eval.Load(f)
	if err != nil {
		return err
	}
	qs := eval.Select(all, *split)

	// リンク文はリポジトリ全体から集める。索引の対象は notes/ 直下のまま
	anchors, err := anchor.Collect(os.DirFS(root))
	if err != nil {
		return err
	}
	ix, err := notesearch.Build(os.DirFS(filepath.Join(root, "notes")), anchors)
	if err != nil {
		return err
	}

	if *ranks {
		fmt.Printf("%-46s", "")
		for _, c := range candidates {
			fmt.Printf(" %12s", c.name)
		}
		fmt.Println("  gap")
		for _, q := range qs {
			if *gapOnly && !q.Gap {
				continue
			}
			fmt.Printf("%-46s", q.Want)
			for _, c := range candidates {
				fmt.Printf(" %12d", c.rank()(ix, q, c.p))
			}
			if q.Gap {
				fmt.Print("  gap")
			}
			fmt.Println()
		}
		return nil
	}

	if *diff {
		base := candidates[0]
		for _, c := range candidates[1:] {
			fmt.Printf("== %s -> %s\n", base.name, c.name)
			for _, q := range qs {
				a, b := base.rank()(ix, q, base.p), c.rank()(ix, q, c.p)
				if a == b {
					continue
				}
				g := ""
				if q.Gap {
					g = "gap"
				}
				fmt.Printf("%-42s %3d -> %3d  %s\n", q.Want, a, b, g)
			}
		}
		return nil
	}
	for _, c := range candidates {
		a, g := eval.Evaluate(ix, qs, c.p, c.rank())
		fmt.Printf("%-16s 全体 %s | gap %s\n", c.name, a, g)
	}
	return nil
}
