package mutation

import "fmt"

const (
	errorOneAgentOperatorExists                = "OneAgentAPM object detected - DynaKube webhook won't inject until the OneAgent Operator has been uninstalled"
	errorNoFeaturesEnabled                     = "no features are enabled, skipping injection"
	errorDecodingPod                           = "failed to decode pod"
	errorFailedToQueryNamespace                = "failed to query namespace"
	errorFailedToQueryDynakube                 = "failed to query DynaKube"
	errorAppInjectionDisabled                  = "app injection is disabled"
	errorFailedToQueryInitSecret               = "failed to query the init secret before pod injection"
	errorCreatingInitSecret                    = "failed to create the init secret before pod injection"
	errorFailedToQueryDataIngestEndpointSecret = "failed to query the data-ingest endpoint secret before pod injection"
	errorCreatingQueryDataIngestEndpointSecret = "failed to create the data-ingest endpoint secret before pod injection"
)

const (
	errorDynakubeLabelNotSetTemplate             = "namespace '%s' has no DynaKube label"
	errorDynakubeAssignedButDoesNotExistTemplate = "DynaKube '%s' is assigned to namespace '%s', but DynaKube '%s' does not exist"
)

func errorDynakubeLabelNotSet(namespaceName string) string {
	return fmt.Sprintf(errorDynakubeLabelNotSetTemplate, namespaceName)
}

func errorDynakubeAssignedButDoesNotExist(dynakubeName string, namespaceName string) string {
	return fmt.Sprintf(errorDynakubeAssignedButDoesNotExistTemplate, dynakubeName, namespaceName, dynakubeName)
}