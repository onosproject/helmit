// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package job

import "os"

// Run runs the job
func Run(job *Job) error {
	coordinator := newRunner(job.Namespace, false)
	status, err := coordinator.RunJob(job)
	if err != nil {
		return err
	}
	os.Exit(status)
	return nil
}
