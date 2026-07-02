package api

import (
	"net/http"
	"regexp"
	"time"

	"github.com/salesdoc/monitoring-api/internal/onlinepbx"
	"github.com/salesdoc/monitoring-api/internal/store"
)

var extRe = regexp.MustCompile(`^[1-4][0-9]{2,3}$`)

func isExt(s string) bool { return extRe.MatchString(s) }

// operatorExt qo'ng'iroqning operator extension'ini qaytaradi
// (chiquvchi → qo'ng'iroq qiluvchi, kiruvchi → manzil), aks holda "".
func operatorExt(c store.CallRow) string {
	if c.Direction == "outbound" {
		if isExt(c.CallerIDNumber) {
			return c.CallerIDNumber
		}
		return ""
	}
	if isExt(c.DestinationNumber) {
		return c.DestinationNumber
	}
	return ""
}

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

// SurveyStat anketa qoplanishi: javob berilgan (suhbatli) qo'ng'iroqlar bo'yicha.
type SurveyStat struct {
	Filled   int     `json:"filled"`   // anketa to'ldirilgan qo'ng'iroqlar
	Unfilled int     `json:"unfilled"` // anketa to'ldirilmagan qo'ng'iroqlar
	Total    int     `json:"total"`    // javob berilgan qo'ng'iroqlar (filled + unfilled)
	Pct      float64 `json:"pct"`      // filled / total * 100
}

func (s *SurveyStat) add(filled bool) {
	s.Total++
	if filled {
		s.Filled++
	} else {
		s.Unfilled++
	}
}
func (s *SurveyStat) finish() {
	if s.Total > 0 {
		s.Pct = float64(s.Filled) / float64(s.Total) * 100
	}
}

type opStat struct {
	Ext            string  `json:"ext"`
	Incoming       int     `json:"incoming"`
	IncTime        int64   `json:"incoming_time"`
	Outgoing       int     `json:"outgoing"`
	OutTime        int64   `json:"outgoing_time"`
	Total          int64   `json:"total_time"`
	Missed         int     `json:"missed"`
	SurveyFilled   int     `json:"survey_filled"`   // to'ldirilgan anketa (bugun)
	SurveyUnfilled int     `json:"survey_unfilled"` // to'ldirilmagan anketa (bugun)
	Servers        int     `json:"servers"`         // biriktirilgan serverlar soni
	Pct            float64 `json:"pct"`
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
	// Kompaniyani aniqlash uchun ext→kompaniya (kiruvchi gateway fallback) va server sonlari.
	extCompany, _ := s.store.ExtCompanyMap(r.Context())
	serverCount, _ := s.store.ServerCountByExt(r.Context())

	var incToday, incWeek, incMonth Period
	var outToday, outWeek, outMonth Period
	var survToday, survWeek, survMonth SurveyStat
	ops := map[string]*opStat{}
	grandTotal := 0

	for _, c := range calls {
		ext := operatorExt(c)
		// Kompaniya: avval gateway, keyin operator ext bo'yicha (kiruvchi qo'ng'iroqlar uchun).
		comp := onlinepbx.CompanyByGateway(c.Gateway)
		if comp == "" && ext != "" {
			comp = extCompany[ext]
		}
		if company != "" && comp != company {
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
			survMonth.add(filled)
			if inWeek {
				survWeek.add(filled)
			}
			if inToday {
				survToday.add(filled)
			}
		}

		// operator jadvali (bugun)
		if inToday && ext != "" {
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
			// anketa: faqat javob berilgan (suhbatli) qo'ng'iroqlar hisobga olinadi
			if answered {
				if respByCall[c.UUID] {
					o.SurveyFilled++
				} else {
					o.SurveyUnfilled++
				}
			}
			grandTotal++
		}
	}

	for _, p := range []*Period{&incToday, &incWeek, &incMonth, &outToday, &outWeek, &outMonth} {
		p.finish()
	}
	survToday.finish()
	survWeek.finish()
	survMonth.finish()

	opList := make([]opStat, 0, len(ops))
	for _, o := range ops {
		if grandTotal > 0 {
			o.Pct = float64(o.Incoming+o.Outgoing) / float64(grandTotal) * 100
		}
		o.Servers = serverCount[o.Ext]
		opList = append(opList, *o)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"incoming":  map[string]Period{"today": incToday, "week": incWeek, "month": incMonth},
		"outgoing":  map[string]Period{"today": outToday, "week": outWeek, "month": outMonth},
		"surveys":   map[string]SurveyStat{"today": survToday, "week": survWeek, "month": survMonth},
		"operators": opList,
	})
}
