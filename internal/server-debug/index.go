package serverdebug

import (
	"html/template"

	"github.com/labstack/echo/v4"

	"github.com/zestagio/chat-service/internal/logger"
)

type page struct {
	Path        string
	Description string
}

type indexPage struct {
	pages []page
}

func newIndexPage() *indexPage {
	return &indexPage{}
}

func (i *indexPage) addPage(path string, description string) {
	i.pages = append(i.pages, page{path, description})
}

func (i indexPage) handler(eCtx echo.Context) error {
	return template.Must(
		template.New("index").Parse(
			`<html>
	<title>Chat Service Debug</title>
<body>
	<h2>Chat Service Debug</h2>
	<ul>
		{{range .Pages}}
		<li><a href="{{.Path}}">{{.Path}}</a> {{.Description}}</li>
		{{end}}
	</ul>

	<h2>Log Level</h2>
	<form onSubmit="putLogLevel()">
		<select id="log-level-select">
			<option value="debug" {{ if eq .LogLevel "debug" }}selected{{ end }}>DEBUG</option>
			<option value="info" {{ if eq .LogLevel "info" }}selected{{ end }}>INFO</option>
			<option value="warn" {{ if eq .LogLevel "warn" }}selected{{ end }}>WARN</option>
			<option value="error" {{ if eq .LogLevel "error" }}selected{{ end }}>ERROR</option>
		</select>
		<input type="submit" value="Change"></input>
	</form>

	<script>
		function putLogLevel() {
			const req = new XMLHttpRequest();
			req.open('PUT', '/log/level', false);
			req.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
			req.setRequestHeader('Accept', 'application/json');
			req.onload = function() { window.location.reload(); };
			req.send('level='+document.getElementById('log-level-select').value);
		};
	</script>
</body>
</html>
`,
		),
	).Execute(
		eCtx.Response(), struct {
			Pages    []page
			LogLevel string
		}{
			Pages:    i.pages,
			LogLevel: logger.Level.String(),
		},
	)
}
