// Licensed to The Moov Authors under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. The Moov Authors licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package consul

import (
	"testing"

	"github.com/moov-io/base/log"
	"github.com/stretchr/testify/assert"
)

func TestAcquireLock(t *testing.T) {
	a := assert.New(t)
	logger := log.NewDefaultLogger()

	cc1, err := NewConsulClient(logger, &Config{
		Address:     "http://127.0.0.1:8500",
		SessionPath: "/",
	})
	t.Cleanup(func() {
		cc1.Shutdown()
	})
	a.Nil(err)

	err = cc1.AcquireLock("achgateway/odfi/testing")
	a.Nil(err)

	// Initialize another client and attempt leader election
	cc2, err := NewConsulClient(logger, &Config{
		Address:     "http://127.0.0.1:8500",
		SessionPath: "/",
	})
	t.Cleanup(func() {
		cc2.Shutdown()
	})
	a.Nil(err)

	err = cc2.AcquireLock("achgateway/odfi/testing")
	a.NotNil(err)
	a.Contains(err.Error(), `we are not the leader of achgateway/odfi/testing`)
}
