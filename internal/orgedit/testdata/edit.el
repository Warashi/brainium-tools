;;; edit.el --- orgedit の正解を Emacs の org で作る -*- lexical-binding: t -*-

;; 差分テストが --eval で渡す操作の前に読み込む。時刻を固定し、ID の見出
;; しへ移る補助を用意する。操作の後は edit-finish で全バッファを保存する。

(require 'org)
(require 'org-id)
(require 'org-archive)
(require 'org-refile)
(require 'org-capture)

(setq make-backup-files nil
      create-lockfiles nil
      auto-save-default nil
      org-id-locations-file (expand-file-name "org-id-locations" temporary-file-directory))

(defvar edit-time (encode-time (list 0 8 21 16 11 2026 nil nil t))
  "操作の時刻。Go 側の Env.Now と揃える。")

(advice-add 'current-time :override (lambda () edit-time))
(advice-add 'format-time-string :around
            (lambda (orig format &optional time zone)
              (funcall orig format (or time edit-time) (or zone t))))

(defvar edit-ids nil
  "org-id-new が返す ID の列。Go 側の Env.NewID と揃える。")

(advice-add 'org-id-new :override (lambda (&optional _prefix) (pop edit-ids)))

(defun edit-goto-id (file id)
  "FILE を開き、:ID: が ID の見出しへ移る。"
  (find-file file)
  (goto-char (point-min))
  (unless (re-search-forward
           (concat "^[ \t]*:ID:[ \t]+" (regexp-quote id) "[ \t]*$") nil t)
    (error "ID %s が %s に無い" id file))
  (org-back-to-heading t))

(defun edit-todo (state)
  "org-todo で STATE にし、記録が予約されていれば org-add-log-note で書く。
batch では post-command-hook が走らないので、ここで呼ぶ。"
  (org-todo state)
  (when (memq 'org-add-log-note post-command-hook)
    (org-add-log-note)))

(defun edit-finish ()
  "開いた全バッファを保存する。"
  (save-some-buffers t))

;;; edit.el ends here
