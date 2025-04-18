// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (c) 2014-2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package osutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	. "gopkg.in/check.v1"

	"github.com/canonical/pebble/internals/osutil"
	"github.com/canonical/pebble/internals/osutil/sys"
	"github.com/canonical/pebble/internals/testutil"
)

type AtomicWriteTestSuite struct{}

var _ = Suite(&AtomicWriteTestSuite{})

func (ts *AtomicWriteTestSuite) TestAtomicWriteFile(c *C) {
	tmpdir := c.MkDir()

	p := filepath.Join(tmpdir, "foo")
	err := osutil.AtomicWriteFile(p, []byte("canary"), 0644, 0)
	c.Assert(err, IsNil)

	c.Check(p, testutil.FileEquals, "canary")

	// no files left behind!
	d, err := os.ReadDir(tmpdir)
	c.Assert(err, IsNil)
	c.Assert(len(d), Equals, 1)
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFilePermissions(c *C) {
	tmpdir := c.MkDir()

	p := filepath.Join(tmpdir, "foo")
	err := osutil.AtomicWriteFile(p, []byte(""), 0600, 0)
	c.Assert(err, IsNil)

	st, err := os.Stat(p)
	c.Assert(err, IsNil)
	c.Assert(st.Mode()&os.ModePerm, Equals, os.FileMode(0600))
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileOverwrite(c *C) {
	tmpdir := c.MkDir()
	p := filepath.Join(tmpdir, "foo")
	c.Assert(os.WriteFile(p, []byte("hello"), 0644), IsNil)
	c.Assert(osutil.AtomicWriteFile(p, []byte("hi"), 0600, 0), IsNil)

	c.Assert(p, testutil.FileEquals, "hi")
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileSymlinkNoFollow(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink(s, p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	if os.Getuid() == 0 {
		c.Skip("requires running as non-root user")
	}
	err := osutil.AtomicWriteFile(p, []byte("hi"), 0600, 0)
	c.Assert(err, NotNil)
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileAbsoluteSymlinks(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink(s, p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	err := osutil.AtomicWriteFile(p, []byte("hi"), 0600, osutil.AtomicWriteFollow)
	c.Assert(err, IsNil)

	c.Assert(p, testutil.FileEquals, "hi")
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileChmod(c *C) {
	tmpdir := c.MkDir()
	oldmask := syscall.Umask(0222)
	defer syscall.Umask(oldmask)

	path := filepath.Join(tmpdir, "foo")
	err := osutil.AtomicWriteFile(path, []byte{}, 0777, osutil.AtomicWriteChmod)
	c.Assert(err, IsNil)

	st, err := os.Stat(path)
	c.Assert(err, IsNil)
	c.Assert(st.Mode()&os.ModePerm, Equals, os.FileMode(0777))
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileOverwriteAbsoluteSymlink(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink(s, p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	c.Assert(os.WriteFile(s, []byte("hello"), 0644), IsNil)
	c.Assert(osutil.AtomicWriteFile(p, []byte("hi"), 0600, osutil.AtomicWriteFollow), IsNil)

	c.Assert(p, testutil.FileEquals, "hi")
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileRelativeSymlinks(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink("../foo", p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	err := osutil.AtomicWriteFile(p, []byte("hi"), 0600, osutil.AtomicWriteFollow)
	c.Assert(err, IsNil)

	c.Assert(p, testutil.FileEquals, "hi")
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileOverwriteRelativeSymlink(c *C) {
	tmpdir := c.MkDir()
	rodir := filepath.Join(tmpdir, "ro")
	p := filepath.Join(rodir, "foo")
	s := filepath.Join(tmpdir, "foo")
	c.Assert(os.MkdirAll(rodir, 0755), IsNil)
	c.Assert(os.Symlink("../foo", p), IsNil)
	c.Assert(os.Chmod(rodir, 0500), IsNil)
	defer os.Chmod(rodir, 0700)

	c.Assert(os.WriteFile(s, []byte("hello"), 0644), IsNil)
	c.Assert(osutil.AtomicWriteFile(p, []byte("hi"), 0600, osutil.AtomicWriteFollow), IsNil)

	c.Assert(p, testutil.FileEquals, "hi")
}

func (ts *AtomicWriteTestSuite) TestAtomicWriteFileNoOverwriteTmpExisting(c *C) {
	tmpdir := c.MkDir()
	osutil.FakeRandomString(func(length int) string {
		// ensure we always get the same result
		return strings.Repeat("a", length)
	})

	p := filepath.Join(tmpdir, "foo")
	err := os.WriteFile(p+"."+strings.Repeat("a", 12)+"~", []byte(""), 0644)
	c.Assert(err, IsNil)

	err = osutil.AtomicWriteFile(p, []byte(""), 0600, 0)
	c.Assert(err, ErrorMatches, "open .*: file exists")
}

func (ts *AtomicWriteTestSuite) TestAtomicFileChownError(c *C) {
	eUid := sys.UserID(42)
	eGid := sys.GroupID(74)
	eErr := errors.New("this didn't work")
	defer osutil.FakeChown(func(fd *os.File, uid sys.UserID, gid sys.GroupID) error {
		c.Check(uid, Equals, eUid)
		c.Check(gid, Equals, eGid)
		return eErr
	})()

	d := c.MkDir()
	p := filepath.Join(d, "foo")

	aw, err := osutil.NewAtomicFile(p, 0644, 0, eUid, eGid)
	c.Assert(err, IsNil)
	defer aw.Cancel()

	_, err = aw.Write([]byte("hello"))
	c.Assert(err, IsNil)

	c.Check(aw.Commit(), Equals, eErr)
}

func (ts *AtomicWriteTestSuite) TestAtomicFileCancelError(c *C) {
	d := c.MkDir()
	p := filepath.Join(d, "foo")
	aw, err := osutil.NewAtomicFile(p, 0644, 0, osutil.NoChown, osutil.NoChown)
	c.Assert(err, IsNil)

	c.Assert(aw.File.Close(), IsNil)
	// Depending on golang version the error is one of the two.
	c.Check(aw.Cancel(), ErrorMatches, "invalid argument|file already closed")
}

func (ts *AtomicWriteTestSuite) TestAtomicFileCancelBadError(c *C) {
	d := c.MkDir()
	p := filepath.Join(d, "foo")
	aw, err := osutil.NewAtomicFile(p, 0644, 0, osutil.NoChown, osutil.NoChown)
	c.Assert(err, IsNil)
	defer aw.Close()

	osutil.SetAtomicFileRenamed(aw, true)

	c.Check(aw.Cancel(), Equals, osutil.ErrCannotCancel)
}

func (ts *AtomicWriteTestSuite) TestAtomicFileCancelNoClose(c *C) {
	d := c.MkDir()
	p := filepath.Join(d, "foo")
	aw, err := osutil.NewAtomicFile(p, 0644, 0, osutil.NoChown, osutil.NoChown)
	c.Assert(err, IsNil)
	c.Assert(aw.Close(), IsNil)

	c.Check(aw.Cancel(), IsNil)
}

func (ts *AtomicWriteTestSuite) TestAtomicFileCancel(c *C) {
	d := c.MkDir()
	p := filepath.Join(d, "foo")

	aw, err := osutil.NewAtomicFile(p, 0644, 0, osutil.NoChown, osutil.NoChown)
	c.Assert(err, IsNil)
	fn := aw.File.Name()
	c.Check(osutil.CanStat(fn), Equals, true)
	c.Check(aw.Cancel(), IsNil)
	c.Check(osutil.CanStat(fn), Equals, false)
}

// SafeIoAtomicWriteTestSuite runs all AtomicWrite with safe
// io enabled
type SafeIoAtomicWriteTestSuite struct {
	AtomicWriteTestSuite

	restoreUnsafeIO func()
}

var _ = Suite(&SafeIoAtomicWriteTestSuite{})

func (s *SafeIoAtomicWriteTestSuite) SetUpSuite(c *C) {
	s.restoreUnsafeIO = osutil.SetUnsafeIO(false)
}

func (s *SafeIoAtomicWriteTestSuite) TearDownSuite(c *C) {
	s.restoreUnsafeIO()
}
