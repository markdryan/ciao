// // Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package testutil

import (
	"errors"
	"fmt"
	"sync"

	"github.com/01org/ciao/payloads"
	"github.com/01org/ciao/ssntp"
	"gopkg.in/yaml.v2"
)

// SsntpTestController is global state for the testutil SSNTP controller
type SsntpTestController struct {
	Ssntp          ssntp.Client
	Name           string
	UUID           string
	CmdChans       map[ssntp.Command]chan CmdResult
	CmdChansLock   *sync.Mutex
	EventChans     map[ssntp.Event]chan CmdResult
	EventChansLock *sync.Mutex
}

// NewSsntpTestControllerConnection creates an SsntpTestController and dials the server.
// Calling with a unique name parameter string for inclusion in the
// SsntpTestClient.Name field aides in debugging.  The uuid string
// parameter allows tests to specify a known uuid for simpler tests.
func NewSsntpTestControllerConnection(name string, uuid string) (*SsntpTestController, error) {
	if uuid == "" {
		return nil, errors.New("no uuid specified")
	}

	var role ssntp.Role = ssntp.Controller
	ctl := &SsntpTestController{
		Name: "Test " + role.String() + " " + name,
		UUID: uuid,
	}

	ctl.CmdChans = make(map[ssntp.Command]chan CmdResult)
	ctl.CmdChansLock = &sync.Mutex{}
	ctl.EventChans = make(map[ssntp.Event]chan CmdResult)
	ctl.EventChansLock = &sync.Mutex{}

	config := &ssntp.Config{
		URI:    "",
		CAcert: ssntp.DefaultCACert,
		Cert:   ssntp.RoleToDefaultCertName(ssntp.Controller),
		Log:    ssntp.Log,
		UUID:   ctl.UUID,
	}

	if err := ctl.Ssntp.Dial(config, ctl); err != nil {
		return nil, err
	}
	return ctl, nil
}

// AddCmdChan adds a command to the command channel to the SsntpTestServer
func (ctl *SsntpTestController) AddCmdChan(cmd ssntp.Command, ch chan CmdResult) {
	ctl.CmdChansLock.Lock()
	ctl.CmdChans[cmd] = ch
	ctl.CmdChansLock.Unlock()
}

// AddEventChan adds a command to the command channel to the SsntpTestServer
func (ctl *SsntpTestController) AddEventChan(cmd ssntp.Event, ch chan CmdResult) {
	ctl.EventChansLock.Lock()
	ctl.EventChans[cmd] = ch
	ctl.EventChansLock.Unlock()
}

// ConnectNotify implements the SSNTP client ConnectNotify callback for SsntpTestController
func (ctl *SsntpTestController) ConnectNotify() {
	var result CmdResult

	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[ssntp.NodeConnected]
	if ok {
		delete(ctl.EventChans, ssntp.NodeConnected)
		c <- result
		close(c)
	}
}

// DisconnectNotify implements the SSNTP client DisconnectNotify callback for SsntpTestController
func (ctl *SsntpTestController) DisconnectNotify() {
	var result CmdResult

	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[ssntp.NodeDisconnected]
	if ok {
		delete(ctl.EventChans, ssntp.NodeDisconnected)
		c <- result
		close(c)
	}
}

// StatusNotify implements the SSNTP client StatusNotify callback for SsntpTestController
func (ctl *SsntpTestController) StatusNotify(status ssntp.Status, frame *ssntp.Frame) {
}

// CommandNotify implements the SSNTP client CommandNotify callback for SsntpTestController
func (ctl *SsntpTestController) CommandNotify(command ssntp.Command, frame *ssntp.Frame) {
	var result CmdResult

	switch command {
	case ssntp.STATS:
		var stats payloads.Stat

		stats.Init()

		err := yaml.Unmarshal(frame.Payload, &stats)
		if err != nil {
			result.Err = err
		}
	default:
		fmt.Printf("controller unhandled command: %s\n", command.String())
	}

	ctl.CmdChansLock.Lock()
	defer ctl.CmdChansLock.Unlock()
	c, ok := ctl.CmdChans[command]
	if ok {
		delete(ctl.CmdChans, command)
		c <- result
		close(c)
	}
}

// EventNotify implements the SSNTP client EventNotify callback for SsntpTestController
func (ctl *SsntpTestController) EventNotify(event ssntp.Event, frame *ssntp.Frame) {
	var result CmdResult

	switch event {
	case ssntp.InstanceDeleted:
		var deleteEvent payloads.EventInstanceDeleted

		err := yaml.Unmarshal(frame.Payload, &deleteEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.TraceReport:
		var traceEvent payloads.Trace

		err := yaml.Unmarshal(frame.Payload, &traceEvent)
		if err != nil {
			result.Err = err
		}
	case ssntp.ConcentratorInstanceAdded:
		var concentratorAddedEvent payloads.EventConcentratorInstanceAdded

		err := yaml.Unmarshal(frame.Payload, &concentratorAddedEvent)
		if err != nil {
			result.Err = err
		}
	default:
		fmt.Printf("controller unhandled event: %s\n", event.String())
	}

	ctl.EventChansLock.Lock()
	defer ctl.EventChansLock.Unlock()
	c, ok := ctl.EventChans[event]
	if ok {
		delete(ctl.EventChans, event)
		c <- result
		close(c)
	}
}

// ErrorNotify implements the SSNTP client ErrorNotify callback for SsntpTestController
func (ctl *SsntpTestController) ErrorNotify(error ssntp.Error, frame *ssntp.Frame) {
}
