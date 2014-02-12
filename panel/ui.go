package panel

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/appmon"
	"html/template"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

const (
	appmonUIRoutes = "appmon:ui:routes"
	appmonUICalls  = "appmon:ui:calls"
	appmonUIMain   = "appmon:ui:main"
)

var baseHref string

// Router adds panel routes to an existing mux.Router.
func UIRouter(theBaseHref string, rt *mux.Router) *mux.Router {
	baseHref = theBaseHref
	rt.Path("/calls").Methods("GET").HandlerFunc(uiCalls).Name(appmonUICalls)
	rt.Path("/").Methods("GET").HandlerFunc(uiMain).Name(appmonUIMain)

	return rt
}

func uiCalls(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	lastNHoursStr := q.Get("lastNHours")
	var lastNHours int
	if lastNHoursStr == "" {
		lastNHours = 7
	} else {
		var err error
		lastNHours, err = strconv.Atoi(lastNHoursStr)
		if err != nil {
			http.Error(w, "bad 'lastNHours' parameter: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	failedOnlyStr := q.Get("failedOnly")
	var failedOnly bool
	if failedOnlyStr != "" {
		var err error
		failedOnly, err = strconv.ParseBool(failedOnlyStr)
		if err != nil {
			http.Error(w, "bad 'failedOnly' parameter: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	sort := q.Get("sort")
	if sort == "" {
		sort = "date"
	}
	sorts := map[string]string{"date": "start", "duration": `"end" - "start"`}
	if _, ok := sorts[sort]; !ok {
		http.Error(w, "bad 'sort' parameter", http.StatusBadRequest)
		return
	}

	callRoutes, err := getCallRoutes(lastNHours, failedOnly)
	if err != nil {
		http.Error(w, "getCallRoutes failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var calls []*appmon.Call
	selectedRoute := q.Get("route")
	if selectedRoute != "" {
		calls, err = appmon.QueryCalls(`WHERE route = $1 AND (current_timestamp - "start" < ($2::int * interval '1 hour')) AND ((NOT $3) OR (http_status_code < 200 OR http_status_code >= 400)) ORDER BY `+sorts[sort]+` DESC LIMIT 100`, selectedRoute, lastNHours, failedOnly)
		if err != nil {
			http.Error(w, "QueryCalls failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tmpl(appmonUICalls, uiCallsHTML)(w, struct {
		common
		LastNHours    int
		FailedOnly    bool
		Sort          string
		CallRoutes    []*callRoute
		SelectedRoute string
		Calls         []*appmon.Call
	}{
		common:        newCommon("Calls"),
		LastNHours:    lastNHours,
		FailedOnly:    failedOnly,
		Sort:          sort,
		CallRoutes:    callRoutes,
		SelectedRoute: selectedRoute,
		Calls:         calls,
	})
}

type callRoute struct {
	Route       string
	Count       int
	AvgDuration int64
}

func getCallRoutes(lastNHours int, failedOnly bool) (callRoutes []*callRoute, err error) {
	var rows *sql.Rows
	callRouteSQL := `
      SELECT * FROM (
        SELECT c.route, COUNT(c.*) AS count, ROUND(AVG(extract(epoch from (c."end" - c.start))*1000000))::bigint AS avg_duration
        FROM "` + appmon.DBSchema + `".call c
        WHERE current_timestamp - c.start < ($1::int * interval '1 hour')
          AND c."end" IS NOT NULL
          AND ((NOT $2) OR (http_status_code < 200 OR http_status_code >= 400))
        GROUP BY route
      ) q ORDER BY count DESC
`
	rows, err = appmon.DB.Query(callRouteSQL, lastNHours, failedOnly)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		cr := new(callRoute)
		err = rows.Scan(&cr.Route, &cr.Count, &cr.AvgDuration)
		if err != nil {
			return
		}
		callRoutes = append(callRoutes, cr)
	}
	return
}

var uiCallsHTML = `
<h1>Calls</h1>
<div class="row">
  <div class="col-md-2">
    <form action="calls" method="get" class="form">
      {{if .SelectedRoute}}<input type="hidden" name="route" value="{{.SelectedRoute}}">{{end}}
      <div class="form-group">
        <label for="lastNHours">Last # hours</label>
        <input type="number" class="form-control" id="lastNHours" name="lastNHours" placeholder="#" value="{{.LastNHours}}">
      </div>
      <div class="form-group">
        <label>Only show:</label>
        <div class="checkbox">
          <label><input type="checkbox" name="failedOnly" value="t" {{if .FailedOnly }}checked{{end}}> Failures</label>
        </div>
      </div>
      <div class="form-group">
        <label>Sort order:</label>
        <div class="radio">
          <label><input type="radio" name="sort" value="date" {{if eq .Sort "date"}}checked{{end}}> Most recent</label>
        </div>
        <div class="radio">
          <label><input type="radio" name="sort" value="duration" {{if eq .Sort "duration"}}checked{{end}}> Longest duration</label>
        </div>
      </div>
      <button type="submit" class="btn btn-primary">Update list</button>
    </form>
  </div>
  <div class="col-md-3">
    <div class="list-group">
    {{$LastNHours := .LastNHours}}
    {{$FailedOnly := .FailedOnly}}
    {{$Sort := .Sort}}
    {{$SelectedRoute := .SelectedRoute}}
    {{range .CallRoutes}}
      <a href="calls?sort={{$Sort}}&failedOnly={{$FailedOnly}}&lastNHours={{$LastNHours}}&route={{.Route}}" class="list-group-item {{if eq $SelectedRoute .Route}}active{{end}}">
        {{if .Route}}{{.Route}}{{else}}(unnamed){{end}}
        <span class="badge">{{.Count}}</span>
        <span class="badge"><span class="glyphicon glyphicon-time" style="font-size:0.85em"></span> {{duration .AvgDuration}}</span>
      </a>
    {{else}}
      <li><div class="alert alert-error">No routes to show.</div></li>
    {{end}}
    </div>
  </div>
  <div class="col-md-7">
     {{if eq .SelectedRoute ""}}
       <div class="alert alert-warning">Select a route.</div>
     {{else}}
       <table class="table">
         <thead><tr><th>ID</th><th>Start</th><th>URL</th><th>User</th><th>Duration</th><th>Bytes</th><th>Status</th></thead>
         <tbody>
           {{range .Calls}}
             <tr class="{{if isHTTPError .HTTPStatusCode}}danger{{end}}">
               <td>{{.ID}} {{if .ParentCallID}}<br><span class="text-muted" title="ParentCallID">{{.ParentCallID}}</span>{{end}}</td>
               <td>
                 {{.Start.Format "2006-01-02 15:04:05"}}<br>
                 <span class="text-muted">{{timeAgo .Start}}</span>
               </td>
               <td style="word-wrap:break-word;max-width:200px;"><a href="{{.URL}}" target="_blank">{{.URL}}</a></td>
               <td title="{{.RemoteAddr}} -- {{.UserAgent}}">{{if .UID}}{{.UID}}{{else}}Anon{{end}}</td>
               <td>{{.Duration}}</td>
               <td>{{bytes .BodyLength}}</td>
               <td title="{{.Err}}">{{.HTTPStatusCode}}</td>
             </tr>
           {{else}}
             <tr><td colspan="5" class="alert alert-warning">No calls found for route {{$SelectedRoute}}.</td></tr>
           {{end}}
         </tbody>
       </table>
     {{end}}
  </div>
</div>
`

func uiMain(w http.ResponseWriter, r *http.Request) {
	tmpl(appmonUIMain, uiMainHTML)(w, newCommon("Main"))
}

var uiMainHTML = `

`

type common struct {
	Title    string
	BaseHref string
}

func newCommon(title string) common {
	return common{title, baseHref}
}

func tmpl(name, bodySource string) func(http.ResponseWriter, interface{}) {
	src := `
<!DOCTYPE html>
<html>
  <head>
    <base href="{{.BaseHref}}">
    <title>{{.Title}} - Appmon</title>
    <meta charset="utf-8">
    <link rel="shortcut icon" href="data:image/x-icon;," type="image/x-icon"> 
    <link href="//netdna.bootstrapcdn.com/bootstrap/3.0.0/css/bootstrap.min.css" rel="stylesheet">
  </head>
  <body>
    <div class="navbar navbar-default">
      <div class="navbar-header">
        <button type="button" class="navbar-toggle" data-toggle="collapse" data-target=".navbar-collapse">
          <span class="icon-bar"></span>
          <span class="icon-bar"></span>
          <span class="icon-bar"></span>
        </button>
        <a class="navbar-brand" href="#">Appmon</a>
      </div>
      <div class="navbar-collapse collapse">
        <ul class="nav navbar-nav">
          <li><a href="calls">Calls</a></li>
        </ul>
      </div><!--/.nav-collapse -->
    </div>
    <div class="container-fluid">
` + bodySource + `
    </div>
  </body>
</html>
`
	return func(w http.ResponseWriter, data interface{}) {
		roundPow := func(n int64, pow int64) int64 {
			return int64(math.Pow(10, float64(pow)) * float64(int64((float64(n) / math.Pow(10, float64(pow))))))
		}
		t := template.New(name).Funcs(template.FuncMap{
			"eq":           reflect.DeepEqual,
			"isHTTPError":  func(code int) bool { return code < 200 || code >= 400 },
			"duration":     func(usec int64) time.Duration { return time.Duration(time.Microsecond * time.Duration(usec)) },
			"timeAgo":      func(t time.Time) string { return time.Duration(roundPow(int64(time.Since(t)), 9)).String() },
			"roundMillion": func(n int64) int64 { return roundPow(n, 6) },
			"bytes":        func(bytes int) string { return fmt.Sprintf("%.1f kb", float64(bytes)/1000.0) },
		})

		t, err := t.Parse(src)
		if err != nil {
			http.Error(w, "template parse error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		err = t.Execute(w, data)
		if err != nil {
			http.Error(w, "template execution error: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
