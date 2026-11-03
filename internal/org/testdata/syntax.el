;;; syntax.el --- org-mode の構文表による文字の分類を範囲で出す -*- lexical-binding: t -*-

;; 使い方: emacs -Q --batch -l syntax.el
;; 標準出力に {"space": [[LO, HI], ...], "word": ..., ...} を出す。
;; latin_word は、直後のラテン文字を単語の始まり(\<)にしない文字。
;; latex_fragment_end は、$ で囲む LaTeX の断片の直後に置ける文字の
;; うち構文の分類で決まるもの。

(require 'org)
(require 'json)

(defun syntax--ranges (pred)
  (let (out start prev)
    (dotimes (c #x110000)
      (if (and (not (<= #xd800 c #xdfff)) (funcall pred c))
          (progn (unless start (setq start c)) (setq prev c))
        (when start (push (vector start prev) out) (setq start nil))))
    (when start (push (vector start prev) out))
    (vconcat (nreverse out))))

(with-temp-buffer
  (let ((org-inhibit-startup t))
    (org-mode))
  (princ (json-serialize
          (list :space (syntax--ranges (lambda (c) (eq (char-syntax c) ?\s)))
                :word (syntax--ranges (lambda (c) (eq (char-syntax c) ?w)))
                :latin_word (syntax--ranges
                             (lambda (c)
                               (not (string-match-p "\\`[^z-a]\\<" (string c ?h)))))
                :latex_fragment_end (syntax--ranges
                                     (lambda (c)
                                       (string-match-p "\\`\\(?:\\s.\\|\\s-\\|\\s(\\|\\s)\\|\\s\"\\)"
                                                       (string c))))))))
