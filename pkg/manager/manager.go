// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"sample-xapp/pkg/southbound"
	"strconv"
	"strings"

	"github.com/onosproject/onos-lib-go/pkg/logging"
)

var log = logging.GetLogger("manager")

type Config struct {
	CAPath      string
	KeyPath     string
	CertPath    string
	E2tEndpoint string
	GRPCPort    int
	SMName      string
	SMVersion   string
}

func NewManager(config Config) *Manager {
	return &Manager{
		config: config,
	}
}

type Manager struct {
	config Config
}

func (m *Manager) Start() error {
	log.Info("Start manager")

	e2tAddr := strings.Split(m.config.E2tEndpoint, ":")[0]
	e2tPort, err := strconv.Atoi(strings.Split(m.config.E2tEndpoint, ":")[1])
	if err != nil {
		return err
	}

	e2SubManager, err := southbound.NewE2SubManager(m.config.SMName, m.config.SMVersion, e2tAddr, e2tPort)
	if err != nil {
		return err
	}

	e2SubManager.Start()
	return nil
}
