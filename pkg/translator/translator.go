package translator

import (
	"github.com/solo-io/sg/pkg/api/external/istio/networking/v1alpha3"
)

// A container for the entire set of istio configuration
type IstioConfig struct {
	DesinationRules v1alpha3.DestinationRuleList
	VirtualServices v1alpha3.VirtualService
}

// todo: first create all desintation rules for all subsets of each upstream
// then we need to apply the MUTUAL or ISTIO_MUTUAL policy depending on
// whether mtls is enabled, and if so, if the user is using a selfsignedcert
// if MUTUAL, also need to provide the paths for the certs/keys
// i assume these are loaded to pilot somewhere from a secret