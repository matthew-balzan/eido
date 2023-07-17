package vars

import "github.com/matthew-balzan/eido/internal/models"

var (
	Config    *models.Config
	Instances = map[string]*models.ServerInstance{}
)
