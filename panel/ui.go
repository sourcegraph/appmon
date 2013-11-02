package panel

import (
	"database/sql"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sourcegraph/go-nnz/nnz"
	"github.com/sourcegraph/track"
	"html/template"
	"math"
	"net/http"
	"reflect"
	"strconv"
	"time"
)

const (
	trackUICalls = "track:ui:calls"
	trackUIMain  = "track:ui:main"
	trackUIUsers = "track:ui:users"
	trackUIViews = "track:ui:views"
)

var baseHref string

// Router adds panel routes to an existing mux.Router.
func UIRouter(theBaseHref string, rt *mux.Router) *mux.Router {
	baseHref = theBaseHref
	rt.Path("/calls").Methods("GET").HandlerFunc(uiCalls).Name(trackUICalls)
	rt.Path("/views").Methods("GET").HandlerFunc(uiViews).Name(trackUIViews)
	rt.Path("/users").Methods("GET").HandlerFunc(uiUsers).Name(trackUIUsers)
	rt.Path("/").Methods("GET").HandlerFunc(uiMain).Name(trackUIMain)

	return rt
}

func uiUsers(w http.ResponseWriter, r *http.Request) {
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

	users, err := getUsers(lastNHours)
	if err != nil {
		http.Error(w, "getUsers failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var userViews []*track.View
	selectedUser := q.Get("user")
	if selectedUser != "" {
		userViews, err = track.QueryViews(`INNER JOIN "`+track.DBSchema+`".instance i ON i.id = view.instance WHERE i."user" = $1 AND (current_timestamp - view.date < ($2::int * interval '1 hour')) ORDER BY view.date DESC LIMIT 100`, selectedUser, lastNHours)
		if err != nil {
			http.Error(w, "QueryViews failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tmpl(trackUIUsers, uiUsersHTML)(w, struct {
		common
		LastNHours   int
		Sort         string
		Users        []*user
		SelectedUser string
		Views        []*track.View
	}{
		common:       newCommon("Users"),
		LastNHours:   lastNHours,
		Users:        users,
		SelectedUser: selectedUser,
		Views:        userViews,
	})
}

type user struct {
	User  string
	Views nnz.Int
}

func getUsers(lastNHours int) (users []*user, err error) {
	var rows *sql.Rows
	userSQL := `
      SELECT * FROM (
        SELECT i."user", COUNT(v.*) AS count
        FROM "` + track.DBSchema + `".view v
        INNER JOIN "` + track.DBSchema + `".instance i ON i.id = v.instance
        WHERE i."user" IS NOT NULL AND current_timestamp - date < ($1::int * interval '1 hour')
        GROUP BY i."user"
      ) q ORDER BY count DESC
`
	rows, err = track.DB.Query(userSQL, lastNHours)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		u := new(user)
		err = rows.Scan(&u.User, &u.Views)
		if err != nil {
			return
		}
		users = append(users, u)
	}
	return
}

var uiUsersHTML = `
<h1>Users</h1>
<div class="row">
  <div class="col-md-2">
    <form action="users" method="get" class="form">
      {{if .SelectedUser}}<input type="hidden" name="state" value="{{.SelectedUser}}">{{end}}
      <div class="form-group">
        <label for="lastNHours">Last # hours</label>
        <input type="number" class="form-control" id="lastNHours" name="lastNHours" placeholder="#" value="{{.LastNHours}}">
      </div>
      <button type="submit" class="btn btn-primary">Update list</button>
    </form>
  </div>
  <div class="col-md-3">
    <div class="list-group">
    {{$LastNHours := .LastNHours}}
    {{$SelectedUser := .SelectedUser}}
    {{range .Users}}
      <a href="users?lastNHours={{$LastNHours}}&user={{.User}}" class="list-group-item {{if eq $SelectedUser .User}}active{{end}}">
        {{.User}}
        <span class="badge">{{.Views}}</span>
      </a>
    {{else}}
      <li><div class="alert alert-error">No users to show.</div></li>
    {{end}}
    </div>
  </div>
  <div class="col-md-7">
     {{if eq .SelectedUser ""}}
       <div class="alert alert-warning">Select a user.</div>
     {{else}}
       <table class="table">
         <thead><tr><th>Date</th><th>URL</th>
         <tbody>
           {{range .Views}}
             <tr>
               <td>
                 {{.Date.Format "2006-01-02 15:04:05"}}<br>
                 <span class="text-muted">{{timeAgo .Date}}</span>
               </td>
               <td style="word-wrap:break-word;max-width:200px;"><a href="{{.RequestURI}}" target="_blank">{{.RequestURI}}</a></td>
             </tr>
           {{else}}
             <tr><td colspan="5" class="alert alert-warning">No views found for user {{$SelectedUser}}.</td></tr>
           {{end}}
         </tbody>
       </table>
     {{end}}
  </div>
</div>
`

func uiViews(w http.ResponseWriter, r *http.Request) {
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

	sort := q.Get("sort")
	if sort == "" {
		sort = "date"
	}
	okSorts := map[string]struct{}{"date": struct{}{}}
	if _, ok := okSorts[sort]; !ok {
		http.Error(w, "bad 'sort' parameter", http.StatusBadRequest)
		return
	}

	viewStates, err := getViewStates(lastNHours)
	if err != nil {
		http.Error(w, "getViewStates failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var views []*track.View
	selectedState := q.Get("state")
	if selectedState != "" {
		views, err = track.QueryViews(`WHERE state = $1 AND (current_timestamp - date < ($2::int * interval '1 hour')) ORDER BY `+sort+` DESC LIMIT 100`, selectedState, lastNHours)
		if err != nil {
			http.Error(w, "QueryViews failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tmpl(trackUIViews, uiViewsHTML)(w, struct {
		common
		LastNHours    int
		Sort          string
		ViewStates    []*viewState
		SelectedState string
		Views         []*track.View
	}{
		common:        newCommon("Views"),
		LastNHours:    lastNHours,
		Sort:          sort,
		ViewStates:    viewStates,
		SelectedState: selectedState,
		Views:         views,
	})
}

type viewState struct {
	State string
	Count int
}

func getViewStates(lastNHours int) (viewStates []*viewState, err error) {
	var rows *sql.Rows
	viewStateSQL := `
      SELECT * FROM (
        SELECT v.state, COUNT(v.*) AS count
        FROM track.view v
        WHERE current_timestamp - date < ($1::int * interval '1 hour')
        GROUP BY state
      ) q ORDER BY count DESC
`
	rows, err = track.DB.Query(viewStateSQL, lastNHours)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		vs := new(viewState)
		err = rows.Scan(&vs.State, &vs.Count)
		if err != nil {
			return
		}
		viewStates = append(viewStates, vs)
	}
	return
}

var uiViewsHTML = `
<h1>Views</h1>
<div class="row">
  <div class="col-md-2">
    <form action="views" method="get" class="form">
      {{if .SelectedState}}<input type="hidden" name="state" value="{{.SelectedState}}">{{end}}
      <div class="form-group">
        <label for="lastNHours">Last # hours</label>
        <input type="number" class="form-control" id="lastNHours" name="lastNHours" placeholder="#" value="{{.LastNHours}}">
      </div>
      <div class="form-group">
        <label>Sort order:</label>
        <div class="radio">
          <label><input type="radio" name="sort" value="date" {{if eq .Sort "date"}}checked{{end}}> Most recent</label>
        </div>
      </div>
      <button type="submit" class="btn btn-primary">Update list</button>
    </form>
  </div>
  <div class="col-md-3">
    <div class="list-group">
    {{$LastNHours := .LastNHours}}
    {{$Sort := .Sort}}
    {{$SelectedState := .SelectedState}}
    {{range .ViewStates}}
      <a href="views?sort={{$Sort}}&lastNHours={{$LastNHours}}&state={{.State}}" class="list-group-item {{if eq $SelectedState .State}}active{{end}}">
        {{.State}}
        <span class="badge">{{.Count}}</span>
      </a>
    {{else}}
      <li><div class="alert alert-error">No states to show.</div></li>
    {{end}}
    </div>
  </div>
  <div class="col-md-7">
     {{if eq .SelectedState ""}}
       <div class="alert alert-warning">Select a state.</div>
     {{else}}
       <table class="table">
         <thead><tr><th>Date</th><th>URL</th>
         <tbody>
           {{range .Views}}
             <tr>
               <td>
                 {{.Date.Format "2006-01-02 15:04:05"}}<br>
                 <span class="text-muted">{{timeAgo .Date}}</span>
               </td>
               <td style="word-wrap:break-word;max-width:200px;"><a href="{{.RequestURI}}" target="_blank">{{.RequestURI}}</a></td>
             </tr>
           {{else}}
             <tr><td colspan="5" class="alert alert-warning">No views found for state {{$SelectedState}}.</td></tr>
           {{end}}
         </tbody>
       </table>
     {{end}}
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
	okSorts := map[string]struct{}{"date": struct{}{}, "duration": struct{}{}}
	if _, ok := okSorts[sort]; !ok {
		http.Error(w, "bad 'sort' parameter", http.StatusBadRequest)
		return
	}

	callRoutes, err := getCallRoutes(lastNHours, failedOnly)
	if err != nil {
		http.Error(w, "getCallRoutes failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var calls []*track.CallWithStatus
	selectedRoute := q.Get("route")
	if selectedRoute != "" {
		calls, err = track.QueryCallsWithStatus(`WHERE route = $1 AND (current_timestamp - date < ($2::int * interval '1 hour')) AND ((NOT $3) OR (http_status_code < 200 OR http_status_code >= 400 OR panicked)) ORDER BY `+sort+` DESC LIMIT 100`, selectedRoute, lastNHours, failedOnly)
		if err != nil {
			http.Error(w, "QueryCalls failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	tmpl(trackUICalls, uiCallsHTML)(w, struct {
		common
		LastNHours    int
		FailedOnly    bool
		Sort          string
		CallRoutes    []*callRoute
		SelectedRoute string
		Calls         []*track.CallWithStatus
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
        SELECT c.route, COUNT(c.*) AS count, ROUND(AVG(COALESCE(cs.duration, 0))::bigint, -6) AS avg_duration
        FROM track.call c
        LEFT JOIN track.call_status cs ON cs.call_id = c.id
        WHERE current_timestamp - date < ($1::int * interval '1 hour')
          AND ((NOT $2) OR (http_status_code < 200 OR http_status_code >= 400 OR panicked))
        GROUP BY route
      ) q ORDER BY count DESC
`
	rows, err = track.DB.Query(callRouteSQL, lastNHours, failedOnly)
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
        {{.Route}}
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
         <thead><tr><th>Date</th><th>URL</th><th>Duration</th><th>Bytes</th><th>Status</th></thead>
         <tbody>
           {{range .Calls}}
             <tr class="{{if isHTTPError .HTTPStatusCode}}danger{{end}}">
               <td>
                 {{.Date.Format "2006-01-02 15:04:05"}}<br>
                 <span class="text-muted">{{timeAgo .Date}}</span>
               </td>
               <td style="word-wrap:break-word;max-width:200px;"><a href="{{.URL}}" target="_blank">{{.URL}}</a></td>
               <td>{{duration (roundMillion .Duration)}}</td>
               <td>{{bytes .BodyLength}}</td>
               <td>{{.HTTPStatusCode}}</td>
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
	tmpl(trackUIMain, uiMainHTML)(w, newCommon("Main"))
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
    <title>{{.Title}} - Track</title>
    <meta charset="utf-8">
    <link rel="shortcut icon" href="data:image/x-icon;," type="image/x-icon"> 
    <link href="//netdna.bootstrapcdn.com/bootstrap/3.0.0/css/bootstrap.min.css" rel="stylesheet">
  </head>
  <body>
    <div class="container">
      <div class="navbar navbar-default">
        <div class="navbar-header">
          <button type="button" class="navbar-toggle" data-toggle="collapse" data-target=".navbar-collapse">
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
            <span class="icon-bar"></span>
          </button>
          <a class="navbar-brand" href="#">Track</a>
        </div>
        <div class="navbar-collapse collapse">
          <ul class="nav navbar-nav">
            <li><a href="users">Users</a></li>
            <li><a href="views">Views</a></li>
            <li><a href="calls">Calls</a></li>
          </ul>
        </div><!--/.nav-collapse -->
      </div>
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
			"duration":     func(nano int64) time.Duration { return time.Duration(nano) },
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
