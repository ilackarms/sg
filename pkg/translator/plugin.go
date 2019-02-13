package translator

import (
	"github.com/solo-io/supergloo/pkg/api/v1"
	"github.com/solo-io/supergloo/pkg/translator/utils"
)

// destinationrules
func destinationRulesForUpstreams(rules v1.RoutingRuleList, meshes v1.MeshList, upstreams gloov1.UpstreamList) (v1alpha3.DestinationRuleList, error) {
	var meshesWithRouteRules v1.MeshList
	for _, rule := range rules {
		mesh, err := meshes.Find(rule.TargetMesh.Namespace, rule.TargetMesh.Name)
		if err != nil {
			return nil, errors.Wrapf(err, "finding target mesh %v", rule.TargetMesh)
		}
		if _, err := getIstioMeshForRule(rule, meshes); err != nil {
			return nil, err
		}
		var found bool
		for _, addedMesh := range meshesWithRouteRules {
			if mesh == addedMesh {
				found = true
				break
			}
		}
		if !found {
			meshesWithRouteRules = append(meshesWithRouteRules, mesh)
		}
	}
	if len(meshesWithRouteRules) == 0 {
		return nil, nil
	}

	var destinationRules v1alpha3.DestinationRuleList
	for _, mesh := range meshesWithRouteRules {
		mtlsEnabled := mesh.Encryption != nil && mesh.Encryption.TlsEnabled
		labelsByHost := make(map[string][]map[string]string)
		for _, us := range upstreams {
			labels := utils.GetLabelsForUpstream(us)
			host, err := utils.GetHostForUpstream(us)
			if err != nil {
				return nil, errors.Wrapf(err, "getting host for upstream")
			}
			labelsByHost[host] = append(labelsByHost[host], labels)
		}
		for host, labelSets := range labelsByHost {
			var subsets []*v1alpha3.Subset
			for _, labels := range labelSets {
				if len(labels) == 0 {
					continue
				}
				subsets = append(subsets, &v1alpha3.Subset{
					Name:   subsetName(labels),
					Labels: labels,
				})
			}
			var trafficPolicy *v1alpha3.TrafficPolicy
			if mtlsEnabled {
				trafficPolicy = &v1alpha3.TrafficPolicy{
					Tls: &v1alpha3.TLSSettings{
						Mode: v1alpha3.TLSSettings_ISTIO_MUTUAL,
					},
				}
			}
			destinationRules = append(destinationRules, &v1alpha3.DestinationRule{
				Metadata: core.Metadata{
					Namespace: mesh.Metadata.Namespace,
					Name:      mesh.Metadata.Name + "-" + host,
				},
				Host:          host,
				TrafficPolicy: trafficPolicy,
				Subsets:       subsets,
			})
		}
	}

	return destinationRules.Sort(), nil
}
