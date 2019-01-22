package appctx

import (
	"fmt"
	"os"
)

// PidfileCreate creates pid file with current process PID
func PidfileCreate(pidfile string) error {

	if len(pidfile) == 0 {
		return nil
	}

	// Check Pidfile exists
	if _, err := os.Stat(pidfile); err == nil {
		return fmt.Errorf("pidfile already exists: %s", pidfile)
	}

	f, err := os.OpenFile(pidfile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("pidfile create error: %s", err)
	}
	defer f.Close()

	fmt.Fprintf(f, "%d", os.Getpid())

	return nil
}

// PidfileRemove deletes specified pid file
func PidfileRemove(pidfile string) error {

	if len(pidfile) == 0 {
		return nil
	}

	if err := os.Remove(pidfile); err != nil {
		return fmt.Errorf("pidfile remove error: %s", err)
	}

	return nil
}

// PidfileChange changes pid file: remove old pid file (if `pidfileFrom` is set)
// and create the new one (if `pidfileTo` is set).
func PidfileChange(pidfileFrom, pidfileTo string) error {

	if pidfileFrom == pidfileTo {
		return nil
	}

	if len(pidfileTo) > 0 {
		if err := PidfileCreate(pidfileTo); err != nil {
			return err
		}
	}

	if len(pidfileFrom) > 0 {
		return PidfileRemove(pidfileFrom)
	}

	return nil
}
