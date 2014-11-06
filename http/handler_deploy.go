package http

import (
	"bytes"
	"fmt"

	"github.com/mcuadros/dockership/core"

	"gopkg.in/igm/sockjs-go.v2/sockjs"
)

func (s *server) HandleDeploy(msg Message, session sockjs.Session) {

	//writer := NewSocketioWriter(s.socketio, "deploy", "foo")

	//subs := subscribeWriteToEvents(writer)
	//defer unsubscribeEvents(subs)

	writer := bytes.NewBuffer([]byte(""))
	force := true

	fmt.Println(msg.Request)
	project, ok := msg.Request["project"]
	if !ok {
		core.Error("Missing project", "request", "deploy")
		return
	}

	environment, ok := msg.Request["environment"]
	if !ok {
		core.Error("Missing environment", "request", "deploy")
		return
	}

	if p, ok := s.config.Projects[project]; ok {
		core.Info("Starting deploy", "project", p, "environment", environment, "force", force)
		err := p.Deploy(environment, writer, force)
		if len(err) != 0 {
			for _, e := range err {
				core.Critical(e.Error(), "project", project)
			}
		} else {
			core.Info("Deploy success", "project", p, "environment", environment)
		}
	} else {
		core.Error("Project not found", "project", p)
	}
}
