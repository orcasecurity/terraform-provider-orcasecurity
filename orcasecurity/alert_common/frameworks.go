// Shared compliance_frameworks reconciliation for custom alert resources.
package alert_common

func ClearFramesIfReplacing(hadOld, hasNew bool, doClear func() error) (cleared bool, err error) {
	if !hadOld || !hasNew {
		return false, nil
	}
	return true, doClear()
}
