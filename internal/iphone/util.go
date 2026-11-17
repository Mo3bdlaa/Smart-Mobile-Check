package iphone

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

func str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case []byte:
		return strings.TrimSpace(string(t))
	case bool:
		if t {
			return "true"
		}
		return "false"
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func num(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int8:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint:
		return float64(t), true
	case uint8:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	}
	return 0, false
}

func boolv(v any) (bool, bool) {
	switch t := v.(type) {
	case bool:
		return t, true
	case string:
		switch strings.ToLower(t) {
		case "true", "yes", "1":
			return true, true
		case "false", "no", "0":
			return false, true
		}
	}
	if f, ok := num(v); ok {
		return f != 0, true
	}
	return false, false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func sub(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

// versionParts turns "18.6.2" into [18 6 2].
func versionParts(v string) []int {
	var out []int
	for _, p := range strings.Split(strings.TrimSpace(v), ".") {
		n, err := strconv.Atoi(strings.TrimFunc(p, func(r rune) bool { return r < '0' || r > '9' }))
		if err != nil {
			break
		}
		out = append(out, n)
	}
	return out
}

// compareVersions returns -1, 0, 1 comparing only as many parts as b has.
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < len(pb); i++ {
		x := 0
		if i < len(pa) {
			x = pa[i]
		}
		if x < pb[i] {
			return -1
		}
		if x > pb[i] {
			return 1
		}
	}
	return 0
}

func luhnValid(s string) bool {
	if len(s) < 2 {
		return false
	}
	sum := 0
	double := false
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if c < '0' || c > '9' {
			return false
		}
		d := int(c - '0')
		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
		double = !double
	}
	return sum%10 == 0
}

func humanBytes(b float64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	i := 0
	for b >= 1000 && i < len(units)-1 {
		b /= 1000
		i++
	}
	if i == 0 {
		return fmt.Sprintf("%.0f %s", b, units[i])
	}
	return fmt.Sprintf("%.1f %s", b, units[i])
}

func gbLabel(gb int) string {
	if gb >= 1024 && gb%1024 == 0 {
		return fmt.Sprintf("%d TB", gb/1024)
	}
	return fmt.Sprintf("%d GB", gb)
}

func joinInts(v []int, suffix string) string {
	var s []string
	for _, x := range v {
		s = append(s, fmt.Sprintf("%d%s", x, suffix))
	}
	return strings.Join(s, " / ")
}
