package panel

import (
	"net/http"
	"text/template"

	"github.com/sourcegraph/appmon"
)

var uiStatsTmpl = template.Must(template.New("").Parse(`
<p>{{.Foo}}</p>
`))

func init() {
	appmon.InitSQL = append(appmon.InitSQL, `
CREATE OR REPLACE FUNCTION appmon.sessions_in_range(range_start timestamp without time zone, range_end timestamp without time zone)
RETURNS TABLE(uid integer, session_start timestamp without time zone, session_end timestamp without time zone, count bigint) AS $$
  SELECT c.uid, min(c.start) AS session_start, max(c.end) AS session_end, count(*)
  FROM appmon.call c
  WHERE c.end <= range_end AND c.start > range_start
  GROUP BY uid, date_trunc('hour', c.start)
$$ LANGUAGE SQL STABLE`)
	appmon.InitSQL = append(appmon.InitSQL, `
CREATE OR REPLACE FUNCTION appmon.url_matches(url text, pattern text) RETURNS boolean AS $$
  SELECT true -- TODO
$$ LANGUAGE SQL IMMUTABLE`)
}

func uiStats(w http.ResponseWriter, r *http.Request) {
	dAct, err := getDailyActivity()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	uiStatsTmpl.Execute(w, map[string]interface{}{
		"DailyActivity": dAct,
	})
}

type dailyActivity struct {
	DaysAgo   string
	UserCount int
}

// This takes 5.8 seconds for one day (2196316 page views)
func getDailyActivity() (*dailyActivity, error) {
	var dAct dailyActivity

	rows, err := appmon.DB.Query(`
WITH sessions AS (
  SELECT * FROM appmon.sessions_in_range(localtimestamp - interval '2 weeks', localtimestamp)
), user_days AS (
  SELECT date_trunc('day', age(sessions.session_start, localtimestamp)) AS daysAgo, uid, count(*) as sessions_count FROM sessions GROUP BY date_trunc('day', age(sessions.session_start, localtimestamp)), uid
), days AS (
  SELECT daysAgo, count(*) as userCount FROM user_days GROUP BY daysAgo
)
SELECT * FROM days;
`)

	// TODO: make this use Query...
	err := dbh.Select(&dAct, `
WITH sessions AS (
  SELECT * FROM appmon.sessions_in_range(localtimestamp - interval '2 weeks', localtimestamp)
), user_days AS (
  SELECT date_trunc('day', age(sessions.session_start, localtimestamp)) AS daysAgo, uid, count(*) as sessions_count FROM sessions GROUP BY date_trunc('day', age(sessions.session_start, localtimestamp)), uid
), days AS (
  SELECT daysAgo, count(*) as userCount FROM user_days GROUP BY daysAgo
)
SELECT * FROM days;
`)
	if err != nil {
		return nil, err
	}

	return &dAct, nil
}
