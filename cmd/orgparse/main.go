// Command orgparse は org ファイルを brainium-tools の parser で解析した
// 木を出す。
//
//	orgparse [-lines A[-B]] file
//
// 書いた記法が意図どおりに解釈されたかを agent が確かめるためのもの。
// -lines を渡すと、その行と重なる節点だけを祖先と一緒に出す。
//
// 1 行に 1 節点を、親子を 2 桁の字下げで表して出す。各行は種類と範囲
// (行:桁-行:桁、桁は 1 始まりの文字数で終端は含まない)に続けて、
// 見出しの深さやリンクの行き先などの属性を付ける。object の間にある素の
// 文字列は text として、原文の抜粋は Go の %q で引用して出す。
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/Warashi/brainium-tools/internal/org"
)

func main() {
	if err := execute(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "orgparse:", err)
		os.Exit(2)
	}
}

func execute(args []string, out io.Writer) error {
	flags := flag.NewFlagSet("orgparse", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	lines := flags.String("lines", "", "出す行の範囲。A か A-B (1 始まり、両端を含む)")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			flags.SetOutput(out)
			fmt.Fprintln(out, "使い方: orgparse [-lines A[-B]] file")
			flags.PrintDefaults()
			return nil
		}
		return err
	}
	if flags.NArg() != 1 {
		return fmt.Errorf("ファイルを 1 つ渡す")
	}
	from, to, err := parseLines(*lines)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(flags.Arg(0))
	if err != nil {
		return err
	}
	src := string(data)
	_, err = io.WriteString(out, render(src, org.Parse(src), from, to))
	return err
}

// parseLines は -lines の値を 1 始まりで両端を含む行の範囲にする。空なら
// 全体を返す。
func parseLines(s string) (from, to int, err error) {
	if s == "" {
		return 1, math.MaxInt, nil
	}
	a, b, isRange := strings.Cut(s, "-")
	from, err = strconv.Atoi(a)
	if err != nil || from < 1 {
		return 0, 0, fmt.Errorf("-lines %q: 行は 1 以上の整数", s)
	}
	if !isRange {
		return from, from, nil
	}
	to, err = strconv.Atoi(b)
	if err != nil || to < from {
		return 0, 0, fmt.Errorf("-lines %q: 終わりの行は始まりの行以上の整数", s)
	}
	return from, to, nil
}
