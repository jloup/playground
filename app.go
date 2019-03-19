package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
)

func exitOnError(err error) {
	if err == nil {
		return
	}

	fmt.Println(err)
	fmt.Printf("EXITING\n")
	os.Exit(1)
}

type Params map[string]string

func (p Params) GetString(key string, defaultValue string) string {
	if v, exists := p[key]; exists {
		return v
	}

	return defaultValue
}

func (p Params) GetDecimal(key string, defaultValue decimal.Decimal) decimal.Decimal {
	if v, exists := p[key]; exists {
		d, err := decimal.NewFromString(v)
		exitOnError(err)
		return d
	}

	return defaultValue
}

func main() {
	wsChannels := make(map[string]chan interface{})

	http.HandleFunc("/", mainHandler(wsChannels))
	http.HandleFunc("/script.js", jsFileHandler)
	http.HandleFunc("/socket", wshandler(wsChannels))
	fmt.Printf("Server listening at 'http://localhost:8080'\n")
	err := http.ListenAndServe(":8080", nil)
	exitOnError(err)
}

func errorResponse(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	fmt.Fprintf(w, "ERROR %s", err)
}

func jsFileHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "script.js")
}

func queryToMap(v url.Values) map[string]string {
	m := make(map[string]string)

	for key, values := range v {
		m[key] = values[0]
	}

	return m
}

func mainHandler(channels map[string]chan interface{}) http.HandlerFunc {
	t, err := template.New("playground").Parse(tpl)
	exitOnError(err)
	t, err = t.ParseFiles("index.html")
	exitOnError(err)

	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			return
		}

		channelUUID := uuid.Must(uuid.NewV4())
		channels[channelUUID.String()] = make(chan interface{})

		data, err := main_(queryToMap(r.URL.Query()), channels[channelUUID.String()])
		if err != nil {
			errorResponse(w, err)
			return
		}

		b, err := json.Marshal(data)
		if err != nil {
			errorResponse(w, err)
			return
		}

		err = t.Execute(w, struct {
			Data string
			UUID string
		}{
			string(b),
			channelUUID.String(),
		})
		if err != nil {
			errorResponse(w, err)
			return
		}
	}
}

func wshandler(channels map[string]chan interface{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var upgrader = websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		exitOnError(err)

		_, p, err := conn.ReadMessage()
		exitOnError(err)

		uuid := string(p)
		dataChan := channels[uuid]

		for {
			event := <-dataChan
			b, err := json.Marshal(event)
			exitOnError(err)

			err = conn.WriteMessage(websocket.TextMessage, b)
			if err != nil {
				fmt.Println(err)
				conn.Close()
				return
			}
		}

	}
}

var tpl string = `
<head>
  <meta charset="utf-8">
  <title>Playground</title>
  <meta name="author" content="">
  <meta name="description" content="">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body {
      width: 1400px;
      margin-right: auto;
      margin-left: auto;
      margin-top: 3cm;
      margin-bottom: 2cm;
      font-size: 1.2em;
    }

    li {
      list-style-type: none;
    }

  </style>
</head>

<body>
  {{template "index.html"}}
</body>
<script type="text/javascript" src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.7.3/Chart.min.js"></script>
<script type="text/javascript" src="script.js"></script>
<script type="text/javascript">
  window.data = JSON.parse({{.Data}});

  window.onData(window.data);

  var socket = new WebSocket("ws://localhost:8080/socket");

  console.log("WS UUID: {{.UUID}}");
  socket.onopen = function() {
    socket.send("{{.UUID}}");
  }

  socket.onmessage = function(event) {
    window.data.cpy = JSON.parse(event.data);
    window.onData(window.data);
  }
</script>
</html>
`
