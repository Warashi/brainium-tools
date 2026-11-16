package orgedit

import (
	"strconv"
	"strings"
	"testing"
)

func TestSetTagsMatchesEmacs(t *testing.T) {
	tags := func(name, id string, tags []string) emacsCase {
		var quoted []string
		for _, t := range tags {
			quoted = append(quoted, strconv.Quote(t))
		}
		return emacsCase{
			name:  name,
			files: map[string]string{"tasks.org": tasksOrg},
			run: func(root string, _ Env) error {
				_, err := SetTags(root, id, tags)
				return err
			},
			elisp: `(edit-set-tags "tasks.org" "` + id + `" (list ` + strings.Join(quoted, " ") + `))`,
		}
	}
	runEmacsCases(t, []emacsCase{
		tags("タグの無い見出しに付ける", "t-sched", []string{"handoff"}),
		tags("揃っていないタグを替える", "t-tag", []string{"handoff", "weekend"}),
		tags("タグを外す", "t-logged", nil),
		tags("和文の見出しに付ける", "t-tag", []string{"weekend"}),
	})
}
