;;; org-element.el --- 差分テストの正解として org-element の木を JSON で出す -*- lexical-binding: t -*-

;; 使い方: emacs -Q --batch -l org-element.el GRANULARITY FILE...
;; GRANULARITY は org-element-parse-buffer に渡す粒度(element か object)。
;; 標準出力に {"org_version": ..., "files": {FILE: TREE, ...}} を出す。
;; TREE の位置は 0 始まりの文字(code point)単位。plain-text は出さない。

(require 'org)
(require 'org-element)
(require 'org-id)
(require 'json)

(setq org-element-use-cache nil)

(defun org-diff--pos (pos)
  (and pos (1- pos)))

(defun org-diff--node (node)
  (let ((children (delq nil (mapcar #'org-diff--node (org-element-contents node)))))
    (unless (eq (org-element-type node) 'plain-text)
      (list :type (symbol-name (org-element-type node))
            :begin (org-diff--pos (org-element-begin node))
            :end (org-diff--pos (org-element-end node))
            :contents_begin (or (org-diff--pos (org-element-contents-begin node)) :null)
            :contents_end (or (org-diff--pos (org-element-contents-end node)) :null)
            :post_blank (or (org-element-post-blank node) 0)
            :children (vconcat children)))))

(defun org-diff--parse (file granularity)
  (with-temp-buffer
    (insert-file-contents file)
    (let ((org-inhibit-startup t))
      (org-mode))
    (org-diff--node (org-element-parse-buffer granularity))))

(let ((granularity (intern (pop command-line-args-left)))
      (files nil))
  (dolist (file command-line-args-left)
    (push (cons (intern file) (org-diff--parse file granularity)) files))
  (setq command-line-args-left nil)
  (princ (json-serialize
          (list :org_version org-version
                :files (let (plist)
                         (dolist (f files plist)
                           (setq plist (append (list (car f) (cdr f)) plist))))))))
