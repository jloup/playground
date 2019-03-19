# Golang playground

App served at localhost:8080

Edit `main_(params Params, dataChan chan interface{}) (interface{}, error)` in main.go
 - `params` are key/value parameters mapping to query string passed to http://localhost:8080
 - Any data sent to `dataChan` is sent to client via websocket.
 
Edit `onData(data)` in script.js
  - data is what is sent by `main_`
