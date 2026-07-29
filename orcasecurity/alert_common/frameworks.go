// Package alert_common holds logic shared by the custom_discovery_alert and custom_sonar_alert
// resources for reconciling compliance_frameworks against the Orca alert API.
package alert_common

func ClearFramesIfReplacing(hadOld, hasNew bool, doClear func() error) (cleared bool, err error) {
	if !hadOld || !hasNew {
		return false, nil
	}
	return true, doClear()
}
