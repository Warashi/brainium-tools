package orgedit

// tasksOrg は brainium の tasks.org を縮めた差分テストの入力。
const tasksOrg = `:PROPERTIES:
:ID:       file-tasks
:END:
#+title: Tasks
#+filetags: :tasks:
#+startup: logdone logdrawer
#+todo: TODO(t!) WIP(w!) | DONE(d!) CANCELLED(c!)

* Next
:PROPERTIES:
:CATEGORY: Next
:ID:       sec-next
:END:
** TODO 和文の task :handoff:
:PROPERTIES:
:ID:       t-tag
:CAPTURED: [2026-11-12 Thu 23:19]
:END:

本文 a

** TODO scheduled task
SCHEDULED: <2026-11-20 Fri>
:PROPERTIES:
:ID:       t-sched
:END:

body b

** WIP logged task                                                   :weekend:
:PROPERTIES:
:ID:       t-logged
:CURRENT_STEP: 何かをする
:END:
:LOGBOOK:
- State "WIP"        from "TODO"       [2026-11-01 Sun 10:00]
:END:
body c
*** child of logged
:PROPERTIES:
:ID:       t-child
:END:
** DONE closed task
CLOSED: [2026-11-13 Fri 06:17]
:PROPERTIES:
:ID:       t-done
:END:
:LOGBOOK:
- State "DONE"       from "TODO"       [2026-11-13 Fri 06:17]
:END:

* Someday
:PROPERTIES:
:CATEGORY: Someday
:ID:       sec-someday
:END:
** TODO last task
:PROPERTIES:
:ID:       t-last
:END:`

// inboxOrg は brainium の inbox.org を縮めた差分テストの入力。
const inboxOrg = `:PROPERTIES:
:ID:       file-inbox
:END:
#+title: Inbox

# org-capture の投入先。

* 新しい capture
:PROPERTIES:
:ID:       c-new
:CAPTURED: [2026-11-16 Mon 20:26]
:END:

本文 1

** 子の見出し :tag:

* 古い capture
:PROPERTIES:
:ID:       c-old
:CAPTURED: [2026-11-13 Fri 06:22]
:END:

本文 2
`
