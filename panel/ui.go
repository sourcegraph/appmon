package panel

import (
	"database/sql"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/sourcegraph/appmon"
	"github.com/sqs/mux"
)

const (
	appmonUIRoutes = "appmon:ui:routes"
	appmonUICall   = "appmon:ui:call"
	appmonUICalls  = "appmon:ui:calls"
	appmonUIMain   = "appmon:ui:main"
)

var baseHref string

// Router adds panel routes to an existing mux.Router.
func UIRouter(theBaseHref string, rt *mux.Router) *mux.Router {
	baseHref = theBaseHref
	rt.Path(`/calls/{CallID:\d+}`).Methods("GET").HandlerFunc(uiCall).Name(appmonUICall)
	rt.Path("/calls").Methods("GET").HandlerFunc(uiCalls).Name(appmonUICalls)
	rt.Path("/").Methods("GET").HandlerFunc(uiMain).Name(appmonUIMain)

	return rt
}

func uiCall(w http.ResponseWriter, r *http.Request) {
	v := mux.Vars(r)
	callID, _ := strconv.ParseInt(v["CallID"], 10, 64)

	calls, err := appmon.QueryCalls(`WHERE id=$1 OR parent_call_id=$1 ORDER BY start ASC`, callID)
	if err != nil {
		http.Error(w, "QueryCalls failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl(appmonUICall, uiCallHTML)(w, struct {
		common
		CallID int64
		Calls  []*appmon.Call
	}{
		common: newCommon("Call"),
		CallID: callID,
		Calls:  calls,
	})
}

var uiCallHTML = `
<h1>Parent call {{.CallID}}</h1>
<style>
.parent-call { border-bottom: solid 5px #999; }
</style>
<div class="row-fluid">
  <div class="col-md-12">
    <table class="table">
      <thead><tr><th>ID</th><th>Route</th><th>Duration</th><th>URL</th><th>Bytes</th><th>Status</th></thead>
      <tbody>
        {{$CallID:=.CallID}}
        {{range .Calls}}
          {{$isParent:=(eq .ID $CallID)}}
          <tr class="{{if isHTTPError .HTTPStatusCode}}danger{{end}} {{if $isParent}}parent-call{{end}}">
            <td>{{.ID}} {{if $isParent}}<br><strong class="text-muted">Parent</strong>{{end}}</td>
            <td style="max-width:150px"><strong>{{.Route}}</strong></td>
            <td>{{.Duration}}</td>
            <td style="word-wrap:break-word;max-width:200px;"><tt style="font-size:0.85em"><a href="{{.URL}}" target="_blank">{{.URL}}</a></tt></td>
            <td>{{bytes .BodyLength}}</td>
            <td title="{{.Err}}">{{.HTTPStatusCode}}</td>
          </tr>
        {{else}}
          <tr><td colspan="5" class="alert alert-warning">No calls found for call ID {{.CallID}}.</td></tr>
        {{end}}
      </tbody>
    </table>
  </div>
</div>
`

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
	selectedApp := q.Get("app")
	if selectedRoute != "" && selectedApp != "" {
		calls, err = appmon.QueryCalls(`WHERE app = $1 AND route = $2 AND (current_timestamp - "start" < ($3::int * interval '1 hour')) AND ((NOT $4) OR (http_status_code < 200 OR http_status_code >= 400)) ORDER BY `+sorts[sort]+` DESC LIMIT 100`, selectedApp, selectedRoute, lastNHours, failedOnly)
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
		SelectedApp   string
		SelectedRoute string
		Calls         []*appmon.Call
	}{
		common:        newCommon("Calls"),
		LastNHours:    lastNHours,
		FailedOnly:    failedOnly,
		Sort:          sort,
		CallRoutes:    callRoutes,
		SelectedApp:   selectedApp,
		SelectedRoute: selectedRoute,
		Calls:         calls,
	})
}

type callRoute struct {
	App         string
	Route       string
	Count       int
	AvgDuration int64
}

func getCallRoutes(lastNHours int, failedOnly bool) (callRoutes []*callRoute, err error) {
	var rows *sql.Rows
	callRouteSQL := `
      SELECT * FROM (
        SELECT c.app, c.route, COUNT(c.*) AS count, ROUND(AVG(extract(epoch from (c."end" - c.start))*1000000))::bigint AS avg_duration
        FROM "` + appmon.DBSchema + `".call c
        WHERE current_timestamp - c.start < ($1::int * interval '1 hour')
          AND c."end" IS NOT NULL
          AND ((NOT $2) OR (http_status_code < 200 OR http_status_code >= 400))
        GROUP BY app, route
      ) q ORDER BY count DESC
`
	rows, err = appmon.DB.Query(callRouteSQL, lastNHours, failedOnly)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		cr := new(callRoute)
		err = rows.Scan(&cr.App, &cr.Route, &cr.Count, &cr.AvgDuration)
		if err != nil {
			return
		}
		callRoutes = append(callRoutes, cr)
	}
	return
}

var uiCallsHTML = `
<h1>Calls</h1>
<div class="row-fluid">
  <div class="col-md-2">
    <form action="calls" method="get" class="form">
      {{if .SelectedRoute}}<input type="hidden" name="route" value="{{.SelectedRoute}}">{{end}}
      {{if .SelectedApp}}<input type="hidden" name="app" value="{{.SelectedApp}}">{{end}}
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
    {{$SelectedApp := .SelectedApp}}
    {{range .CallRoutes}}
      <a href="calls?sort={{$Sort}}&failedOnly={{$FailedOnly}}&lastNHours={{$LastNHours}}&route={{.Route}}&app={{.App}}" class="list-group-item {{if and (eq $SelectedRoute .Route) (eq $SelectedApp .App)}}active{{end}}">
        <strong>{{if .App}}{{.App}}{{else}}(no app){{end}}</strong>
        {{if .Route}}{{.Route}}{{else}}(unnamed){{end}}
        <span class="badge">{{.Count|num}}</span>
        <span class="badge {{durationBadgeClass .AvgDuration}}"><span class="glyphicon glyphicon-time" style="font-size:0.85em"></span> {{duration .AvgDuration}}</span>
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
    <style>
    .d0 { background-color: #bbb; }
    .d1 { background-color: #ccb7b7; }
    .d2 { background-color: #cca7a7; }
    .d3 { background-color: #cc9999; }
    .d4 { background-color: #cc8282; }
    .d5 { background-color: #cc7676; }
    .d6 { background-color: #cc4f4f; }
    .d7 { background-color: #cc3939; }
    .d8 { background-color: #cc2a2a; }
    .d9 { background-color: #cc1c1c; }
    .d10 { background-color: #cc0e0e; }
    </style>
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
			"eq":                 reflect.DeepEqual,
			"isHTTPError":        func(code int) bool { return code < 200 || code >= 400 },
			"duration":           func(usec int64) time.Duration { return time.Duration(time.Microsecond * time.Duration(usec)) },
			"timeAgo":            func(t time.Time) string { return time.Duration(roundPow(int64(time.Since(t)), 9)).String() },
			"roundMillion":       func(n int64) int64 { return roundPow(n, 6) },
			"bytes":              func(bytes int) string { return fmt.Sprintf("%.1f kb", float64(bytes)/1000.0) },
			"num":                num,
			"durationBadgeClass": durationBadgeClass,
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

// num abbreviates and rounds n. Examples: 150, 13.2K, 1.5K.
func num(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	} else if n < 30000 {
		s := fmt.Sprintf("%.1fk", float64(n)/1000)
		return strings.Replace(s, ".0k", "k", 1)
	} else if n < 500000 {
		return strconv.Itoa(n/1000) + "k"
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000.0)
}

func durationBadgeClass(usec int64) string {
	msec := usec / 1000
	if msec < 30 {
		return "d0"
	} else if msec < 60 {
		return "d1"
	} else if msec < 90 {
		return "d2"
	} else if msec < 150 {
		return "d3"
	} else if msec < 250 {
		return "d4"
	} else if msec < 400 {
		return "d5"
	} else if msec < 600 {
		return "d6"
	} else if msec < 900 {
		return "d7"
	} else if msec < 1300 {
		return "d8"
	} else if msec < 1900 {
		return "d9"
	}
	return "d10"
}
