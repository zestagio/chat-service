package serverdebug

import (
	"html/template"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
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
			<option{{ if eq .LogLevel "DEBUG" }} selected{{ end }}>DEBUG</option>
			<option{{ if eq .LogLevel "INFO" }} selected{{ end }}>INFO</option>
			<option{{ if eq .LogLevel "WARN" }} selected{{ end }}>WARN</option>
			<option{{ if eq .LogLevel "ERROR" }} selected{{ end }}>ERROR</option>
		</select>
		<input type="submit" value="Change"></input>
	</form>

	<script>
		function putLogLevel() {
			const req = new XMLHttpRequest();
			req.open('PUT', '/log/level', false);
			req.setRequestHeader('Content-Type', 'application/json');
			req.onload = function() { window.location.reload(); };
			req.send(JSON.stringify({"level": document.getElementById('log-level-select').value}));
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
			LogLevel: zap.L().Level().CapitalString(),
		},
	)
}
