package org

import "regexp"

const TypeTimestamp Type = "timestamp"

var (
	// timestampRe は org-element--timestamp-regexp。開きと閉じの括弧の
	// 種類が違っても org は timestamp とみなす。
	timestampRe = regexp.MustCompile(`^(?:[\[<][0-9]{4}-[0-9]{2}-[0-9]{2}(?: .*?)?[\]>]|<%%\([^>\n]+\)[^\n>]*>)`)
	// timestampRawRe は org-element--timestamp-raw-value-regexp で、
	// timestamp の終わりを決める。
	timestampRawRe = regexp.MustCompile(`^[<\[](?:%%)?.*?[\]>](?:--[\[<][0-9]{4}-[0-9]{2}-[0-9]{2}(?: .*?)?[\]>])?`)
)

// timestamp は pos から始まる timestamp を解析する。
func (s objectSpan) timestamp(pos int) *Node {
	text := s.p.src[pos:s.end]
	if !timestampRe.MatchString(text) {
		return nil
	}
	n := &Node{Type: TypeTimestamp, Begin: pos, ContentsBegin: -1, ContentsEnd: -1}
	n.PostBlank, n.End = s.skipSpaces(pos + len(timestampRawRe.FindString(text)))
	return n
}
