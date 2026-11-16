;;; edit.el --- orgedit の正解を Emacs の org で作る -*- lexical-binding: t -*-

;; 差分テストが渡す操作の前に読み込む。時刻を固定し、ID の見出
;; しへ移る補助を用意する。操作の後は edit-finish で全バッファを保存する。

(require 'org)
(require 'org-id)
(require 'org-archive)
(require 'org-refile)
(require 'org-capture)


;; nix のサンドボックスはロケールが UTF-8 でなく、保存時に文字コードを尋ね
;; て止まるので固定する
(set-language-environment "UTF-8")
(prefer-coding-system 'utf-8-unix)
(setq-default buffer-file-coding-system 'utf-8-unix)
(setq file-name-coding-system 'utf-8)

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

(defun edit-id-pos (file id)
  "FILE の :ID: が ID の見出しの位置を、今のバッファを変えずに返す。"
  (with-current-buffer (find-file-noselect file)
    (save-excursion
      (goto-char (point-min))
      (unless (re-search-forward
               (concat "^[ \t]*:ID:[ \t]+" (regexp-quote id) "[ \t]*$") nil t)
        (error "ID %s が %s に無い" id file))
      (org-back-to-heading t)
      (point))))

(defun edit-refile (file id target-file &optional target-id first)
  "FILE の ID の subtree を org-refile で TARGET-FILE へ移す。
TARGET-ID があればその見出しの子に、無ければファイルの深さ 1 に移す。
FIRST なら末尾ではなく先頭に移す。"
  (let ((pos (and target-id (edit-id-pos target-file target-id)))
        (org-reverse-note-order first))
    (edit-goto-id file id)
    (org-refile nil nil (list "target" (expand-file-name target-file) nil pos))))

(defun edit-move-before (file id sibling-file sibling-id)
  "FILE の ID の subtree を切り取り、SIBLING-ID の見出しの手前に同じ深さで貼る。
org-refile と同じく、貼った見出しのタグを揃え直す。"
  (edit-goto-id file id)
  (org-cut-subtree)
  ;; set-buffer の後は default-directory が変わるので、位置を先に求める
  (let ((pos (edit-id-pos sibling-file sibling-id)))
    (set-buffer (find-file-noselect sibling-file))
    (goto-char pos))
  (org-paste-subtree (org-outline-level))
  (org-align-tags))

(defun edit-capture (title body)
  "inbox.org の先頭に、orgedit capture と同じ template で capture する。"
  (let ((org-capture-templates
         `(("x" "inbox" entry (file ,(expand-file-name "inbox.org"))
            ,(concat "* " title "\n:PROPERTIES:\n:ID:       %(org-id-new)\n"
                     ":CAPTURED: %U\n:END:\n"
                     (if (string= body "") "" (concat "\n" body "\n")))
            :prepend t :empty-lines 1 :immediate-finish t))))
    (org-capture nil "x")))

(defun edit-archive (file id)
  "FILE の ID の subtree を org-archive-subtree で archive する。"
  (edit-goto-id file id)
  (org-archive-subtree))

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
