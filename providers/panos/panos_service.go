// SPDX-License-Identifier: Apache-2.0

package panos

import (
	"errors"

	"github.com/chenrui333/terraformer/terraformutils"
)

type PanosService struct { //nolint
	terraformutils.Service
	client interface{}
	vsys   string
}

func (p *PanosService) Initialize() error {
	if _, ok := p.Args["vsys"].(string); ok {
		p.vsys = p.Args["vsys"].(string)
	} else {
		return errors.New(p.GetName() + ": " + "vsys name not parsable")
	}

	c, err := Initialize()
	if err != nil {
		return err
	}

	p.client = c

	return nil
}
