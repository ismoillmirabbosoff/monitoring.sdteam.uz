package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
)

var extRe = regexp.MustCompile(`^[1-4][0-9]{2,3}$`)

func isExt(s string) bool { return extRe.MatchString(s) }

type Period struct {
	Total    int   `json:"total"`
	Answered int   `json:"answered"` // kiruvchi: javob berilgan; chiquvchi: muvaffaqiyatli
	Missed   int   `json:"missed"`   // kiruvchi: o'tkazib yuborilgan; chiquvchi: dozvonilmagan
	Talk     int64 `json:"talk"`     // umumiy suhbat (soniya)
	Avg      int64 `json:"avg"`      // o'rtacha suhbat (soniya)
}

func (p *Period) add(answered bool, talk int64) {
	p.Total++
	if answered {
		p.Answered++
		p.Talk += talk
	} else {
		p.Missed++
	}
}
func (p *Period) finish() {
	if p.Answered > 0 {
		p.Avg = p.Talk / int64(p.Answered)
	}
}

type opStat struct {
	Ext      string `json:"ext"`
	Incoming int    `json:"incoming"`
	IncTime  int64  `json:"incoming_time"`
	Outgoing int    `json:"outgoing"`
	OutTime  int64  `json:"outgoing_time"`
	Total    int64  `json:"total_time"`
	Missed   int    `json:"missed"`
	Pct      float64 `json:"pct"`
}

// GET /api/monitoring/stats?company=salesdoc|ibox
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	company := r.URL.Query().Get("company")

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := todayStart.AddDate(0, 0, -6)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	calls, err := s.store.CallsByRange(r.Context(), "", monthStart.Unix(), now.Unix())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp, _ := s.store.ResponsesInRange(r.Context(), monthStart.Unix(), now.Unix())
	respByCall := map[string]bool{}
	for _, x := range resp {
		respByCall[x.CallUUID] = true
	}

	var incToday, incWeek, incMonth Period
	var outToday, outWeek, outMonth Period
	var survToday, survWeek, survMonth [2]int // [filled, answeredTotal]
	ops := map[string]*opStat{}
	grandTotal := 0

	for _, c := range calls {
		if company != "" && onlinepbx.CompanyByGateway(c.Gateway) != company {
			continue
		}
		answered := c.UserTalkTime > 0
		inWeek := c.StartStamp >= weekStart.Unix()
		inToday := c.StartStamp >= todayStart.Unix()

		if c.Direction == "outbound" {
			outMonth.add(answered, c.UserTalkTime)
			if inWeek {
				outWeek.add(answered, c.UserTalkTime)
			}
			if inToday {
				outToday.add(answered, c.UserTalkTime)
			}
		} else {
			incMonth.add(answered, c.UserTalkTime)
			if inWeek {
				incWeek.add(answered, c.UserTalkTime)
			}
			if inToday {
				incToday.add(answered, c.UserTalkTime)
			}
		}

		// anketa qoplanishi (javob berilgan qo'ng'iroqlar bo'yicha)
		if answered {
			filled := respByCall[c.UUID]
			survMonth[1]++
			if filled {
				survMonth[0]++
			}
			if inWeek {
				survWeek[1]++
				if filled {
					survWeek[0]++
				}
			}
			if inToday {
				survToday[1]++
				if filled {
					survToday[0]++
				}
			}
		}

		// operator jadvali (bugun)
		if inToday {
			ext := ""
			if c.Direction == "outbound" {
				if isExt(c.CallerIDNumber) {
					ext = c.CallerIDNumber
				}
			} else if isExt(c.DestinationNumber) {
				ext = c.DestinationNumber
			}
			if ext != "" {
				o := ops[ext]
				if o == nil {
					o = &opStat{Ext: ext}
					ops[ext] = o
				}
				if c.Direction == "outbound" {
					o.Outgoing++
					o.OutTime += c.UserTalkTime
				} else {
					o.Incoming++
					o.IncTime += c.UserTalkTime
					if !answered {
						o.Missed++
					}
				}
				o.Total += c.UserTalkTime
				grandTotal++
			}
		}
	}

	for _, p := range []*Period{&incToday, &incWeek, &incMonth, &outToday, &outWeek, &outMonth} {
		p.finish()
	}
	opList := make([]opStat, 0, len(ops))
	for _, o := range ops {
		if grandTotal > 0 {
			o.Pct = float64(o.Incoming+o.Outgoing) / float64(grandTotal) * 100
		}
		opList = append(opList, *o)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"incoming": map[string]Period{"today": incToday, "week": incWeek, "month": incMonth},
		"outgoing": map[string]Period{"today": outToday, "week": outWeek, "month": outMonth},
		"surveys": map[string][2]int{"today": survToday, "week": survWeek, "month": survMonth},
		"operators": opList,
	})
}
