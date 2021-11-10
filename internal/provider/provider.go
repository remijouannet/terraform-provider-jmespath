package provider

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func New() *schema.Provider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"jmespath": dataSource(),
		},
		ResourcesMap: map[string]*schema.Resource{},
	}
}
