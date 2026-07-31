// Shared compliance_frameworks reconciliation for custom alert resources.
package alert_common

import "fmt"

// API merges posted frameworks; clear first when old and new are both set.
// clearedButFailed: clear succeeded, update failed—caller must persist empty frameworks.
func ReplaceFrameworks(hadOld, hasNew bool, clear, update func() error) (clearedButFailed bool, err error) {
	if hadOld && hasNew {
		if err := clear(); err != nil {
			return false, fmt.Errorf("could not clear the existing compliance frameworks: %w", err)
		}
		if err := update(); err != nil {
			return true, fmt.Errorf("could not update alert, unexpected error: %w", err)
		}
		return false, nil
	}
	if err := update(); err != nil {
		return false, fmt.Errorf("could not update alert, unexpected error: %w", err)
	}
	return false, nil
}
