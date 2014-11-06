package http

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/mcuadros/dockership/config"
	"github.com/mcuadros/dockership/core"

	"github.com/googollee/go-socket.io"
	"github.com/gorilla/mux"
)

var configFile string

func Start(version, build string) {
	core.Info("Starting HTTP daemon", "version", version, "build", build)
	flag.StringVar(&configFile, "config", config.DEFAULT_CONFIG, "config file")
	flag.Parse()

	s := &server{serverId: fmt.Sprintf("dockership %s, build %s", version, build)}
	s.readConfig(configFile)
	s.configure()
	s.configureAuth()
	s.run()
}

type server struct {
	serverId string
	socketio *socketio.Server
	mux      *mux.Router
	oauth    *OAuth
	config   config.Config
}

func (s *server) configure() {
	s.mux = mux.NewRouter()

	var err error
	s.socketio, err = socketio.NewServer(nil)
	if err != nil {
		fmt.Println(err)
	}

	s.socketio.On("connection", func(so socketio.Socket) {
		so.Join("deploy")
		fmt.Println("hola")
	})

	s.socketio.On("error", func(so socketio.Socket, err error) {
		fmt.Println("error")
	})

	s.mux.Path("/socket.io/").Handler(s.socketio)

	// status
	s.mux.Path("/rest/status").Methods("GET").HandlerFunc(s.HandleStatus)
	s.mux.Path("/rest/status/{project:.*}").Methods("GET").HandlerFunc(s.HandleStatus)

	// containers
	s.mux.Path("/rest/containers").Methods("GET").HandlerFunc(s.HandleContainers)
	s.mux.Path("/rest/containers/{project:.*}").Methods("GET").HandlerFunc(s.HandleContainers)

	// deploy
	s.mux.Path("/rest/deploy/{project:.*}/{environment:.*}").Methods("GET").HandlerFunc(s.HandleDeploy)

	// logged-user
	s.mux.Path("/rest/user").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := s.oauth.getUser(s.oauth.getToken(r))
		s.json(w, 200, user)
	})

	// assets
	s.mux.Path("/").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		content, _ := Asset("static/index.html")
		w.Write(content)
	})

	s.mux.Path("/dockership.png").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		content, _ := Asset("static/dockership.png")
		w.Write(content)
	})

	s.mux.Path("/app.js").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		content, _ := Asset("static/app.js")
		w.Write(content)
	})
}

func (s *server) configureAuth() {
	s.oauth = NewOAuth(&s.config)
}

func (s *server) readConfig(configFile string) {
	if err := s.config.LoadFile(configFile); err != nil {
		panic(err)
	}
}

func (s *server) run() {
	core.Info("HTTP server running", "host:port", s.config.HTTP.Listen)
	if err := http.ListenAndServe(s.config.HTTP.Listen, s); err != nil {
		panic(err)
	}
}

func (s *server) json(w http.ResponseWriter, code int, response interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	encoder := json.NewEncoder(w)
	encoder.Encode(response)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.oauth.Handler(w, r) {
		core.Debug("Handling request", "url", r.URL)
		w.Header().Set("Server", s.serverId)
		s.mux.ServeHTTP(w, r)
	}
}

type SocketioWriter struct {
	socketio *socketio.Server
	room     string
	message  string
}

func NewSocketioWriter(socketio *socketio.Server, room, message string) *SocketioWriter {
	return &SocketioWriter{
		socketio: socketio,
		room:     room,
		message:  message,
	}
}

func (s *SocketioWriter) Write(p []byte) (int, error) {
	s.socketio.BroadcastTo(s.room, s.message, string(p))
	return len(p), nil
}
