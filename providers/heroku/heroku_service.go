// SPDX-License-Identifier: Apache-2.0

package heroku

import (
	"github.com/chenrui333/terraformer/terraformutils"
	heroku "github.com/heroku/heroku-go/v6"
)

type HerokuService struct { //nolint
	terraformutils.Service
}

func (s *HerokuService) generateService() *heroku.Service {
	heroku.DefaultTransport.Password = s.Args["api_key"].(string)
	heroku.DefaultTransport.Debug = s.Verbose
	return heroku.NewService(heroku.DefaultClient)
}
