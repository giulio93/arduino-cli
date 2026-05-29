// This file is part of arduino-cli.
//
// Copyright 2025 ARDUINO SA (http://www.arduino.cc/)
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

package detector

import (
	"os"
	"testing"
	"time"

	"github.com/arduino/go-paths-helper"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// TestSketchPreprocessingUpToDate_ReproducesIssue3202 pins the race condition
// that caused stale-cache hits on large/library-heavy projects (issue #3202).
//
// On a large project the GCC preprocessing depfile is written by GCC *after*
// sketch.ino.cpp.merged in every successful build. If the user edits the
// sketch and immediately recompiles, the new .cpp.merged mtime can land
// between the preprocessed-output mtime and the old depfile mtime:
//
//	T0 (.ino.cpp) < T1 (new .cpp.merged) < T2 (depfile)
//
// The old sourceFile.ObjFileIsUpToDate check compared merged source vs the
// depfile. With T1 < T2 it returned "unchanged" even though the source was
// edited — allowing the stale .ino.cpp.o to be reused.
//
// The fix, isSketchPreprocessingUpToDate, compares merged source vs .ino.cpp.
// With T1 > T0 it correctly returns "changed", triggering re-preprocessing.
func TestSketchPreprocessingUpToDate_ReproducesIssue3202(t *testing.T) {
	dir := paths.New(t.TempDir())
	mergedSource := dir.Join("sketch.ino.cpp.merged")
	preprocessed := dir.Join("sketch.ino.cpp")
	depfile := dir.Join("sketch.ino.cpp.merged.d")

	now := time.Now()
	T0 := now.Add(-5 * time.Second) // .ino.cpp   — preprocessed output from the last build
	T1 := now.Add(-3 * time.Second) // .cpp.merged — user edited sketch, then hit compile
	T2 := now.Add(-1 * time.Second) // depfile     — GCC wrote it late in the large build

	write := func(p *paths.Path, content string, mtime time.Time) {
		t.Helper()
		require.NoError(t, p.WriteFile([]byte(content)))
		require.NoError(t, os.Chtimes(p.String(), mtime, mtime))
	}

	write(preprocessed, "// preprocessed output from previous build\nvoid setup(){} void loop(){}", T0)
	write(mergedSource, "void setup(){} void loop(){} // EDITED", T1)
	// Depfile first dep must match the source path (checked by ObjFileIsUpToDate).
	write(depfile, "/dev/null: \\\n\t"+mergedSource.String()+"\n", T2)

	sf := sourceFile{SourcePath: mergedSource, DepfilePath: depfile}

	// OLD behaviour (the bug): merged (T1) is older than depfile (T2) so
	// ObjFileIsUpToDate reads the deps, finds nothing newer than the depfile,
	// and incorrectly declares the sketch unchanged.
	t.Run("OldCheck_FalsePositive", func(t *testing.T) {
		unchanged, err := sf.ObjFileIsUpToDate(logrus.WithField("test", t.Name()))
		require.NoError(t, err)
		require.True(t, unchanged,
			"old depfile-based check incorrectly reports 'unchanged' for an edited sketch (reproduces #3202)")
	})

	// NEW behaviour (the fix): merged (T1) is newer than .ino.cpp (T0) so
	// isSketchPreprocessingUpToDate correctly reports that re-preprocessing is needed.
	t.Run("NewCheck_CorrectlyDetectsEdit", func(t *testing.T) {
		unchanged := isSketchPreprocessingUpToDate(mergedSource, preprocessed)
		require.False(t, unchanged,
			"new check must detect that the edited sketch needs re-preprocessing")
	})
}

// TestSketchPreprocessingUpToDate_NoRebuildNeeded verifies that the fix does
// not trigger unnecessary re-preprocessing when the sketch is unmodified.
func TestSketchPreprocessingUpToDate_NoRebuildNeeded(t *testing.T) {
	dir := paths.New(t.TempDir())
	mergedSource := dir.Join("sketch.ino.cpp.merged")
	preprocessed := dir.Join("sketch.ino.cpp")

	now := time.Now()
	require.NoError(t, mergedSource.WriteFile([]byte("void setup(){} void loop(){}")))
	require.NoError(t, os.Chtimes(mergedSource.String(), now.Add(-2*time.Second), now.Add(-2*time.Second)))
	require.NoError(t, preprocessed.WriteFile([]byte("// preprocessed")))
	require.NoError(t, os.Chtimes(preprocessed.String(), now.Add(-1*time.Second), now.Add(-1*time.Second)))

	unchanged := isSketchPreprocessingUpToDate(mergedSource, preprocessed)
	require.True(t, unchanged, "preprocessed output is newer than merged source — no rebuild needed")
}

// TestSketchPreprocessingUpToDate_MissingPreprocessed verifies that the check
// triggers re-preprocessing when sketch.ino.cpp does not yet exist (first
// build, or after the cache was cleared).
func TestSketchPreprocessingUpToDate_MissingPreprocessed(t *testing.T) {
	dir := paths.New(t.TempDir())
	mergedSource := dir.Join("sketch.ino.cpp.merged")
	preprocessed := dir.Join("sketch.ino.cpp") // intentionally absent

	require.NoError(t, mergedSource.WriteFile([]byte("void setup(){} void loop(){}")))

	unchanged := isSketchPreprocessingUpToDate(mergedSource, preprocessed)
	require.False(t, unchanged, "missing preprocessed file must trigger preprocessing")
}
