package dashboard

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// MaxDataFileBytes bounds one external snapshot and an atomic publication.
const MaxDataFileBytes = 1 << 20

// DataSource authorizes one caller-selected local file, never a command or URL.
// A zero StaleAfterMS uses the conservative five-second default.
type DataSource struct {
	Type         string `json:"type"`
	Path         string `json:"path"`
	StaleAfterMS int    `json:"stale_after_ms,omitempty"`
}

// ValueBinding references a presentation value, not an observed entity identity.
// Path is an RFC 6901 JSON pointer. Formats are a bounded allowlist, not expressions.
type ValueBinding struct {
	Source string `json:"source"`
	Path   string `json:"path"`
	Format string `json:"format,omitempty"`
	Prefix string `json:"prefix,omitempty"`
	Suffix string `json:"suffix,omitempty"`
}

type dataSnapshot struct {
	config  DataSource
	value   any
	stamp   time.Time
	problem string
}

var dataSourceName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)

func (s DataSource) staleAfter() time.Duration {
	if s.StaleAfterMS == 0 {
		return 5 * time.Second
	}
	return time.Duration(s.StaleAfterMS) * time.Millisecond
}

func pointerParts(pointer string) ([]string, error) {
	if len(pointer) > 1024 || Clean(pointer, 1024) != pointer {
		return nil, errors.New("value path must be a bounded printable JSON pointer")
	}
	if pointer == "" {
		return nil, nil
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil, errors.New("value path must be an RFC 6901 JSON pointer")
	}
	parts := strings.Split(pointer[1:], "/")
	if len(parts) > 32 {
		return nil, errors.New("value path exceeds 32 segments")
	}
	for i, part := range parts {
		var decoded strings.Builder
		for j := 0; j < len(part); j++ {
			if part[j] != '~' {
				decoded.WriteByte(part[j])
				continue
			}
			j++
			if j == len(part) || part[j] != '0' && part[j] != '1' {
				return nil, errors.New("invalid JSON pointer escape")
			}
			if part[j] == '0' {
				decoded.WriteByte('~')
			} else {
				decoded.WriteByte('/')
			}
		}
		parts[i] = decoded.String()
	}
	return parts, nil
}

func validateDataBindings(l *Layout) error {
	if len(l.DataSources) > 16 {
		return errors.New("data_sources limited to 16 local files")
	}
	for name, s := range l.DataSources {
		if !dataSourceName.MatchString(name) || s.Type != "json-file" || !filepath.IsAbs(s.Path) || len(s.Path) > 4096 || Clean(s.Path, 4096) != s.Path {
			return errors.New("data source requires a simple name, json-file type and absolute printable path")
		}
		if s.StaleAfterMS != 0 && (s.StaleAfterMS < 100 || s.StaleAfterMS > 3600000) {
			return errors.New("stale_after_ms must be 100..3600000 (zero uses 5000)")
		}
	}
	return walkValueBindings(l.Root, func(b *ValueBinding) error {
		if b == nil {
			return nil
		}
		if _, ok := l.DataSources[b.Source]; !ok {
			return errors.New("value binding references an undeclared data source")
		}
		if _, err := pointerParts(b.Path); err != nil {
			return err
		}
		switch b.Format {
		case "", "text", "number", "mm:ss", "percent", "bar", "deadline":
		default:
			return errors.New("value format must be text, number, mm:ss, percent, bar or deadline")
		}
		if len(b.Prefix) > 160 || len(b.Suffix) > 160 || Clean(b.Prefix, 160) != b.Prefix || Clean(b.Suffix, 160) != b.Suffix {
			return errors.New("value prefix/suffix must be printable and at most 160 bytes")
		}
		return nil
	})
}

// One walk owns validation and discovery for every presentation surface.
func walkValueBindings(p Panel, visit func(*ValueBinding) error) error {
	if p.Value != nil && p.Kind != "text" {
		return errors.New("value belongs to text panels; title_value decorates any leaf")
	}
	if p.TitleValue != nil && (p.Kind == "row" || p.Kind == "column") {
		return errors.New("title_value requires a leaf panel with a visible title")
	}
	for _, b := range []*ValueBinding{p.Value, p.TitleValue} {
		if err := visit(b); err != nil {
			return err
		}
	}
	if p.DAG != nil {
		if p.DAG.Node != nil {
			if err := visit(p.DAG.Node.LabelValue); err != nil {
				return err
			}
		}
		for _, n := range p.DAG.Nodes {
			if err := visit(n.LabelValue); err != nil {
				return err
			}
		}
		for _, e := range p.DAG.Edges {
			if err := visit(e.LabelValue); err != nil {
				return err
			}
		}
	}
	for _, child := range p.Children {
		if err := walkValueBindings(child, visit); err != nil {
			return err
		}
	}
	return nil
}

// Decode bounded JSON with exact numbers and duplicate-key/depth rejection.
func decodeExternalJSON(raw []byte) (any, error) {
	if len(raw) > MaxDataFileBytes {
		return nil, errors.New("external JSON exceeds 1 MiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	count := 0
	var value func(int) (any, error)
	value = func(depth int) (any, error) {
		count++
		if count > 16384 || depth > 32 {
			return nil, errors.New("external JSON exceeds 16384 values or depth 32")
		}
		token, err := d.Token()
		if err != nil {
			return nil, err
		}
		switch token {
		case json.Delim('{'):
			m := map[string]any{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, err
				}
				name, ok := key.(string)
				if !ok || len(name) > 1024 {
					return nil, errors.New("external JSON key exceeds limits")
				}
				if _, exists := m[name]; exists {
					return nil, errors.New("duplicate external JSON key")
				}
				m[name], err = value(depth + 1)
				if err != nil {
					return nil, err
				}
			}
			_, err = d.Token()
			return m, err
		case json.Delim('['):
			a := []any{}
			for d.More() {
				v, err := value(depth + 1)
				if err != nil {
					return nil, err
				}
				a = append(a, v)
			}
			_, err = d.Token()
			return a, err
		default:
			if _, invalid := token.(json.Delim); invalid {
				return nil, errors.New("invalid external JSON delimiter")
			}
			return token, nil
		}
	}
	v, err := value(0)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, errors.New("exactly one external JSON value required")
	}
	return v, nil
}

func externalStamp(value any, modified, now time.Time) (time.Time, error) {
	stamp := modified
	if object, ok := value.(map[string]any); ok {
		if raw, exists := object["updated_at"]; exists {
			text, ok := raw.(string)
			t, err := time.Parse(time.RFC3339Nano, text)
			if !ok || err != nil {
				return stamp, errors.New("invalid updated_at timestamp")
			}
			if t.After(now.Add(2 * time.Second)) {
				return stamp, errors.New("future updated_at timestamp (clock skew)")
			}
			if t.Before(stamp) {
				stamp = t
			}
		}
	}
	if modified.After(now.Add(2 * time.Second)) {
		return stamp, errors.New("future file timestamp (clock skew)")
	}
	return stamp, nil
}

func readExternalData(source DataSource, now time.Time) dataSnapshot {
	result := dataSnapshot{config: source}
	fail := func(message string) dataSnapshot { result.problem = message; return result }
	info, err := os.Lstat(source.Path)
	if err != nil {
		return fail("file unavailable")
	}
	if !info.Mode().IsRegular() || info.Size() > MaxDataFileBytes {
		return fail("expected a regular file of at most 1 MiB, not a link")
	}
	f, err := os.Open(source.Path)
	if err != nil {
		return fail("file unavailable")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) || opened.Size() > MaxDataFileBytes {
		return fail("file changed during open")
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxDataFileBytes+1))
	if err != nil {
		return fail("file read failed")
	}
	after, err := f.Stat()
	if err != nil || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return fail("file changed during read; publish atomically")
	}
	result.value, err = decodeExternalJSON(raw)
	if err != nil {
		return fail("invalid or oversized JSON")
	}
	result.stamp, err = externalStamp(result.value, after.ModTime(), now)
	if err != nil {
		return fail(err.Error())
	}
	return result
}

func prepareExternalData(s *State, now time.Time) *State {
	if len(s.Layout.DataSources) == 0 {
		return s
	}
	clone := *s
	clone.dataResults = map[string]dataSnapshot{}
	files := map[string]dataSnapshot{}
	_ = walkValueBindings(s.Layout.Root, func(b *ValueBinding) error {
		if b == nil {
			return nil
		}
		if _, loaded := clone.dataResults[b.Source]; loaded {
			return nil
		}
		config := s.Layout.DataSources[b.Source]
		result, loaded := files[config.Path]
		if !loaded {
			result = readExternalData(config, now)
			files[config.Path] = result
		}
		result.config = config
		clone.dataResults[b.Source] = result
		return nil
	})
	return &clone
}

func externalPointer(value any, path string) (any, bool) {
	parts, err := pointerParts(path)
	if err != nil {
		return nil, false
	}
	for _, part := range parts {
		switch current := value.(type) {
		case map[string]any:
			var ok bool
			value, ok = current[part]
			if !ok {
				return nil, false
			}
		case []any:
			if part == "" || len(part) > 1 && part[0] == '0' {
				return nil, false
			}
			i, err := strconv.Atoi(part)
			if err != nil || i < 0 || i >= len(current) || strconv.Itoa(i) != part {
				return nil, false
			}
			value = current[i]
		default:
			return nil, false
		}
	}
	return value, value != nil
}

func formatExternalValue(value any, format string, now time.Time) (string, error) {
	if format == "" || format == "text" {
		switch v := value.(type) {
		case string:
			return Clean(v, 400), nil
		case json.Number:
			return Clean(v.String(), 400), nil
		case bool:
			return strconv.FormatBool(v), nil
		default:
			return "", errors.New("expected a scalar")
		}
	}
	var n float64
	var original string
	if format == "deadline" {
		raw, ok := value.(string)
		deadline, err := time.Parse(time.RFC3339Nano, raw)
		if !ok || err != nil {
			return "", errors.New("expected an RFC 3339 deadline")
		}
		n = max(0, deadline.Sub(now).Seconds())
	} else {
		v, ok := value.(json.Number)
		if !ok {
			return "", errors.New("expected a number")
		}
		var err error
		n, err = v.Float64()
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
			return "", errors.New("expected a finite number")
		}
		original = v.String()
	}
	switch format {
	case "number":
		return original, nil
	case "mm:ss", "deadline":
		if n < 0 || n > 315360000 {
			return "", errors.New("duration outside 0..315360000 seconds")
		}
		seconds := int64(math.Ceil(n))
		return fmt.Sprintf("%02d:%02d", seconds/60, seconds%60), nil
	case "percent", "bar":
		if n < 0 || n > 100 {
			return "", errors.New("percentage outside 0..100")
		}
		text := original + "%"
		if format == "bar" {
			filled := int(n / 10)
			text = "[" + strings.Repeat("#", filled) + strings.Repeat("-", 10-filled) + "] " + text
		}
		return text, nil
	}
	return "", errors.New("unknown value format")
}

// Resolution is pure and shared by text, titles, and static/dynamic DAG labels.
// Failed/stale inputs never fall back to a previous apparently-live value.
func renderExternalValue(s *State, b *ValueBinding, now time.Time) string {
	unknown := func(reason string) string { return b.Prefix + "unknown (" + reason + ")" + b.Suffix }
	if s.Layout == nil {
		return unknown("data source not loaded")
	}
	config, configured := s.Layout.DataSources[b.Source]
	snapshot, loaded := s.dataResults[b.Source]
	if !configured || !loaded || snapshot.config != config {
		return unknown("data source not loaded")
	}
	if snapshot.problem != "" {
		return unknown(snapshot.problem)
	}
	if now.IsZero() {
		now = time.Now()
	}
	if snapshot.stamp.After(now.Add(2 * time.Second)) {
		return unknown("clock skew")
	}
	if now.Sub(snapshot.stamp) > config.staleAfter() {
		return b.Prefix + "stale (external data expired)" + b.Suffix
	}
	value, ok := externalPointer(snapshot.value, b.Path)
	if !ok {
		return unknown("missing or null value")
	}
	text, err := formatExternalValue(value, b.Format, now)
	if err != nil {
		return unknown(err.Error())
	}
	return Clean(b.Prefix+text+b.Suffix, 400)
}

// Projection changes caller-authored display strings only, never identity/state.
func projectExternalLabels(p *Panel, s *State, now time.Time) {
	if p.TitleValue != nil {
		p.Title = boundedLabel("[data] " + renderExternalValue(s, p.TitleValue, now))
	}
	if p.DAG != nil {
		for i := range p.DAG.Nodes {
			n := &p.DAG.Nodes[i]
			if n.LabelValue != nil {
				n.Label = boundedLabel("[data] " + renderExternalValue(s, n.LabelValue, now))
			}
		}
		for i := range p.DAG.Edges {
			e := &p.DAG.Edges[i]
			if e.LabelValue != nil {
				e.Label = boundedLabel("[data] " + renderExternalValue(s, e.LabelValue, now))
			}
		}
	}
	for i := range p.Children {
		projectExternalLabels(&p.Children[i], s, now)
	}
}

// PublishDataFile is an optional agent/script-invoked atomic producer. It writes
// one complete value, never a layout, telemetry event, process, or model request.
func PublishDataFile(path string, raw []byte) error {
	if !filepath.IsAbs(path) || len(path) > 4096 || Clean(path, 4096) != path {
		return errors.New("data file requires an absolute printable path")
	}
	value, err := decodeExternalJSON(raw)
	if err != nil {
		return err
	}
	now := time.Now()
	if object, ok := value.(map[string]any); ok {
		if _, exists := object["updated_at"]; !exists {
			object["updated_at"] = now.UTC().Format(time.RFC3339Nano)
		}
	}
	if _, err := externalStamp(value, now, now); err != nil {
		return err
	}
	raw, err = json.Marshal(value)
	if err != nil || len(raw) > MaxDataFileBytes {
		return errors.New("published external JSON exceeds 1 MiB")
	}
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return errors.New("data destination must be a regular file, not a link")
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return atomicWrite(path, raw, 0600)
}
