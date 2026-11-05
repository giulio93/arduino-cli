// This file is part of arduino-cli.
//
// Copyright 2020 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the GNU General Public License version 3,
// which covers the main part of arduino-cli.
// The terms of this license can be found at:
// https://www.gnu.org/licenses/gpl-3.0.en.html
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package feedback

import (
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
)

// ProgressBar returns a DownloadProgressCB that prints a progress bar.
func ProgressBar() rpc.DownloadProgressCB {
	if format == Text {
		return NewDownloadProgressBarCB()
	}
	return func(curr *rpc.DownloadProgress) {
		// Non interactive output, no progress bar
	}
}

// TaskProgress returns a TaskProgressCB that prints the task progress.
func TaskProgress() rpc.TaskProgressCB {
	if format == Text {
		return NewTaskProgressCB()
	}
	return func(curr *rpc.TaskProgress) {
		// Non interactive output, no task progess
	}
}

// NewTaskProgressCB returns a commands.TaskProgressCB progress listener
// that outputs to terminal
func NewTaskProgressCB() func(curr *rpc.TaskProgress) {
	return func(curr *rpc.TaskProgress) {
		msg := curr.GetMessage()
		if msg == "" {
			msg = curr.GetName()
		}
		if msg != "" {
			if !curr.GetCompleted() {
				msg += "..."
			}
			Print(msg)
		}
	}
}
