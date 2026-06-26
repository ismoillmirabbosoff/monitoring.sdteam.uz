package onlinepbx

import "strings"

// CompanyByQueue navbat (fifo tr1) bo'yicha kompaniyani aniqlaydi.
//   5201 -> salesdoc, 5202 -> ibox
func CompanyByQueue(queue string) string {
	switch strings.TrimSpace(queue) {
	case "5201":
		return "salesdoc"
	case "5202":
		return "ibox"
	}
	return ""
}

// CompanyByGateway gateway (trunk raqami) bo'yicha kompaniyani aniqlaydi.
//   712* -> salesdoc (Sales doctor), 781* -> ibox
func CompanyByGateway(gateway string) string {
	g := strings.TrimSpace(gateway)
	switch {
	case strings.HasPrefix(g, "712"):
		return "salesdoc"
	case strings.HasPrefix(g, "781"):
		return "ibox"
	}
	return ""
}
