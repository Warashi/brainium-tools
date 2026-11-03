;;; org-element.el --- 差分テストの正解として org-element の木を JSON で出す -*- lexical-binding: t -*-

;; 使い方: emacs -Q --batch -l org-element.el FILE...
;; 標準出力に {"org_version": ..., "files": {FILE: TREE, ...}} を出す。
;; TREE の位置は 0 始まりの文字(code point)単位。plain-text は出さない。
;; attrs には org-diff--attributes に挙げた型の属性だけを出す。

(require 'org)
(require 'org-element)
(require 'org-id)
(require 'json)

(setq org-element-use-cache nil)

(defconst org-diff--attributes
  '((headline :level)
    (node-property :key :value)
    (keyword :key :value))
  "型ごとに比べる属性。ツールが使う属性だけを挙げる。")

(defun org-diff--pos (pos)
  (and pos (1- pos)))

;; 属性は Go 側で文字列の map として読むので、数値も文字列にして出す
(defun org-diff--attr (value)
  (cond ((null value) :null)
        ((numberp value) (number-to-string value))
        (t value)))

(defun org-diff--attrs (node)
  (let (plist)
    (dolist (prop (alist-get (org-element-type node) org-diff--attributes) plist)
      (setq plist (append plist (list prop (org-diff--attr (org-element-property prop node))))))))

(defun org-diff--node (node)
  (let ((children (delq nil (mapcar #'org-diff--node (org-element-contents node)))))
    (unless (eq (org-element-type node) 'plain-text)
      (list :type (symbol-name (org-element-type node))
            :begin (org-diff--pos (org-element-begin node))
            :end (org-diff--pos (org-element-end node))
            :contents_begin (or (org-diff--pos (org-element-contents-begin node)) :null)
            :contents_end (or (org-diff--pos (org-element-contents-end node)) :null)
            :post_blank (or (org-element-post-blank node) 0)
            :attrs (org-diff--attrs node)
            :secondary (org-diff--secondary node)
            :children (vconcat children)))))

(defun org-diff--secondary (node)
  "headline の title や item の tag のような secondary string の object。"
  (let (objects)
    (dolist (prop (alist-get (org-element-type node) org-element-secondary-value-alist))
      (setq objects (append objects (delq nil (mapcar #'org-diff--node (org-element-property prop node))))))
    (vconcat objects)))

(defun org-diff--parse (file)
  (with-temp-buffer
    (insert-file-contents file)
    (let ((org-inhibit-startup t))
      (org-mode))
    (org-diff--node (org-element-parse-buffer 'object))))

(let ((files nil))
  (dolist (file command-line-args-left)
    (push (cons (intern file) (org-diff--parse file)) files))
  (setq command-line-args-left nil)
  ;; json-serialize は UTF-8 の unibyte 文字列を返し、そのまま princ する
  ;; と非 ASCII が raw byte として出力し直されて化ける
  (princ (decode-coding-string
          (json-serialize
           (list :org_version org-version
                 :files (let (plist)
                          (dolist (f files plist)
                            (setq plist (append (list (car f) (cdr f)) plist))))))
          'utf-8)))
