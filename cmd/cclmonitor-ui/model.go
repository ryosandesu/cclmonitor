package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/ryosandesu/cclmonitor/internal/eventlog"
	"github.com/ryosandesu/cclmonitor/internal/metrics"
)

type period int

const (
	periodToday period = iota
	period7d
	period30d
	periodAll
)

type tab int

const (
	tabOverview tab = iota
	tabTools
	tabTimeline
	tabEvents
)

// model は TUI の全状態を保持する（Bubble Tea の tea.Model インタフェースを実装）。
type model struct {
	logDir    string
	grace     time.Duration
	snapshot  bool
	paused    bool
	activeTab tab
	period    period
	width     int
	height    int

	// 集計済みデータ
	invocations []metrics.Invocation
	stats       metrics.Stats
	perTool     map[string]metrics.Stats
	daily       []metrics.DailyBucket
	offenders   []metrics.ValueCount
	recentEvts  []eventlog.Event // 生イベント（Events タブ用）

	// ライブ更新用
	reader    *eventlog.Reader
	readerDay string // "2006-01-02"

	// Events タブ用スクロール位置
	eventsOffset int
}

func newModel(logDir string, grace time.Duration, snapshot bool) model {
	m := model{
		logDir:   logDir,
		grace:    grace,
		snapshot: snapshot,
		period:   periodToday,
	}
	m.reload()
	return m
}

func (m model) Init() tea.Cmd {
	if m.snapshot {
		return nil
	}
	return tickCmd()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Tab1):
			m.activeTab = tabOverview
		case key.Matches(msg, keys.Tab2):
			m.activeTab = tabTools
		case key.Matches(msg, keys.Tab3):
			m.activeTab = tabTimeline
		case key.Matches(msg, keys.Tab4):
			m.activeTab = tabEvents
		case key.Matches(msg, keys.PeriodT):
			m.period = periodToday
			m.reload()
		case key.Matches(msg, keys.Period7):
			m.period = period7d
			m.reload()
		case key.Matches(msg, keys.PeriodM):
			m.period = period30d
			m.reload()
		case key.Matches(msg, keys.PeriodA):
			m.period = periodAll
			m.reload()
		case key.Matches(msg, keys.Refresh):
			m.reload()
		case key.Matches(msg, keys.Pause):
			m.paused = !m.paused
		case key.Matches(msg, keys.Down):
			if m.activeTab == tabEvents {
				m.eventsOffset++
			}
		case key.Matches(msg, keys.Up):
			if m.activeTab == tabEvents && m.eventsOffset > 0 {
				m.eventsOffset--
			}
		}

	case tickMsg:
		if !m.paused {
			m.poll()
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return ""
	}
	header := m.renderHeader()
	var body string
	switch m.activeTab {
	case tabOverview:
		body = renderOverview(m)
	case tabTools:
		body = renderTools(m)
	case tabTimeline:
		body = renderTimeline(m)
	case tabEvents:
		body = renderEvents(m)
	}
	return header + "\n" + body
}

// reload は現在の period に合わせてログを全件読み直す。
func (m *model) reload() {
	now := time.Now()
	from, to := m.periodRange(now)

	// 期間フィルタ分（カード・ツール集計用）
	evts, err := eventlog.ReadRange(m.logDir, from, to)
	if err != nil {
		return
	}
	m.recentEvts = evts

	// Timeline は常に 30 日分（期間フィルタに依存しない）
	y, mo, d := now.Date()
	todayStart := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	thirtyFrom := todayStart.AddDate(0, 0, -29)
	allEvts, _ := eventlog.ReadRange(m.logDir, thirtyFrom, to)
	m.recalc(now, allEvts)

	// ライブ更新用 Reader を今日のファイルに向ける
	if !m.snapshot {
		m.initReader(now)
	}
}

// poll は今日の増分イベントを Reader から取得して集計に加算する。
func (m *model) poll() {
	now := time.Now()
	today := now.Format("2006-01-02")

	// 日付ロールオーバー検出
	if today != m.readerDay {
		m.reload()
		return
	}

	if m.reader == nil {
		return
	}

	newEvts, err := m.reader.Poll()
	if err != nil || len(newEvts) == 0 {
		return
	}
	m.recentEvts = append(m.recentEvts, newEvts...)

	// Timeline 用の 30 日分にも今日の増分を反映
	y, mo, d := now.Date()
	todayStart := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	thirtyFrom := todayStart.AddDate(0, 0, -29)
	allEvts, _ := eventlog.ReadRange(m.logDir, thirtyFrom, now.Add(time.Second))
	m.recalc(now, allEvts)
}

// recalc は recentEvts から期間フィルタ分を、allEvts から Timeline 分を計算する。
func (m *model) recalc(now time.Time, allEvts []eventlog.Event) {
	invs := metrics.PairInvocations(m.recentEvts, now, m.grace)
	m.invocations = invs
	m.stats = metrics.Summarize(invs)
	m.perTool = metrics.PerTool(invs)
	m.offenders = metrics.TopOffenders(invs, []string{"denied", "unknown"}, 10)

	// Timeline は常に 30 日分の invocations から算出
	allInvs := metrics.PairInvocations(allEvts, now, m.grace)
	m.daily = metrics.Daily(allInvs, 30, now)
}

// periodRange は現在の period 設定から from/to を返す。
func (m *model) periodRange(now time.Time) (from, to time.Time) {
	y, mo, d := now.Date()
	todayStart := time.Date(y, mo, d, 0, 0, 0, 0, now.Location())
	to = now.Add(time.Second) // to は排他的なので少し先

	switch m.period {
	case periodToday:
		from = todayStart
	case period7d:
		from = todayStart.AddDate(0, 0, -6)
	case period30d:
		from = todayStart.AddDate(0, 0, -29)
	case periodAll:
		from = time.Time{} // ゼロ値 = 制限なし
	}
	return from, to
}

func (m *model) initReader(now time.Time) {
	if m.reader != nil {
		m.reader.Close()
		m.reader = nil
	}
	day := now.Format("2006-01-02")
	path := filepath.Join(m.logDir, "cclmonitor."+day+".log")
	if _, err := os.Stat(path); err != nil {
		m.readerDay = day
		return
	}
	r, err := eventlog.NewReader(path)
	if err != nil {
		return
	}
	// 既に全件を ReadRange で読んでいるので、Poll で重複しないよう末尾まで進める
	r.Poll() //nolint
	m.reader = r
	m.readerDay = day
}

func (m model) renderHeader() string {
	periods := []struct {
		key   string
		label string
		p     period
	}{
		{"t", "Today", periodToday},
		{"7", "7d", period7d},
		{"m", "30d", period30d},
		{"a", "All", periodAll},
	}
	periodStr := ""
	for _, p := range periods {
		label := fmt.Sprintf("[%s] %s", p.key, p.label)
		if p.p == m.period {
			periodStr += styleTabActive.Render(label) + "  "
		} else {
			periodStr += styleTabInactive.Render(label) + "  "
		}
	}

	pauseHint := ""
	if m.paused {
		pauseHint = styleDenied.Render(" [paused]")
	}

	return styleHeader.Render("cclmonitor") + "  " + periodStr +
		styleHeader.Render(" r refresh  s pause  q quit") + pauseHint
}
