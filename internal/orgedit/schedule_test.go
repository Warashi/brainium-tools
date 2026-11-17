package orgedit

import (
	"strings"
	"testing"
)

func TestScheduleMatchesEmacs(t *testing.T) {
	repeating := strings.Replace(tasksOrg, "SCHEDULED: <2026-11-20 Fri>", "SCHEDULED: <2026-11-17 Tue 09:00-10:00 .+1d>", 1)
	withDeadline := strings.Replace(tasksOrg, "SCHEDULED: <2026-11-20 Fri>", "DEADLINE: <2026-11-25 Wed> SCHEDULED: <2026-11-20 Fri>", 1)
	scheduledFirst := strings.Replace(tasksOrg, "SCHEDULED: <2026-11-20 Fri>", "SCHEDULED: <2026-11-20 Fri> DEADLINE: <2026-11-25 Wed>", 1)
	closedAndScheduled := strings.Replace(tasksOrg, "CLOSED: [2026-11-13 Fri 06:17]", "CLOSED: [2026-11-13 Fri 06:17] SCHEDULED: <2026-11-12 Thu>", 1)
	deadlineOnly := strings.Replace(tasksOrg, "SCHEDULED: <2026-11-20 Fri>", "DEADLINE: <2026-11-25 Wed>", 1)
	schedule := func(name, src, id, when string) emacsCase {
		arg := `nil "` + when + `"`
		if when == "" {
			arg = `'(4)`
		}
		return emacsCase{
			name:  name,
			files: map[string]string{"tasks.org": src},
			run: func(root string, _ Env) error {
				_, err := Schedule(root, id, when)
				return err
			},
			elisp: `(edit-goto-id "tasks.org" "` + id + `") (org-schedule ` + arg + `)`,
		}
	}
	runEmacsCases(t, []emacsCase{
		schedule("計画行の無い見出しに日付を足す", tasksOrg, "t-tag", "2026-11-18"),
		schedule("時刻を足す", tasksOrg, "t-tag", "2026-11-18 9:05"),
		schedule("時刻の範囲を足す", tasksOrg, "t-tag", "2026-11-18 09:00-10:30"),
		schedule("子のある見出しに足す", tasksOrg, "t-logged", "2026-11-18"),
		schedule("CLOSED のある見出しに足す", tasksOrg, "t-done", "2026-11-18"),
		schedule("替える", tasksOrg, "t-sched", "2026-11-18 09:00-10:00"),
		schedule("repeater を保って替える", repeating, "t-sched", "2026-11-18 09:00-10:00"),
		schedule("DEADLINE の後の SCHEDULED を替える", withDeadline, "t-sched", "2026-11-18"),
		schedule("DEADLINE だけの計画行に足す", deadlineOnly, "t-sched", "2026-11-18 13:00"),
		schedule("外す", tasksOrg, "t-sched", ""),
		schedule("DEADLINE を残して外す", withDeadline, "t-sched", ""),
		schedule("DEADLINE の前の SCHEDULED を替える", scheduledFirst, "t-sched", "2026-11-18"),
		schedule("DEADLINE の前の SCHEDULED を外す", scheduledFirst, "t-sched", ""),
		schedule("CLOSED を残して外す", closedAndScheduled, "t-done", ""),
	})
}

func TestScheduleRejectsLogReschedule(t *testing.T) {
	src := strings.Replace(tasksOrg, "#+startup: logdone logdrawer", "#+startup: logdone logdrawer logreschedule", 1)
	root := writeRepo(t, t.TempDir(), map[string]string{"tasks.org": src})
	if _, err := Schedule(root, "t-sched", "2026-11-18"); err == nil {
		t.Error("logreschedule のファイルで Schedule が通った")
	}
}

func TestScheduleRejectsMalformedTimes(t *testing.T) {
	for _, when := range []string{"2026-11-31", "2026/11/18", "2026-11-18 25:00", "2026-11-18 9:00-", "2026-11-18 +1d", "tomorrow"} {
		if _, err := Schedule(t.TempDir(), "t-sched", when); err == nil {
			t.Errorf("Schedule(%q) が通った", when)
		}
	}
}
