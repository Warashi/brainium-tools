package orgedit

import "github.com/Warashi/brainium-tools/internal/org"

// SetProperty は :ID: が id の見出しの key を value にする。value が空な
// ら key を消す。
func SetProperty(root string, id, key, value string) ([]string, error) {
	return Update(root, func(f *Files) error {
		e, err := Find(f, id)
		if err != nil {
			return err
		}
		if value == "" {
			f.Write(e.Path, DeleteProperty(e.Src, e.Headline, key))
			return nil
		}
		src, err := PutProperty(e.Src, e.Headline, key, value)
		if err != nil {
			return err
		}
		f.Write(e.Path, src)
		return nil
	})
}

// reparse は src を解析し直し、:ID: が id の見出しを返す。
func reparse(src, id string) (*org.Node, *org.Node, error) {
	root := org.Parse(src)
	h, err := FindIn(root, id)
	return root, h, err
}
